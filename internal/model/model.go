package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Role string

const (
	RoleUser  Role = "USER"
	RoleAdmin Role = "ADMIN"
)

type Category string

const (
	CategoryGeneral  Category = "GENERAL"
	CategoryQuestion Category = "QUESTION"
)

func (c Category) Valid() bool {
	return c == CategoryGeneral || c == CategoryQuestion
}

type User struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	Email        string         `gorm:"unique;size:320;not null" json:"email"`
	DisplayName  string         `gorm:"size:40;not null" json:"displayName"`
	PasswordHash string         `gorm:"not null" json:"-"`
	Role         Role           `gorm:"type:varchar(16);not null;default:USER;check:users_role_check,role IN ('USER','ADMIN')" json:"role"`
	Active       bool           `gorm:"not null;default:true" json:"active"`
	CreatedAt    time.Time      `gorm:"not null;default:now()" json:"createdAt"`
	UpdatedAt    time.Time      `gorm:"not null;default:now()" json:"updatedAt"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

func (u *User) BeforeCreate(_ *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

type Post struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	AuthorID  uuid.UUID      `gorm:"type:uuid;not null;index" json:"authorId"`
	Author    User           `gorm:"foreignKey:AuthorID;constraint:posts_author_id_fkey" json:"author"`
	Title     string         `gorm:"size:200;not null" json:"title"`
	Body      string         `gorm:"type:text;not null" json:"body"`
	Category  Category       `gorm:"type:varchar(16);not null;index;check:posts_category_check,category IN ('GENERAL','QUESTION')" json:"category"`
	Images    []PostImage    `gorm:"foreignKey:PostID;constraint:post_images_post_id_fkey,OnDelete:CASCADE" json:"images"`
	CreatedAt time.Time      `gorm:"not null;default:now();index:idx_posts_created_at,sort:desc" json:"createdAt"`
	UpdatedAt time.Time      `gorm:"not null;default:now()" json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (p *Post) BeforeCreate(_ *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

type PostImage struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	PostID    uuid.UUID `gorm:"type:uuid;not null;index;uniqueIndex:post_images_post_id_position_key" json:"-"`
	ObjectKey string    `gorm:"size:512;not null" json:"objectKey"`
	URL       string    `gorm:"size:1024;not null" json:"url"`
	Position  int       `gorm:"type:integer;not null;uniqueIndex:post_images_post_id_position_key" json:"position"`
}

func (i *PostImage) BeforeCreate(_ *gorm.DB) error {
	if i.ID == uuid.Nil {
		i.ID = uuid.New()
	}
	return nil
}

type Comment struct {
	Post      Post           `gorm:"foreignKey:PostID;constraint:comments_post_id_fkey" json:"-"`
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	PostID    uuid.UUID      `gorm:"type:uuid;not null;index" json:"postId"`
	AuthorID  uuid.UUID      `gorm:"type:uuid;not null;index" json:"authorId"`
	Author    User           `gorm:"foreignKey:AuthorID;constraint:comments_author_id_fkey" json:"author"`
	Body      string         `gorm:"type:text;not null" json:"body"`
	CreatedAt time.Time      `gorm:"not null;default:now()" json:"createdAt"`
	UpdatedAt time.Time      `gorm:"not null;default:now()" json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (c *Comment) BeforeCreate(_ *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}
