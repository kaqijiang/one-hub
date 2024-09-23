package middleware

import (
	"fmt"
	"net/http"
	"one-api/common/logger"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

func RelayPanicRecover() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				errorResponse := gin.H{
					"error": gin.H{
						"message": fmt.Sprintf("Panic detected, error: %v. Please submit a issue here: https://bento.me/aijie", err),
						"type":    "omini_api_panic",
					},
				}
				handlePanic(c, err, errorResponse)
			}
		}()

		c.Next()
	}
}

func RelayCluadePanicRecover() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				errorResponse := gin.H{
					"type": "omini_api_panic",
					"error": gin.H{
						"type":    "omini_api_panic",
						"message": fmt.Sprintf("Panic detected, error: %v. Please submit a issue here: https://bento.me/aijie.", err),
					},
				}
				handlePanic(c, err, errorResponse)
			}
		}()
		c.Next()
	}
}

func RelayGeminiPanicRecover() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				errorResponse := gin.H{
					"error": gin.H{
						"code":    500,
						"status":  "omini_api_panic",
						"message": fmt.Sprintf("Panic detected, error: %v. Please submit a issue here: https://bento.me/aijie.", err),
					},
				}
				handlePanic(c, err, errorResponse)
			}
		}()
		c.Next()
	}
}

func RelayMJPanicRecover() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				errorResponse := gin.H{
					"error": gin.H{
						"description": fmt.Sprintf("Panic detected, error: %v. Please submit a issue here: https://bento.me/aijie.", err),
						"type":        "omini_api_panic",
						"code":        500,
					},
				}
				handlePanic(c, err, errorResponse)
			}
		}()

		c.Next()
	}
}

func RelaySunoPanicRecover() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				errorResponse := gin.H{
					"code":    "omini_api_panic",
					"message": fmt.Sprintf("Panic detected, error: %v. Please submit a issue here: https://bento.me/aijie.", err),
				}
				handlePanic(c, err, errorResponse)
			}
		}()
		c.Next()
	}
}

func handlePanic(c *gin.Context, err interface{}, errorResponse gin.H) {
	logger.SysError(fmt.Sprintf("panic detected: %v", err))
	logger.SysError(fmt.Sprintf("stacktrace from panic: %s", string(debug.Stack())))
	c.JSON(http.StatusInternalServerError, errorResponse)
	c.Abort()
}
