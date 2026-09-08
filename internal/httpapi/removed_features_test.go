package httpapi

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http/httptest"
	"testing"

	"github.com/m161awm2/velocity-bulletinBE/internal/model"
)

func TestRemovedPostFeatures(t *testing.T) {
	router := New(nil, nil, nil, nil, slog.New(slog.NewTextHandler(io.Discard, nil)), nil)
	for _, tc := range []struct {
		method, path string
		status       int
	}{
		{"GET", "/api/v1/posts?sort=likes", 400},
		{"GET", "/api/v1/posts?sort=views", 400},
		{"GET", "/api/v1/posts?category=NOTICE", 400},
		{"POST", "/api/v1/posts/00000000-0000-0000-0000-000000000001/likes/toggle", 404},
	} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(tc.method, tc.path, nil))
		if recorder.Code != tc.status {
			t.Errorf("%s: got %d, want %d", tc.path, recorder.Code, tc.status)
		}
	}
	encoded, err := json.Marshal(presentPost(&model.Post{}))
	if err != nil {
		t.Fatal(err)
	}
	var response map[string]any
	if err := json.Unmarshal(encoded, &response); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"viewCount", "likeCount"} {
		if _, exists := response[field]; exists {
			t.Errorf("removed field %s still returned", field)
		}
	}
}
