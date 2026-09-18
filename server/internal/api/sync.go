package api

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"menguz/internal/badger"
	"menguz/internal/models"
	"menguz/internal/sae"
)

// adminTriggerSync runs the SAE sync on demand (protected by admin JWT).
func (s *Server) adminTriggerSync(c echo.Context) error {
	go func() {
		if _, err := sae.Sync(s.Cfg, s.Store, s.Ana); err != nil {
			// error is recorded in sync_log by Sync itself
			_ = err
		}
	}()
	return c.JSON(http.StatusAccepted, map[string]any{"started": true, "at": time.Now().Format(time.RFC3339)})
}

func (s *Server) adminSyncLogs(c echo.Context) error {
	out := []models.SyncLog{}
	var lg models.SyncLog
	err := s.Store.ScanPrefix(badger.PrefSyncLog, &lg, func(key string) bool {
		out = append(out, lg)
		return true
	})
	if err != nil {
		return err
	}
	for i := 1; i < len(out); i++ { // newest first
		for j := i; j > 0 && out[j].Inicio.After(out[j-1].Inicio); j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	if len(out) > 30 {
		out = out[:30]
	}
	return c.JSON(http.StatusOK, map[string]any{"items": out, "total": len(out)})
}
