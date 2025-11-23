// Package router provides team module routes registration.
package router

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/festy23/avito_internship/internal/team/handler"
	"github.com/festy23/avito_internship/internal/team/repository"
	"github.com/festy23/avito_internship/internal/team/service"
)

// RegisterRoutes registers team module routes.
func RegisterRoutes(r *gin.Engine, db *gorm.DB, logger *zap.SugaredLogger) {
	repo := repository.New(db, logger)
	svc := service.New(repo, db, logger)
	h := handler.New(svc, logger)

	r.POST("/team/add", h.AddTeam)
	r.GET("/team/get", h.GetTeam)
}
