package handler

import (
	"net/http"

	"services/generator-a/internal/generator"

	"github.com/labstack/echo/v4"
)

type Handlers struct {
	Loop *generator.Loop
}

func New(loop *generator.Loop) *Handlers {
	return &Handlers{Loop: loop}
}

// PUT /api/v1/frequency
// Body: {"hz": 5}  OR  {"period_ms": 200}
type setFreqReq struct {
	Hz       *float64 `json:"hz,omitempty"`
	PeriodMS *int64   `json:"period_ms,omitempty"`
}

func (h *Handlers) SetFrequency(c echo.Context) error {
	var req setFreqReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid json"})
	}
	if req.Hz != nil && *req.Hz > 0 {
		h.Loop.SetHz(*req.Hz)
		return c.NoContent(http.StatusNoContent)
	}
	if req.PeriodMS != nil && *req.PeriodMS > 0 {
		h.Loop.SetPeriodMS(*req.PeriodMS)
		return c.NoContent(http.StatusNoContent)
	}
	return c.JSON(http.StatusBadRequest, map[string]string{"error": "provide either hz>0 or period_ms>0"})
}

// Router builds a minimal Echo router with only the required endpoint.
func Router(h *Handlers) *echo.Echo {
	e := echo.New()
	api := e.Group("/api/v1")
	api.PUT("/frequency", h.SetFrequency)
	return e
}
