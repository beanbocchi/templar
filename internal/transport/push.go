package transport

import (
	"net/http"

	"github.com/beanbocchi/templar/internal/service"
	"github.com/beanbocchi/templar/pkg/response"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type PushRequest struct {
	TemplateID uuid.UUID `form:"template_id" validate:"required,uuid"`
	Version    int64     `form:"version" validate:"required,min=1"`
}

func (h *Handler) Push(c echo.Context) error {
	var req PushRequest
	if err := c.Bind(&req); err != nil {
		return response.FromError(c.Response().Writer, http.StatusBadRequest, err)
	}

	if err := c.Validate(&req); err != nil {
		return response.FromError(c.Response().Writer, http.StatusBadRequest, err)
	}

	file, err := c.FormFile("file")
	if err != nil {
		return response.FromError(c.Response().Writer, http.StatusBadRequest, err)
	}

	if err := h.svc.Push(c.Request().Context(), service.PushParams{
		TemplateID: req.TemplateID,
		Version:    req.Version,
		File:       file,
	}); err != nil {
		return response.FromError(c.Response().Writer, http.StatusInternalServerError, err)
	}

	return response.FromMessage(c.Response().Writer, http.StatusOK, "Template pushed, will be available in a few seconds")
}
