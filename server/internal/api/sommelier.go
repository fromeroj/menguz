package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"menguz/internal/badger"
	"menguz/internal/chatbot"
	"menguz/internal/models"
	"menguz/internal/sommelier"
)

// sommelierChatHandler powers the storefront "Sommelier" advisor: one user
// message in, a streamed reply plus recommended-product cards out (SSE).
//
// POST /api/sommelier/chat
//
//	{ "message": "...", "session_id": "optional" }
//
// Events: session → delta* → products → done
func (s *Server) sommelierChatHandler(c echo.Context) error {
	if !s.ChatLim.Allow(c.RealIP()) {
		return echo.NewHTTPError(http.StatusTooManyRequests, "demasiados mensajes, espera un momento")
	}
	var req struct {
		Message   string `json:"message" validate:"required,min=1,max=2000"`
		SessionID string `json:"session_id"`
	}
	if err := c.Bind(&req); err != nil {
		return badRequest("JSON inválido")
	}

	// session (shared chat_sess store, canal=sommelier → visible in admin chat review)
	sid := req.SessionID
	if sid == "" {
		sid = newID("somm")
	}
	var sess models.ChatSession
	if err := s.Store.GetJSON(badger.PrefChatSess+sid, &sess); err != nil {
		sess = models.ChatSession{
			ID: sid, Canal: "sommelier", Messages: []models.ChatMessage{},
			CreatedAt: nowUTC(), UpdatedAt: nowUTC(),
		}
	}
	chatbot.AppendMessage(&sess, "user", req.Message)

	w := c.Response().Writer
	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("X-Accel-Buffering", "no")
	fl, ok := w.(http.Flusher)
	if !ok {
		return echo.NewHTTPError(http.StatusInternalServerError, "streaming no soportado")
	}
	sse := func(event, data string) {
		fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, data)
		fl.Flush()
	}
	sse("session", sid)

	// engine: DeepSeek when configured, deterministic sommelier otherwise
	eng := sommelier.NewEngine(s.Store, nil)
	if s.Cfg.DeepSeekKey != "" {
		eng.LLM = sommelier.NewClient(s.Cfg.DeepSeekKey, s.Cfg.DeepSeekModel, s.Cfg.DeepSeekBaseURL)
	}

	// collect recommendations surfaced via tools (dedup, first-seen order)
	recs := map[string]sommelier.Recomendacion{}
	order := []string{}
	eng.OnRecommend = func(batch []sommelier.Recomendacion) {
		for _, r := range batch {
			if _, dup := recs[r.CveArt]; !dup {
				recs[r.CveArt] = r
				order = append(order, r.CveArt)
			}
		}
	}

	ctx, cancel := stdContextWithTimeout(60 * time.Second)
	defer cancel()
	reply, _, err := eng.Answer(ctx, sess.Messages, func(chunk string) {
		sse("delta", chunk)
	})
	if err != nil {
		reply = "Disculpa, tuve un problema técnico. Intenta de nuevo o escríbenos por WhatsApp: https://wa.me/525511322231"
	}

	chatbot.AppendMessage(&sess, "assistant", reply)
	_ = s.Store.PutJSON(badger.PrefChatSess+sess.ID, sess)

	final := []sommelier.Recomendacion{}
	for _, k := range order {
		final = append(final, recs[k])
	}
	if len(final) > 0 {
		if b, err := json.Marshal(final); err == nil {
			sse("products", string(b))
		}
	}
	sse("done", reply)
	return nil
}

// publicPerfil returns the curated wine profile for one product — the
// editorial layer (notes, pairing, grapes) the sommelier reasons over.
// GET /api/products/:cve/perfil
func (s *Server) publicPerfil(c echo.Context) error {
	eng := sommelier.NewEngine(s.Store, nil)
	ficha, ok := eng.FichaVino(c.Param("cve"))
	if !ok {
		return echo.NewHTTPError(http.StatusNotFound, "producto no encontrado")
	}
	return c.JSON(http.StatusOK, ficha)
}
