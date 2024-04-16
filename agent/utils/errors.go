package utils

import (
	"github.com/gin-gonic/gin"
)

// Data structure for response  on HTTP calls
type HTTPResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// A function that return message and code on HTTP calls
func Throws(ctx *gin.Context, status int, msg string) {
	ctx.JSON(status, HTTPResponse{Code: status, Message: msg})
}
