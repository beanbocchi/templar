package transport

import (
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/beanbocchi/templar/internal/service"
	"github.com/beanbocchi/templar/pkg/response"
)

type PullRequest struct {
	TemplateID uuid.UUID `json:"template_id" validate:"required,uuid"`
	Version    int64     `json:"version" validate:"required,min=1"`
}

func (h *Handler) Pull(c echo.Context) error {
	var req PullRequest
	if err := c.Bind(&req); err != nil {
		return response.FromError(c.Response().Writer, http.StatusBadRequest, err)
	}

	if err := c.Validate(&req); err != nil {
		return response.FromError(c.Response().Writer, http.StatusBadRequest, err)
	}

	reader, err := h.svc.Pull(c.Request().Context(), service.PullParams{
		TemplateID: req.TemplateID,
		Version:    req.Version,
	})
	if err != nil {
		return response.FromError(c.Response().Writer, http.StatusInternalServerError, err)
	}
	defer reader.Close()

	c.Response().Header().Set(echo.HeaderContentType, "application/octet-stream")
	c.Response().Header().Set(echo.HeaderContentDisposition, fmt.Sprintf("attachment; filename=template_%s_%d", req.TemplateID.String(), req.Version))
	c.Response().WriteHeader(http.StatusOK)

	if _, err := io.Copy(c.Response().Writer, reader); err != nil {
		return response.FromError(c.Response().Writer, http.StatusInternalServerError, err)
	}

	return nil
}
