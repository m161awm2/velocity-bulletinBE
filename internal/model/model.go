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
	Email        string         `gorm:"uniqueIndex;size:320;not null" json:"email"`
	DisplayName  string         `gorm:"size:40;not null" json:"displayName"`
	PasswordHash string         `gorm:"not null" json:"-"`
	Role         Role           `gorm:"type:varchar(16);not null;default:USER" json:"role"`
	Active       bool           `gorm:"not null;default:true" json:"active"`
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
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
	Author    User           `gorm:"foreignKey:AuthorID" json:"author"`
	Title     string         `gorm:"size:200;not null" json:"title"`
	Body      string         `gorm:"type:text;not null" json:"body"`
	Category  Category       `gorm:"type:varchar(16);not null;index" json:"category"`
	Images    []PostImage    `gorm:"foreignKey:PostID" json:"images"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
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
	PostID    uuid.UUID `gorm:"type:uuid;not null;index" json:"-"`
	ObjectKey string    `gorm:"size:512;not null" json:"objectKey"`
	URL       string    `gorm:"size:1024;not null" json:"url"`
	Position  int       `gorm:"not null" json:"position"`
}

func (i *PostImage) BeforeCreate(_ *gorm.DB) error {
	if i.ID == uuid.Nil {
		i.ID = uuid.New()
	}
	return nil
}

type Comment struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	PostID    uuid.UUID      `gorm:"type:uuid;not null;index" json:"postId"`
	AuthorID  uuid.UUID      `gorm:"type:uuid;not null;index" json:"authorId"`
	Author    User           `gorm:"foreignKey:AuthorID" json:"author"`
	Body      string         `gorm:"type:text;not null" json:"body"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (c *Comment) BeforeCreate(_ *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}
