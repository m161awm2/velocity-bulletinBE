package store

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/m161awm2/velocity-bulletinBE/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Store struct{ db *gorm.DB }

func New(db *gorm.DB) *Store { return &Store{db: db} }

func (s *Store) DB() *gorm.DB { return s.db }

func (s *Store) Ready(ctx context.Context) error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

func (s *Store) CreateUser(ctx context.Context, user *model.User) error {
	return s.db.WithContext(ctx).Create(user).Error
}

func (s *Store) UserByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	if err := s.db.WithContext(ctx).Where("email = ?", strings.ToLower(email)).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *Store) UserByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	var user model.User
	if err := s.db.WithContext(ctx).First(&user, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *Store) UpdateUserName(ctx context.Context, id uuid.UUID, name string) error {
	return s.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", id).Update("display_name", name).Error
}

func (s *Store) DeleteUser(ctx context.Context, id uuid.UUID) error {
	return s.db.WithContext(ctx).Delete(&model.User{}, "id = ?", id).Error
}

func (s *Store) SetUserActive(ctx context.Context, id uuid.UUID, active bool) error {
	result := s.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", id).Update("active", active)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (s *Store) ListUsers(ctx context.Context, page, size int) ([]model.User, int64, error) {
	var users []model.User
	var total int64
	q := s.db.WithContext(ctx).Model(&model.User{})
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Order("created_at DESC").Offset((page - 1) * size).Limit(size).Find(&users).Error; err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

type PostFilter struct {
	Search   string
	Category model.Category
	Sort     string
	Page     int
	Size     int
}

func (s *Store) CreatePost(ctx context.Context, post *model.Post) error {
	return s.db.WithContext(ctx).Create(post).Error
}

func (s *Store) PostByID(ctx context.Context, id uuid.UUID) (*model.Post, error) {
	var post model.Post
	err := s.db.WithContext(ctx).
		Preload("Author").Preload("Images", func(db *gorm.DB) *gorm.DB { return db.Order("position ASC") }).
		First(&post, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &post, nil
}

func (s *Store) IncrementViews(ctx context.Context, id uuid.UUID) error {
	return s.db.WithContext(ctx).Model(&model.Post{}).Where("id = ?", id).
		UpdateColumn("view_count", gorm.Expr("view_count + 1")).Error
}

func (s *Store) ListPosts(ctx context.Context, f PostFilter) ([]model.Post, int64, error) {
	q := s.db.WithContext(ctx).Model(&model.Post{})
	if f.Search != "" {
		pattern := "%" + strings.TrimSpace(f.Search) + "%"
		q = q.Where("title ILIKE ? OR body ILIKE ?", pattern, pattern)
	}
	if f.Category.Valid() {
		q = q.Where("category = ?", f.Category)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	order := "created_at DESC"
	switch f.Sort {
	case "views":
		order = "view_count DESC, created_at DESC"
	case "likes":
		order = "like_count DESC, created_at DESC"
	}
	var posts []model.Post
	err := q.Preload("Author").Preload("Images", func(db *gorm.DB) *gorm.DB { return db.Order("position ASC") }).
		Order(order).Offset((f.Page - 1) * f.Size).Limit(f.Size).Find(&posts).Error
	return posts, total, err
}

func (s *Store) UpdatePost(ctx context.Context, post *model.Post, replaceImages bool) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.Post{}).Where("id = ?", post.ID).Updates(map[string]any{
			"title": post.Title, "body": post.Body, "category": post.Category,
		}).Error; err != nil {
			return err
		}
		if !replaceImages {
			return nil
		}
		if err := tx.Where("post_id = ?", post.ID).Delete(&model.PostImage{}).Error; err != nil {
			return err
		}
		if len(post.Images) > 0 {
			return tx.Create(&post.Images).Error
		}
		return nil
	})
}

func (s *Store) DeletePost(ctx context.Context, id uuid.UUID) error {
	result := s.db.WithContext(ctx).Delete(&model.Post{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (s *Store) ToggleLike(ctx context.Context, userID, postID uuid.UUID) (bool, int64, error) {
	var liked bool
	var count int64
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var post model.Post
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&post, "id = ?", postID).Error; err != nil {
			return err
		}
		var like model.Like
		err := tx.First(&like, "user_id = ? AND post_id = ?", userID, postID).Error
		switch {
		case err == nil:
			if err := tx.Delete(&like).Error; err != nil {
				return err
			}
			if err := tx.Model(&post).UpdateColumn("like_count", gorm.Expr("GREATEST(like_count - 1, 0)")).Error; err != nil {
				return err
			}
			liked = false
		case errors.Is(err, gorm.ErrRecordNotFound):
			if err := tx.Create(&model.Like{UserID: userID, PostID: postID}).Error; err != nil {
				return err
			}
			if err := tx.Model(&post).UpdateColumn("like_count", gorm.Expr("like_count + 1")).Error; err != nil {
				return err
			}
			liked = true
		default:
			return err
		}
		return tx.Model(&model.Post{}).Select("like_count").First(&post, "id = ?", postID).Error
	})
	if err != nil {
		return false, 0, err
	}
	var post model.Post
	if err := s.db.WithContext(ctx).Select("like_count").First(&post, "id = ?", postID).Error; err != nil {
		return false, 0, err
	}
	count = post.LikeCount
	return liked, count, nil
}

func (s *Store) CreateComment(ctx context.Context, comment *model.Comment) error {
	return s.db.WithContext(ctx).Create(comment).Error
}

func (s *Store) CommentByID(ctx context.Context, id uuid.UUID) (*model.Comment, error) {
	var comment model.Comment
	if err := s.db.WithContext(ctx).Preload("Author").First(&comment, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &comment, nil
}

func (s *Store) ListComments(ctx context.Context, postID uuid.UUID, page, size int) ([]model.Comment, int64, error) {
	q := s.db.WithContext(ctx).Model(&model.Comment{}).Where("post_id = ?", postID)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var comments []model.Comment
	err := q.Preload("Author").Order("created_at ASC").Offset((page - 1) * size).Limit(size).Find(&comments).Error
	return comments, total, err
}

func (s *Store) UpdateComment(ctx context.Context, id uuid.UUID, body string) error {
	result := s.db.WithContext(ctx).Model(&model.Comment{}).Where("id = ?", id).Update("body", body)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (s *Store) DeleteComment(ctx context.Context, id uuid.UUID) error {
	result := s.db.WithContext(ctx).Delete(&model.Comment{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func IsNotFound(err error) bool { return errors.Is(err, gorm.ErrRecordNotFound) }

func Wrap(op string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", op, err)
}
