package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/m161awm2/velocity-bulletinBE/internal/auth"
	"github.com/m161awm2/velocity-bulletinBE/internal/model"
	"github.com/m161awm2/velocity-bulletinBE/internal/service"
	"github.com/m161awm2/velocity-bulletinBE/internal/store"
	"github.com/m161awm2/velocity-bulletinBE/internal/upload"
)

const actorKey = "actor"

type API struct {
	service *service.Service
	store   *store.Store
	tokens  *auth.Manager
	uploads *upload.Service
	logger  *slog.Logger
}

func New(svc *service.Service, st *store.Store, tokens *auth.Manager, uploads *upload.Service, logger *slog.Logger, corsOrigins []string) *gin.Engine {
	a := &API{service: svc, store: st, tokens: tokens, uploads: uploads, logger: logger}
	r := gin.New()
	r.Use(a.requestID(), a.accessLog(), gin.Recovery(), cors(corsOrigins), bodyLimit(1<<20))
	r.GET("/health/live", a.live)
	r.GET("/health/ready", a.ready)
	r.StaticFile("/openapi.yaml", "./api/openapi.yaml")

	v1 := r.Group("/api/v1")
	v1.POST("/auth/register", a.register)
	v1.POST("/auth/login", a.login)
	v1.GET("/posts", a.listPosts)
	v1.GET("/posts/:id", a.getPost)
	v1.GET("/posts/:id/comments", a.listComments)

	secured := v1.Group("")
	secured.Use(a.authenticate())
	secured.GET("/users/me", a.me)
	secured.PATCH("/users/me", a.updateMe)
	secured.DELETE("/users/me", a.deleteMe)
	secured.POST("/posts", a.createPost)
	secured.PUT("/posts/:id", a.updatePost)
	secured.DELETE("/posts/:id", a.deletePost)
	secured.POST("/posts/:id/likes/toggle", a.toggleLike)
	secured.POST("/posts/:id/comments", a.createComment)
	secured.PUT("/comments/:id", a.updateComment)
	secured.DELETE("/comments/:id", a.deleteComment)
	secured.POST("/uploads/presign", a.presignUpload)

	admin := secured.Group("/admin")
	admin.Use(a.requireAdmin())
	admin.GET("/users", a.listUsers)
	admin.PATCH("/users/:id/status", a.setUserStatus)
	return r
}

func actor(c *gin.Context) *model.User {
	value, _ := c.Get(actorKey)
	user, _ := value.(*model.User)
	return user
}

func parseID(c *gin.Context, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(name))
	if err != nil {
		fail(c, http.StatusBadRequest, "INVALID_ID", "invalid resource id")
		return uuid.Nil, false
	}
	return id, true
}

func pagination(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	if size > 100 {
		size = 100
	}
	return page, size
}

func fail(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, gin.H{"error": gin.H{"code": code, "message": message}, "requestId": c.GetString("requestId")})
}

func serviceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrInvalid):
		fail(c, http.StatusBadRequest, "INVALID_INPUT", "request validation failed")
	case errors.Is(err, service.ErrUnauthorized):
		fail(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid credentials")
	case errors.Is(err, service.ErrForbidden):
		fail(c, http.StatusForbidden, "FORBIDDEN", "permission denied")
	case errors.Is(err, service.ErrNotFound):
		fail(c, http.StatusNotFound, "NOT_FOUND", "resource not found")
	case errors.Is(err, service.ErrConflict):
		fail(c, http.StatusConflict, "CONFLICT", "resource already exists")
	default:
		slog.ErrorContext(c.Request.Context(), "request failed", "error", err, "requestId", c.GetString("requestId"))
		fail(c, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
	}
}

func bind(c *gin.Context, value any) bool {
	if err := c.ShouldBindJSON(value); err != nil {
		fail(c, http.StatusBadRequest, "INVALID_JSON", "invalid request body")
		return false
	}
	return true
}

func (a *API) live(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) }

func (a *API) ready(c *gin.Context) {
	ctx, cancel := contextWithTimeout(c, 2*time.Second)
	defer cancel()
	if err := a.store.Ready(ctx); err != nil {
		fail(c, http.StatusServiceUnavailable, "NOT_READY", "database is unavailable")
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}

func contextWithTimeout(c *gin.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(c.Request.Context(), timeout)
}

func (a *API) authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			fail(c, http.StatusUnauthorized, "UNAUTHORIZED", "bearer token required")
			return
		}
		id, err := a.tokens.Parse(parts[1])
		if err != nil {
			fail(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid token")
			return
		}
		user, err := a.service.User(c.Request.Context(), id)
		if err != nil || !user.Active {
			fail(c, http.StatusUnauthorized, "UNAUTHORIZED", "user is unavailable")
			return
		}
		c.Set(actorKey, user)
		c.Next()
	}
}

func (a *API) requireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if user := actor(c); user == nil || user.Role != model.RoleAdmin {
			fail(c, http.StatusForbidden, "FORBIDDEN", "administrator role required")
			return
		}
		c.Next()
	}
}
