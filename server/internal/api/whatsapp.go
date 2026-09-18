package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/labstack/echo/v4"

	"menguz/internal/badger"
	"menguz/internal/chatbot"
	"menguz/internal/models"
)

// WhatsApp Business API (Meta) integration — Month 2.
// GET  /webhooks/whatsapp  : Meta verification handshake
// POST /webhooks/whatsapp  : inbound messages → chatbot session → reply
//
// Configure in Meta App dashboard:
//   callback URL: https://<dominio>/webhooks/whatsapp
//   verify token: the value of WA_VERIFY_TOKEN

func (s *Server) waVerify(c echo.Context) error {
	mode := c.QueryParam("hub.mode")
	token := c.QueryParam("hub.verify_token")
	challenge := c.QueryParam("hub.challenge")
	if mode == "subscribe" && token != "" && token == s.Cfg.WAVerifyToken {
		return c.String(http.StatusOK, challenge)
	}
	return c.NoContent(http.StatusForbidden)
}

type waPayload struct {
	Object string `json:"object"`
	Entry  []struct {
		ID      string `json:"id"`
		Changes []struct {
			Value struct {
				MessagingProduct string `json:"messaging_product"`
				Metadata         struct {
					PhoneNumberID string `json:"phone_number_id"`
				} `json:"metadata"`
				Messages []struct {
					From string `json:"from"`
					ID   string `json:"id"`
					Type string `json:"type"`
					Text struct {
						Body string `json:"body"`
					} `json:"text"`
				} `json:"messages"`
				Statuses []json.RawMessage `json:"statuses"`
			} `json:"value"`
		} `json:"changes"`
	} `json:"entry"`
}

func (s *Server) waReceive(c echo.Context) error {
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return badRequest("cuerpo inválido")
	}
	var pl waPayload
	if err := json.Unmarshal(body, &pl); err != nil {
		// Meta expects 200 even for payloads we don't parse
		return c.NoContent(http.StatusOK)
	}
	for _, entry := range pl.Entry {
		for _, ch := range entry.Changes {
			for _, m := range ch.Value.Messages {
				if m.Type != "text" {
					continue
				}
				sess, reply := s.waHandleMessage(m.From, m.Text.Body)
				_ = sess
				if reply != "" && s.Cfg.WAAccessToken != "" {
					_ = s.waSend(m.From, reply)
				}
			}
		}
	}
	return c.NoContent(http.StatusOK)
}

// waHandleMessage runs the chatbot for an inbound WhatsApp text and stores
// the conversation under a whatsapp session key.
func (s *Server) waHandleMessage(from, text string) (*models.ChatSession, string) {
	sid := "wa_" + from
	var sess models.ChatSession
	if err := s.Store.GetJSON(badger.PrefChatSess+sid, &sess); err != nil {
		sess = models.ChatSession{
			ID: sid, Canal: "whatsapp", ClienteTel: from,
			Messages: []models.ChatMessage{}, CreatedAt: nowUTC(), UpdatedAt: nowUTC(),
		}
	}
	chatbot.AppendMessage(&sess, "user", text)
	agent := &chatbot.Agent{Store: s.Store}
	reply := agent.Answer(&sess, text) // non-streaming path for WhatsApp
	chatbot.AppendMessage(&sess, "assistant", reply)
	_ = s.Store.PutJSON(badger.PrefChatSess+sid, sess)
	return &sess, reply
}

// waSend posts a text reply to the Meta Graph API. No-op without credentials.
func (s *Server) waSend(to, text string) error {
	url := "https://graph.facebook.com/v20.0/" + s.Cfg.WAPhoneID + "/messages"
	payload, _ := json.Marshal(map[string]any{
		"messaging_product": "whatsapp",
		"to":                to,
		"type":              "text",
		"text":              map[string]any{"body": text},
	})
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.Cfg.WAAccessToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return &httpError{status: resp.StatusCode, msg: string(b)}
	}
	return nil
}

type httpError struct {
	status int
	msg    string
}

func (e *httpError) Error() string { return e.msg }
