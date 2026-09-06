package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type registerRequest struct {
	Email       string `json:"email" binding:"required"`
	DisplayName string `json:"displayName" binding:"required"`
	Password    string `json:"password" binding:"required"`
}

func (a *API) register(c *gin.Context) {
	var request registerRequest
	if !bind(c, &request) {
		return
	}
	user, err := a.service.Register(c.Request.Context(), request.Email, request.DisplayName, request.Password)
	if err != nil {
		serviceError(c, err)
		return
	}
	token, expiresAt, err := a.tokens.Issue(user)
	if err != nil {
		serviceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"user": user, "accessToken": token, "expiresAt": expiresAt})
}

type loginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (a *API) login(c *gin.Context) {
	var request loginRequest
	if !bind(c, &request) {
		return
	}
	user, err := a.service.Authenticate(c.Request.Context(), request.Email, request.Password)
	if err != nil {
		serviceError(c, err)
		return
	}
	token, expiresAt, err := a.tokens.Issue(user)
	if err != nil {
		serviceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": user, "accessToken": token, "expiresAt": expiresAt})
}

func (a *API) me(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"user": actor(c)}) }

func (a *API) updateMe(c *gin.Context) {
	var request struct {
		DisplayName string `json:"displayName" binding:"required"`
	}
	if !bind(c, &request) {
		return
	}
	user, err := a.service.UpdateProfile(c.Request.Context(), actor(c).ID, request.DisplayName)
	if err != nil {
		serviceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": user})
}

func (a *API) deleteMe(c *gin.Context) {
	if err := a.service.DeleteProfile(c.Request.Context(), actor(c).ID); err != nil {
		serviceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (a *API) listUsers(c *gin.Context) {
	page, size := pagination(c)
	users, total, err := a.service.ListUsers(c.Request.Context(), page, size)
	if err != nil {
		serviceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": users, "page": page, "size": size, "total": total})
}

func (a *API) setUserStatus(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	var request struct {
		Active *bool `json:"active" binding:"required"`
	}
	if !bind(c, &request) {
		return
	}
	if request.Active == nil {
		fail(c, http.StatusBadRequest, "INVALID_INPUT", "active is required")
		return
	}
	if err := a.service.SetUserActive(c.Request.Context(), actor(c), id, *request.Active); err != nil {
		serviceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
