package http

import (
	"net/http"
	"strconv"
	"time"

	"services/ingest-b/internal/auth"
	"services/ingest-b/internal/repo"

	"github.com/labstack/echo/v4"
)

type Handlers struct {
	Store *repo.Store
}

func New(store *repo.Store) *Handlers { return &Handlers{Store: store} }

func parseFilter(c echo.Context) (repo.Filter, error) {
	var f repo.Filter
	if v := c.QueryParam("id1"); v != "" {
		f.ID1 = &v
	}
	if v := c.QueryParam("id2"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			f.ID2 = &i
		}
	}
	if v := c.QueryParam("sensor_type"); v != "" {
		f.SensorType = &v
	}
	parseTs := func(s string) *time.Time {
		if s == "" {
			return nil
		}
		t, err := time.Parse(time.RFC3339Nano, s)
		if err != nil {
			return nil
		}
		return &t
	}
	f.From = parseTs(c.QueryParam("from"))
	f.To = parseTs(c.QueryParam("to"))
	return f, nil
}

// GET /api/v1/readings? id1=A&id2=1&sensor_type=temperature&from=...&to=...&page=1&limit=50
func (h *Handlers) GetReadings(c echo.Context) error {
	f, _ := parseFilter(c)
	page, _ := strconv.Atoi(c.QueryParam("page"))
	limit, _ := strconv.Atoi(c.QueryParam("limit"))

	total, rows, err := h.Store.Query(c.Request().Context(), f, page, limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "query failed"})
	}
	c.Response().Header().Set("X-Total-Count", strconv.FormatInt(total, 10))
	return c.JSON(http.StatusOK, rows)
}

type editReq struct {
	Filter struct {
		ID1        *string `json:"id1"`
		ID2        *int    `json:"id2"`
		SensorType *string `json:"sensor_type"`
		From       *string `json:"from"` // RFC3339
		To         *string `json:"to"`   // RFC3339
	} `json:"filter"`
	Set struct {
		Value *float64 `json:"value"` // update only value
	} `json:"set"`
}

func parseTimePtr(s *string) *time.Time {
	if s == nil || *s == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339Nano, *s)
	if err != nil {
		return nil
	}
	return &t
}

// PATCH /api/v1/readings  (admin only)
func (h *Handlers) EditReadings(c echo.Context) error {
	var req editReq
	if err := c.Bind(&req); err != nil || req.Set.Value == nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid json"})
	}
	f := repo.Filter{
		ID1:        req.Filter.ID1,
		ID2:        req.Filter.ID2,
		SensorType: req.Filter.SensorType,
		From:       parseTimePtr(req.Filter.From),
		To:         parseTimePtr(req.Filter.To),
	}
	affected, err := h.Store.Edit(c.Request().Context(), f, *req.Set.Value)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "edit failed"})
	}
	return c.JSON(http.StatusOK, map[string]any{"updated": affected})
}

// DELETE /api/v1/readings  (admin only)
type deleteReq struct {
	ID1        *string `json:"id1"`
	ID2        *int    `json:"id2"`
	SensorType *string `json:"sensor_type"`
	From       *string `json:"from"`
	To         *string `json:"to"`
}

func (h *Handlers) DeleteReadings(c echo.Context) error {
	var req deleteReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid json"})
	}
	f := repo.Filter{
		ID1:        req.ID1,
		ID2:        req.ID2,
		SensorType: req.SensorType,
		From:       parseTimePtr(req.From),
		To:         parseTimePtr(req.To),
	}
	affected, err := h.Store.Delete(c.Request().Context(), f)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "delete failed"})
	}
	return c.JSON(http.StatusOK, map[string]any{"deleted": affected})
}

func Router(h *Handlers) *echo.Echo {
	e := echo.New()

	// Auth
	e.POST("/api/v1/auth/login", auth.LoginHandler)

	// Read-only (viewer or admin)
	view := e.Group("/api/v1", auth.RequireRole("viewer", "admin"))
	view.GET("/readings", h.GetReadings)

	// Mutations (admin only)
	admin := e.Group("/api/v1", auth.RequireRole("admin"))
	admin.PATCH("/readings", h.EditReadings)
	admin.DELETE("/readings", h.DeleteReadings)

	return e
}
