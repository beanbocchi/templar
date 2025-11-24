package transport

import (
	"github.com/labstack/echo/v4"

	"github.com/beanbocchi/templar/internal/service"
)

type Handler struct {
	svc *service.Service
}

func SetupRoute(e *echo.Echo, svc *service.Service) {
	h := &Handler{svc: svc}
	api := e.Group("/api/v1")

	api.POST("/push", h.Push)
	api.POST("/pull", h.Pull)
	api.GET("/templates", h.ListTemplate)
	api.GET("/versions", h.ListVersions)
	api.GET("/jobs", h.ListJobs)
}
