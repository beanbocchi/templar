package transport

import (
	"github.com/beanbocchi/templar/internal/service"
	"github.com/beanbocchi/templar/pkg/binder"
	"github.com/beanbocchi/templar/pkg/validator"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// NewEcho creates a new Echo instance
func NewEcho(svc *service.Service) (*echo.Echo, error) {
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// Custom validator & binder
	customVal, err := validator.New()
	if err != nil {
		return nil, err
	}
	e.Validator = customVal
	e.Binder = binder.NewCustomBinder()

	// Health check
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(200, map[string]string{"status": "ok"})
	})

	// Setup routes
	SetupRoute(e, svc)

	return e, nil
}
