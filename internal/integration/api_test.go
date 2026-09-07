package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/m161awm2/velocity-bulletinBE/internal/auth"
	"github.com/m161awm2/velocity-bulletinBE/internal/config"
	"github.com/m161awm2/velocity-bulletinBE/internal/database"
	"github.com/m161awm2/velocity-bulletinBE/internal/httpapi"
	"github.com/m161awm2/velocity-bulletinBE/internal/model"
	"github.com/m161awm2/velocity-bulletinBE/internal/service"
	"github.com/m161awm2/velocity-bulletinBE/internal/store"
	"github.com/m161awm2/velocity-bulletinBE/internal/upload"
)

func TestRegisterPostCRUDAndAuthorization(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	if os.Getenv("ALLOW_INTEGRATION_DB_RESET") != "true" {
		t.Fatal("set ALLOW_INTEGRATION_DB_RESET=true to acknowledge destructive test cleanup")
	}
	cfg := config.Config{DatabaseURL: databaseURL, DBMaxOpenConns: 5, DBMaxIdleConns: 2, DBConnMaxLifetime: time.Minute}
	db, err := database.Open(context.Background(), cfg)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Post{}, &model.PostImage{}, &model.Comment{}, &model.Like{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	cleanup := func() {
		if err := db.Exec("TRUNCATE likes, comments, post_images, posts, users CASCADE").Error; err != nil {
			t.Errorf("cleanup: %v", err)
		}
	}
	cleanup()
	t.Cleanup(cleanup)
	st := store.New(db)
	uploads, err := upload.New(context.Background(), "ap-northeast-2", "", "")
	if err != nil {
		t.Fatal(err)
	}
	gin.SetMode(gin.TestMode)
	router := httpapi.New(service.New(st, ""), st, auth.New("01234567890123456789012345678901", time.Hour), uploads, slog.New(slog.NewJSONHandler(io.Discard, nil)), []string{"http://localhost:3000"})

	ownerToken := register(t, router, "owner@example.com", "Owner")
	otherToken := register(t, router, "other@example.com", "Other")

	created := request(t, router, http.MethodPost, "/api/v1/posts", ownerToken, map[string]any{
		"title": "First post", "body": "Hello from Gin", "category": "GENERAL",
	}, http.StatusCreated)
	post := object(t, created, "post")
	postID, _ := post["id"].(string)
	if postID == "" {
		t.Fatalf("missing post id in %#v", post)
	}
	author := object(t, post, "author")
	if _, exposed := author["email"]; exposed {
		t.Fatal("public author response exposed email")
	}

	request(t, router, http.MethodGet, "/api/v1/posts/"+postID, "", nil, http.StatusOK)
	request(t, router, http.MethodPut, "/api/v1/posts/"+postID, otherToken, map[string]any{
		"title": "Stolen", "body": "Must fail", "category": "GENERAL",
	}, http.StatusForbidden)
	request(t, router, http.MethodPost, "/api/v1/posts/"+postID+"/likes/toggle", otherToken, nil, http.StatusOK)
	request(t, router, http.MethodPost, "/api/v1/posts/"+postID+"/comments", otherToken, map[string]any{"body": "Nice post"}, http.StatusCreated)
	request(t, router, http.MethodDelete, "/api/v1/posts/"+postID, ownerToken, nil, http.StatusNoContent)
	request(t, router, http.MethodGet, "/api/v1/posts/"+postID, "", nil, http.StatusNotFound)
}

func register(t *testing.T, handler http.Handler, email, name string) string {
	t.Helper()
	response := request(t, handler, http.MethodPost, "/api/v1/auth/register", "", map[string]any{
		"email": email, "displayName": name, "password": "password123",
	}, http.StatusCreated)
	token, _ := response["accessToken"].(string)
	if token == "" {
		t.Fatalf("missing access token: %#v", response)
	}
	return token
}

func request(t *testing.T, handler http.Handler, method, path, token string, body any, wantStatus int) map[string]any {
	t.Helper()
	var raw []byte
	if body != nil {
		var err error
		raw, err = json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(raw))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	if recorder.Code != wantStatus {
		t.Fatalf("%s %s: got status %d, want %d; body=%s", method, path, recorder.Code, wantStatus, recorder.Body.String())
	}
	if recorder.Code == http.StatusNoContent || strings.TrimSpace(recorder.Body.String()) == "" {
		return nil
	}
	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return response
}

func object(t *testing.T, parent map[string]any, key string) map[string]any {
	t.Helper()
	value, ok := parent[key].(map[string]any)
	if !ok {
		t.Fatalf("%s is not an object in %#v", key, parent)
	}
	return value
}
