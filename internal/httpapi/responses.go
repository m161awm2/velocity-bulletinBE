package httpapi

import (
	"time"

	"github.com/google/uuid"
	"github.com/m161awm2/velocity-bulletinBE/internal/model"
)

type publicAuthor struct {
	ID          uuid.UUID `json:"id"`
	DisplayName string    `json:"displayName"`
}

type postResponse struct {
	ID        uuid.UUID         `json:"id"`
	AuthorID  uuid.UUID         `json:"authorId"`
	Author    publicAuthor      `json:"author"`
	Title     string            `json:"title"`
	Body      string            `json:"body"`
	Category  model.Category    `json:"category"`
	Images    []model.PostImage `json:"images"`
	CreatedAt time.Time         `json:"createdAt"`
	UpdatedAt time.Time         `json:"updatedAt"`
}

type commentResponse struct {
	ID        uuid.UUID    `json:"id"`
	PostID    uuid.UUID    `json:"postId"`
	AuthorID  uuid.UUID    `json:"authorId"`
	Author    publicAuthor `json:"author"`
	Body      string       `json:"body"`
	CreatedAt time.Time    `json:"createdAt"`
	UpdatedAt time.Time    `json:"updatedAt"`
}

func presentPost(post *model.Post) postResponse {
	return postResponse{
		ID: post.ID, AuthorID: post.AuthorID,
		Author: publicAuthor{ID: post.Author.ID, DisplayName: post.Author.DisplayName},
		Title:  post.Title, Body: post.Body, Category: post.Category,
		Images:    post.Images,
		CreatedAt: post.CreatedAt, UpdatedAt: post.UpdatedAt,
	}
}

func presentPosts(posts []model.Post) []postResponse {
	result := make([]postResponse, 0, len(posts))
	for i := range posts {
		result = append(result, presentPost(&posts[i]))
	}
	return result
}

func presentComment(comment *model.Comment) commentResponse {
	return commentResponse{
		ID: comment.ID, PostID: comment.PostID, AuthorID: comment.AuthorID,
		Author: publicAuthor{ID: comment.Author.ID, DisplayName: comment.Author.DisplayName},
		Body:   comment.Body, CreatedAt: comment.CreatedAt, UpdatedAt: comment.UpdatedAt,
	}
}

func presentComments(comments []model.Comment) []commentResponse {
	result := make([]commentResponse, 0, len(comments))
	for i := range comments {
		result = append(result, presentComment(&comments[i]))
	}
	return result
}
