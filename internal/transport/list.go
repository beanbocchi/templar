package transport

import (
	"net/http"

	"github.com/guregu/null/v6"
	"github.com/labstack/echo/v4"

	"github.com/beanbocchi/templar/internal/model"
	"github.com/beanbocchi/templar/internal/service"
	"github.com/beanbocchi/templar/pkg/response"
)

type ListTemplateRequest struct {
	Search null.String `query:"search" validate:"omitempty,min=1"`
}

func (h *Handler) ListTemplate(c echo.Context) error {
	var req ListTemplateRequest
	if err := c.Bind(&req); err != nil {
		return response.FromError(c.Response().Writer, http.StatusBadRequest, err)
	}
	if err := c.Validate(&req); err != nil {
		return response.FromError(c.Response().Writer, http.StatusBadRequest, err)
	}
	templates, err := h.svc.ListTemplate(c.Request().Context(), service.ListTemplateParams{
		Search: req.Search,
	})
	if err != nil {
		return response.FromError(c.Response().Writer, http.StatusInternalServerError, err)
	}

	return response.FromDTO(c.Response().Writer, http.StatusOK, templates)
}

type ListVersionsRequest struct {
	TemplateID string `query:"template_id" validate:"required,uuid"`
}

func (h *Handler) ListVersions(c echo.Context) error {
	var req ListVersionsRequest
	if err := c.Bind(&req); err != nil {
		return response.FromError(c.Response().Writer, http.StatusBadRequest, err)
	}
	if err := c.Validate(&req); err != nil {
		return response.FromError(c.Response().Writer, http.StatusBadRequest, err)
	}

	versions, err := h.svc.ListVersions(c.Request().Context(), service.ListVersionsParams{
		TemplateID: req.TemplateID,
	})
	if err != nil {
		return response.FromError(c.Response().Writer, http.StatusInternalServerError, err)
	}
	return response.FromDTO(c.Response().Writer, http.StatusOK, versions)
}

type GetTemplateVersionRequest struct {
	TemplateID string `param:"template_id" validate:"required,uuid"`
	Version    int64  `param:"version" validate:"required,min=1"`
}

func (h *Handler) GetTemplateVersion(c echo.Context) error {
	var req GetTemplateVersionRequest
	if err := c.Bind(&req); err != nil {
		return response.FromError(c.Response().Writer, http.StatusBadRequest, err)
	}

	if err := c.Validate(&req); err != nil {
		return response.FromError(c.Response().Writer, http.StatusBadRequest, err)
	}

	version, err := h.svc.GetTemplateVersion(c.Request().Context(), service.GetTemplateVersionParams{
		TemplateID: req.TemplateID,
		Version:    req.Version,
	})
	if err != nil {
		return response.FromError(c.Response().Writer, http.StatusInternalServerError, err)
	}
	return response.FromDTO(c.Response().Writer, http.StatusOK, version)
}

type ListJobsRequest struct {
	model.PaginationParams
}

func (h *Handler) ListJobs(c echo.Context) error {
	var req ListJobsRequest
	if err := c.Bind(&req); err != nil {
		return response.FromError(c.Response().Writer, http.StatusBadRequest, err)
	}
	if err := c.Validate(&req); err != nil {
		return response.FromError(c.Response().Writer, http.StatusBadRequest, err)
	}
	jobs, err := h.svc.ListJobs(c.Request().Context(), service.ListJobsParams{
		PaginationParams: req.PaginationParams,
	})
	if err != nil {
		return response.FromError(c.Response().Writer, http.StatusInternalServerError, err)
	}
	return response.FromDTO(c.Response().Writer, http.StatusOK, jobs)
}
