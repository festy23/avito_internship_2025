// Package router provides pullrequest module routes registration.
package router

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/festy23/avito_internship/internal/pullrequest/handler"
	"github.com/festy23/avito_internship/internal/pullrequest/repository"
	"github.com/festy23/avito_internship/internal/pullrequest/service"
)

// RegisterRoutes registers pullrequest module routes.
func RegisterRoutes(r *gin.Engine, db *gorm.DB, logger *zap.SugaredLogger) {
	repo := repository.New(db, logger)
	svc := service.New(repo, db, logger)
	h := handler.New(svc, logger)

	r.POST("/pullRequest/create", h.CreatePullRequest)
	r.POST("/pullRequest/merge", h.MergePullRequest)
	r.POST("/pullRequest/reassign", h.ReassignReviewer)
}
