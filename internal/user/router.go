package user

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/festy23/avito_internship/internal/user/handler"
	"github.com/festy23/avito_internship/internal/user/repository"
	"github.com/festy23/avito_internship/internal/user/service"
)

// RegisterRoutes registers user module routes.
func RegisterRoutes(r *gin.Engine, db *gorm.DB) {
	repo := repository.New(db)
	svc := service.New(repo)
	h := handler.New(svc)

	r.POST("/users/setIsActive", h.SetIsActive)
	r.GET("/users/getReview", h.GetReview)
}
