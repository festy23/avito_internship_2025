// Package router provides team module routes registration.
package router

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/festy23/avito_internship/internal/team/handler"
	"github.com/festy23/avito_internship/internal/team/repository"
	"github.com/festy23/avito_internship/internal/team/service"
)

// RegisterRoutes registers team module routes.
func RegisterRoutes(r *gin.Engine, db *gorm.DB) {
	repo := repository.New(db)
	svc := service.New(repo, db)
	h := handler.New(svc)

	r.POST("/team/add", h.AddTeam)
	r.GET("/team/get", h.GetTeam)
}

