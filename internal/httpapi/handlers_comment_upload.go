package httpapi

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/m161awm2/velocity-bulletinBE/internal/upload"
)

func (a *API) listComments(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	page, size := pagination(c)
	comments, total, err := a.service.ListComments(c.Request.Context(), id, page, size)
	if err != nil {
		serviceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": presentComments(comments), "page": page, "size": size, "total": total})
}

func (a *API) createComment(c *gin.Context) {
	postID, ok := parseID(c, "id")
	if !ok {
		return
	}
	var request struct {
		Body string `json:"body" binding:"required"`
	}
	if !bind(c, &request) {
		return
	}
	comment, err := a.service.CreateComment(c.Request.Context(), actor(c), postID, request.Body)
	if err != nil {
		serviceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"comment": presentComment(comment)})
}

func (a *API) updateComment(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	var request struct {
		Body string `json:"body" binding:"required"`
	}
	if !bind(c, &request) {
		return
	}
	comment, err := a.service.UpdateComment(c.Request.Context(), actor(c), id, request.Body)
	if err != nil {
		serviceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"comment": presentComment(comment)})
}

func (a *API) deleteComment(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	if err := a.service.DeleteComment(c.Request.Context(), actor(c), id); err != nil {
		serviceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (a *API) presignUpload(c *gin.Context) {
	var request struct {
		Filename    string `json:"filename" binding:"required"`
		ContentType string `json:"contentType" binding:"required"`
		Size        int64  `json:"size" binding:"required"`
	}
	if !bind(c, &request) {
		return
	}
	result, err := a.uploads.Presign(c.Request.Context(), request.Filename, request.ContentType, request.Size)
	if errors.Is(err, upload.ErrDisabled) {
		fail(c, http.StatusServiceUnavailable, "UPLOADS_DISABLED", err.Error())
		return
	}
	if err != nil {
		fail(c, http.StatusBadRequest, "INVALID_UPLOAD", err.Error())
		return
	}
	c.JSON(http.StatusOK, result)
}
