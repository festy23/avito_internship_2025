// Package router provides statistics module routes registration.
package router

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/festy23/avito_internship/internal/statistics/handler"
	"github.com/festy23/avito_internship/internal/statistics/repository"
	"github.com/festy23/avito_internship/internal/statistics/service"
)

// RegisterRoutes registers statistics module routes.
func RegisterRoutes(r *gin.Engine, db *gorm.DB, logger *zap.SugaredLogger) {
	repo := repository.New(db, logger)
	svc := service.New(repo, logger)
	h := handler.New(svc, logger)

	r.GET("/statistics/reviewers", h.GetReviewersStatistics)
	r.GET("/statistics/pullrequests", h.GetPullRequestStatistics)
}
