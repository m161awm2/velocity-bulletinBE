package service

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/m161awm2/velocity-bulletinBE/internal/model"
	"github.com/m161awm2/velocity-bulletinBE/internal/store"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrNotFound     = errors.New("resource not found")
	ErrConflict     = errors.New("resource already exists")
	ErrUnauthorized = errors.New("authentication failed")
	ErrForbidden    = errors.New("permission denied")
	ErrInvalid      = errors.New("invalid input")
)

type Service struct {
	store        *store.Store
	imageBaseURL string
}

func New(st *store.Store, imageBaseURL string) *Service {
	return &Service{store: st, imageBaseURL: strings.TrimRight(imageBaseURL, "/")}
}

func (s *Service) Register(ctx context.Context, email, displayName, password string) (*model.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	displayName = strings.TrimSpace(displayName)
	if _, err := mail.ParseAddress(email); err != nil || len(displayName) < 2 || len(displayName) > 40 || len(password) < 8 {
		return nil, ErrInvalid
	}
	if _, err := s.store.UserByEmail(ctx, email); err == nil {
		return nil, ErrConflict
	} else if !store.IsNotFound(err) {
		return nil, fmt.Errorf("check user: %w", err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	user := &model.User{Email: email, DisplayName: displayName, PasswordHash: string(hash), Role: model.RoleUser, Active: true}
	if err := s.store.CreateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return user, nil
}

func (s *Service) Authenticate(ctx context.Context, email, password string) (*model.User, error) {
	user, err := s.store.UserByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if err != nil || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return nil, ErrUnauthorized
	}
	if !user.Active {
		return nil, ErrForbidden
	}
	return user, nil
}

func (s *Service) User(ctx context.Context, id uuid.UUID) (*model.User, error) {
	user, err := s.store.UserByID(ctx, id)
	if store.IsNotFound(err) {
		return nil, ErrNotFound
	}
	return user, err
}

func (s *Service) UpdateProfile(ctx context.Context, id uuid.UUID, displayName string) (*model.User, error) {
	displayName = strings.TrimSpace(displayName)
	if len(displayName) < 2 || len(displayName) > 40 {
		return nil, ErrInvalid
	}
	if err := s.store.UpdateUserName(ctx, id, displayName); err != nil {
		return nil, err
	}
	return s.User(ctx, id)
}

func (s *Service) DeleteProfile(ctx context.Context, id uuid.UUID) error {
	return s.store.DeleteUser(ctx, id)
}

type ImageInput struct {
	ObjectKey string `json:"objectKey"`
}

var objectKeyPattern = regexp.MustCompile(`^posts/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\.(jpg|png|webp)$`)

func (s *Service) publicURL(objectKey string) string {
	if s.imageBaseURL == "" {
		return objectKey
	}
	return s.imageBaseURL + "/" + objectKey
}

func (s *Service) images(postID uuid.UUID, input []ImageInput) ([]model.PostImage, error) {
	if len(input) > 5 {
		return nil, ErrInvalid
	}
	result := make([]model.PostImage, 0, len(input))
	for position, image := range input {
		objectKey := strings.TrimSpace(image.ObjectKey)
		if !objectKeyPattern.MatchString(objectKey) {
			return nil, ErrInvalid
		}
		result = append(result, model.PostImage{PostID: postID, ObjectKey: objectKey, URL: s.publicURL(objectKey), Position: position})
	}
	return result, nil
}

func validatePost(title, body string, category model.Category) error {
	if len(strings.TrimSpace(title)) < 2 || len(title) > 200 || len(strings.TrimSpace(body)) < 1 || !category.Valid() {
		return ErrInvalid
	}

	return nil
}

func (s *Service) CreatePost(ctx context.Context, actor *model.User, title, body string, category model.Category, imageInput []ImageInput) (*model.Post, error) {
	if err := validatePost(title, body, category); err != nil {
		return nil, err
	}
	post := &model.Post{ID: uuid.New(), AuthorID: actor.ID, Title: strings.TrimSpace(title), Body: strings.TrimSpace(body), Category: category}
	var err error
	post.Images, err = s.images(post.ID, imageInput)
	if err != nil {
		return nil, err
	}
	if err := s.store.CreatePost(ctx, post); err != nil {
		return nil, err
	}
	return s.store.PostByID(ctx, post.ID)
}

func (s *Service) GetPost(ctx context.Context, id uuid.UUID) (*model.Post, error) {
	post, err := s.store.PostByID(ctx, id)
	if store.IsNotFound(err) {
		return nil, ErrNotFound
	}
	return post, err
}

func (s *Service) ListPosts(ctx context.Context, filter store.PostFilter) ([]model.Post, int64, error) {
	return s.store.ListPosts(ctx, filter)
}

func canModify(owner uuid.UUID, actor *model.User) bool {
	return owner == actor.ID || actor.Role == model.RoleAdmin
}

func (s *Service) UpdatePost(ctx context.Context, actor *model.User, id uuid.UUID, title, body string, category model.Category, imageInput *[]ImageInput) (*model.Post, error) {
	post, err := s.store.PostByID(ctx, id)
	if store.IsNotFound(err) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if !canModify(post.AuthorID, actor) {
		return nil, ErrForbidden
	}
	if err := validatePost(title, body, category); err != nil {
		return nil, err
	}
	post.Title, post.Body, post.Category = strings.TrimSpace(title), strings.TrimSpace(body), category
	if imageInput != nil {
		post.Images, err = s.images(post.ID, *imageInput)
		if err != nil {
			return nil, err
		}
	}
	if err := s.store.UpdatePost(ctx, post, imageInput != nil); err != nil {
		return nil, err
	}
	return s.store.PostByID(ctx, id)
}

func (s *Service) DeletePost(ctx context.Context, actor *model.User, id uuid.UUID) error {
	post, err := s.store.PostByID(ctx, id)
	if store.IsNotFound(err) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if !canModify(post.AuthorID, actor) {
		return ErrForbidden
	}
	return s.store.DeletePost(ctx, id)
}

func (s *Service) CreateComment(ctx context.Context, actor *model.User, postID uuid.UUID, body string) (*model.Comment, error) {
	body = strings.TrimSpace(body)
	if body == "" || len(body) > 2000 {
		return nil, ErrInvalid
	}
	if _, err := s.store.PostByID(ctx, postID); err != nil {
		if store.IsNotFound(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	comment := &model.Comment{PostID: postID, AuthorID: actor.ID, Body: body}
	if err := s.store.CreateComment(ctx, comment); err != nil {
		return nil, err
	}
	return s.store.CommentByID(ctx, comment.ID)
}

func (s *Service) ListComments(ctx context.Context, postID uuid.UUID, page, size int) ([]model.Comment, int64, error) {
	if _, err := s.store.PostByID(ctx, postID); err != nil {
		if store.IsNotFound(err) {
			return nil, 0, ErrNotFound
		}
		return nil, 0, err
	}
	return s.store.ListComments(ctx, postID, page, size)
}

func (s *Service) UpdateComment(ctx context.Context, actor *model.User, id uuid.UUID, body string) (*model.Comment, error) {
	comment, err := s.store.CommentByID(ctx, id)
	if store.IsNotFound(err) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if !canModify(comment.AuthorID, actor) {
		return nil, ErrForbidden
	}
	body = strings.TrimSpace(body)
	if body == "" || len(body) > 2000 {
		return nil, ErrInvalid
	}
	if err := s.store.UpdateComment(ctx, id, body); err != nil {
		return nil, err
	}
	return s.store.CommentByID(ctx, id)
}

func (s *Service) DeleteComment(ctx context.Context, actor *model.User, id uuid.UUID) error {
	comment, err := s.store.CommentByID(ctx, id)
	if store.IsNotFound(err) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if !canModify(comment.AuthorID, actor) {
		return ErrForbidden
	}
	return s.store.DeleteComment(ctx, id)
}

func (s *Service) ListUsers(ctx context.Context, page, size int) ([]model.User, int64, error) {
	return s.store.ListUsers(ctx, page, size)
}

func (s *Service) SetUserActive(ctx context.Context, actor *model.User, id uuid.UUID, active bool) error {
	if actor.ID == id && !active {
		return ErrInvalid
	}
	if err := s.store.SetUserActive(ctx, id, active); store.IsNotFound(err) {
		return ErrNotFound
	} else {
		return err
	}
}

func (s *Service) SeedAdmin(ctx context.Context, email, displayName, password string) (*model.User, error) {
	if email == "" || password == "" {
		return nil, ErrInvalid
	}
	if existing, err := s.store.UserByEmail(ctx, email); err == nil {
		if existing.Role != model.RoleAdmin {
			return nil, ErrConflict
		}
		return existing, nil
	} else if !store.IsNotFound(err) {
		return nil, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	admin := &model.User{Email: strings.ToLower(email), DisplayName: displayName, PasswordHash: string(hash), Role: model.RoleAdmin, Active: true}
	if err := s.store.CreateUser(ctx, admin); err != nil {
		return nil, err
	}
	return admin, nil
}
