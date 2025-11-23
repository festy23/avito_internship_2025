// Package router provides user module routes registration.
package router

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/festy23/avito_internship/internal/user/handler"
	"github.com/festy23/avito_internship/internal/user/repository"
	"github.com/festy23/avito_internship/internal/user/service"
)

// RegisterRoutes registers user module routes.
func RegisterRoutes(r *gin.Engine, db *gorm.DB, logger *zap.SugaredLogger) {
	repo := repository.New(db, logger)
	svc := service.New(repo, logger)
	h := handler.New(svc, logger)

	r.POST("/users/setIsActive", h.SetIsActive)
	r.GET("/users/getReview", h.GetReview)
}
