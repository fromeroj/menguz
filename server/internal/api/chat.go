package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"menguz/internal/badger"
	"menguz/internal/chatbot"
	"menguz/internal/models"
)

// chatHandler answers one message and streams the reply as SSE.
// Engine selection: OpenAI (stream) when OPENAI_API_KEY is set,
// deterministic fallback otherwise — both share catalog RAG + guardrails.
func (s *Server) chatHandler(c echo.Context) error {
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

	// session
	sid := req.SessionID
	if sid == "" {
		sid = newID("chat")
	}
	var sess models.ChatSession
	if err := s.Store.GetJSON(badger.PrefChatSess+sid, &sess); err != nil {
		sess = models.ChatSession{
			ID: sid, Canal: "web", Messages: []models.ChatMessage{},
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

	agent := &chatbot.Agent{Store: s.Store}
	var reply string

	if s.Cfg.OpenAIKey != "" {
		// RAG: top catalog matches ground the prompt
		prods := agent.SearchProducts(req.Message, 8)
		system := chatbot.BuildSystemPrompt(prods)
		oa := chatbot.NewOpenAIClient(s.Cfg.OpenAIKey, s.Cfg.OpenAIModel)
		ctx, cancel := stdContextWithTimeout(45 * time.Second)
		defer cancel()
		var err error
		reply, err = oa.Stream(ctx, system, sess.Messages, func(chunk string) {
			sse("delta", chunk)
		})
		if err != nil {
			// graceful degradation to the fallback engine
			reply = agent.Answer(&sess, req.Message)
		}
	} else {
		reply = agent.Answer(&sess, req.Message)
	}

	chatbot.AppendMessage(&sess, "assistant", reply)
	_ = s.Store.PutJSON(badger.PrefChatSess+sess.ID, sess)
	sse("done", reply)
	return nil
}

// ---------- Admin: conversation review ----------

func (s *Server) adminListChatSessions(c echo.Context) error {
	out := []models.ChatSession{}
	var sess models.ChatSession
	err := s.Store.ScanPrefix(badger.PrefChatSess, &sess, func(key string) bool {
		cp := sess
		if len(cp.Messages) > 3 { // preview: last 3 messages
			cp.Messages = cp.Messages[len(cp.Messages)-3:]
		}
		out = append(out, cp)
		return true
	})
	if err != nil {
		return err
	}
	// newest first (insertion sort on small N)
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].UpdatedAt.After(out[j-1].UpdatedAt); j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	lim := len(out)
	if lim > 50 {
		lim = 50
	}
	return c.JSON(http.StatusOK, map[string]any{"items": out[:lim], "total": len(out)})
}

func (s *Server) adminGetChatSession(c echo.Context) error {
	id := c.Param("id")
	var sess models.ChatSession
	if err := s.Store.GetJSON(badger.PrefChatSess+id, &sess); err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "conversación no encontrada")
	}
	return c.JSON(http.StatusOK, sess)
}

func (s *Server) adminResolveChatSession(c echo.Context) error {
	id := c.Param("id")
	var sess models.ChatSession
	if err := s.Store.GetJSON(badger.PrefChatSess+id, &sess); err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "conversación no encontrada")
	}
	sess.Escalado = false
	sess.UpdatedAt = nowUTC()
	_ = s.Store.PutJSON(badger.PrefChatSess+id, sess)
	return c.JSON(http.StatusOK, sess)
}
