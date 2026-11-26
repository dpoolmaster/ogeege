package httpserver

import "github.com/gin-gonic/gin"

type errorEnvelope struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code      string            `json:"code"`
	Message   string            `json:"message"`
	Fields    map[string]string `json:"fields,omitempty"`
	RequestID string            `json:"request_id"`
}

// AbortWithError возвращает ошибку в едином формате API.
func AbortWithError(c *gin.Context, status int, code, message string) {
	AbortWithFields(c, status, code, message, nil)
}

// AbortWithFields возвращает ошибку с ошибками полей в едином формате API.
func AbortWithFields(c *gin.Context, status int, code, message string, fields map[string]string) {
	c.AbortWithStatusJSON(status, errorEnvelope{
		Error: errorBody{
			Code:      code,
			Message:   message,
			Fields:    fields,
			RequestID: RequestIDFromGin(c),
		},
	})
}
