package httpserver

import "github.com/gin-gonic/gin"

type errorEnvelope struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}

// AbortWithError возвращает ошибку в едином формате API.
func AbortWithError(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, errorEnvelope{
		Error: errorBody{
			Code:      code,
			Message:   message,
			RequestID: RequestIDFromGin(c),
		},
	})
}
