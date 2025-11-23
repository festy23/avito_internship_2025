// Package handler provides response helpers for statistics module.
package handler

import (
	"github.com/gin-gonic/gin"
)

// ErrorResponse represents an error response.
type ErrorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// errorResponse sends an error response.
func errorResponse(c *gin.Context, code, message string, status int) {
	c.JSON(status, ErrorResponse{
		Error: struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		}{
			Code:    code,
			Message: message,
		},
	})
}
