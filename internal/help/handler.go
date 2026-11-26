package help

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"oge-ege/internal/auth"
	"oge-ege/internal/shared/httpserver"
)

// Handler обслуживает маршруты справки.
type Handler struct {
	service *Service
}

// NewHandler создаёт HTTP-обработчик справки.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Register регистрирует маршрут получения справки раздела.
func (h *Handler) Register(group *gin.RouterGroup) {
	group.GET("/help/:id", h.get)
}

func (h *Handler) get(c *gin.Context) {
	user, ok := auth.CurrentUser(c)
	if !ok {
		httpserver.AbortWithError(c, http.StatusUnauthorized, "unauthorized", "Требуется вход в систему")
		return
	}

	document, err := h.service.Get(c.Request.Context(), c.Param("id"), user.Role)
	switch {
	case errors.Is(err, ErrNotFound):
		httpserver.AbortWithError(c, http.StatusNotFound, "not_found", "Справка не найдена")
		return
	case errors.Is(err, ErrForbidden):
		httpserver.AbortWithError(c, http.StatusForbidden, "forbidden", "Недостаточно прав")
		return
	case err != nil:
		httpserver.AbortWithError(c, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка")
		return
	}

	c.JSON(http.StatusOK, gin.H{"title": document.Title, "html": document.HTML})
}
