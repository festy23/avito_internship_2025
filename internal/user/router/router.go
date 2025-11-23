// Package router provides user module routes registration.
package router

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	pullrequestRepo "github.com/festy23/avito_internship/internal/pullrequest/repository"
	teamRepo "github.com/festy23/avito_internship/internal/team/repository"
	"github.com/festy23/avito_internship/internal/user/handler"
	"github.com/festy23/avito_internship/internal/user/repository"
	"github.com/festy23/avito_internship/internal/user/service"
)

// RegisterRoutes registers user module routes.
func RegisterRoutes(r *gin.Engine, db *gorm.DB, logger *zap.SugaredLogger) {
	repo := repository.New(db, logger)
	teamRepository := teamRepo.New(db, logger)
	pullrequestRepository := pullrequestRepo.New(db, logger)
	svc := service.NewWithDependencies(repo, teamRepository, pullrequestRepository, db, logger)
	h := handler.New(svc, logger)

	r.POST("/users/setIsActive", h.SetIsActive)
	r.GET("/users/getReview", h.GetReview)
	r.POST("/users/bulkDeactivate", h.BulkDeactivateTeamMembers)
}
