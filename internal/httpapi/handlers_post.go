package httpapi

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/m161awm2/velocity-bulletinBE/internal/model"
	"github.com/m161awm2/velocity-bulletinBE/internal/service"
	"github.com/m161awm2/velocity-bulletinBE/internal/store"
)

type postRequest struct {
	Title    string               `json:"title" binding:"required"`
	Body     string               `json:"body" binding:"required"`
	Category model.Category       `json:"category" binding:"required"`
	Images   []service.ImageInput `json:"images"`
}

type updatePostRequest struct {
	Title    string                `json:"title" binding:"required"`
	Body     string                `json:"body" binding:"required"`
	Category model.Category        `json:"category" binding:"required"`
	Images   *[]service.ImageInput `json:"images"`
}

func (a *API) createPost(c *gin.Context) {
	var request postRequest
	if !bind(c, &request) {
		return
	}
	post, err := a.service.CreatePost(c.Request.Context(), actor(c), request.Title, request.Body, request.Category, request.Images)
	if err != nil {
		serviceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"post": presentPost(post)})
}

func (a *API) getPost(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	post, err := a.service.GetPost(c.Request.Context(), id)
	if err != nil {
		serviceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"post": presentPost(post)})
}

func (a *API) listPosts(c *gin.Context) {
	page, size := pagination(c)
	category := model.Category(strings.ToUpper(c.Query("category")))
	if category != "" && !category.Valid() {
		fail(c, http.StatusBadRequest, "INVALID_CATEGORY", "invalid category")
		return
	}
	sort := c.DefaultQuery("sort", "latest")
	if sort != "latest" {
		fail(c, http.StatusBadRequest, "INVALID_SORT", "only latest sorting is supported")
		return
	}
	posts, total, err := a.service.ListPosts(c.Request.Context(), store.PostFilter{Search: c.Query("search"), Category: category, Page: page, Size: size})
	if err != nil {
		serviceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": presentPosts(posts), "page": page, "size": size, "total": total})
}

func (a *API) updatePost(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	var request updatePostRequest
	if !bind(c, &request) {
		return
	}
	post, err := a.service.UpdatePost(c.Request.Context(), actor(c), id, request.Title, request.Body, request.Category, request.Images)
	if err != nil {
		serviceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"post": presentPost(post)})
}

func (a *API) deletePost(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	if err := a.service.DeletePost(c.Request.Context(), actor(c), id); err != nil {
		serviceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
