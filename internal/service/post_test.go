package service

import (
	"errors"
	"testing"

	"github.com/m161awm2/velocity-bulletinBE/internal/model"
)

func TestPostCategories(t *testing.T) {
	for _, category := range []model.Category{model.CategoryGeneral, model.CategoryQuestion} {
		if err := validatePost("Title", "Body", category); err != nil {
			t.Fatal(err)
		}
	}
	if err := validatePost("Title", "Body", model.Category("NOTICE")); !errors.Is(err, ErrInvalid) {
		t.Fatalf("NOTICE must be rejected: %v", err)
	}
}
