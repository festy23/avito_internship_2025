package handler

import (
	"github.com/gin-gonic/gin"
)

// ErrorResponse represents error response structure matching OpenAPI spec.
type ErrorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// errorResponse creates error response matching OpenAPI spec.
func errorResponse(c *gin.Context, code string, message string, statusCode int) {
	resp := ErrorResponse{}
	resp.Error.Code = code
	resp.Error.Message = message
	c.JSON(statusCode, resp)
}

// notFoundResponse creates 404 error response.
func notFoundResponse(c *gin.Context, message string) {
	errorResponse(c, "NOT_FOUND", message, 404)
}
