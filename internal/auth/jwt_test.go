package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/m161awm2/velocity-bulletinBE/internal/model"
)

func TestIssueAndParse(t *testing.T) {
	user := &model.User{ID: uuid.New(), Role: model.RoleUser}
	manager := New("01234567890123456789012345678901", time.Hour)
	raw, expiresAt, err := manager.Issue(user)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	if time.Until(expiresAt) < 59*time.Minute {
		t.Fatalf("unexpected expiry: %v", expiresAt)
	}
	id, err := manager.Parse(raw)
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}
	if id != user.ID {
		t.Fatalf("got user %s, want %s", id, user.ID)
	}
}

func TestRejectsWrongSecret(t *testing.T) {
	user := &model.User{ID: uuid.New(), Role: model.RoleUser}
	raw, _, err := New("01234567890123456789012345678901", time.Hour).Issue(user)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := New("abcdefghijklmnopqrstuvwxyz123456", time.Hour).Parse(raw); err == nil {
		t.Fatal("expected token signed by a different secret to be rejected")
	}
}
