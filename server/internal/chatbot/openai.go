package chatbot

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"menguz/internal/models"
)

// OpenAIClient streams chat completions over SSE (server-sent events).
// Hand-rolled to avoid SDK weight; only chat.completions is used.
type OpenAIClient struct {
	APIKey string
	Model  string
	Client *http.Client
}

func NewOpenAIClient(key, model string) *OpenAIClient {
	return &OpenAIClient{
		APIKey: key, Model: model,
		Client: &http.Client{Timeout: 60 * time.Second},
	}
}

type chatMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Stream calls the chat completions endpoint with stream=true and invokes
// delta for each content chunk. It returns the full assembled text.
func (o *OpenAIClient) Stream(ctx context.Context, system string, history []models.ChatMessage, delta func(string)) (string, error) {
	msgs := []chatMsg{{Role: "system", Content: system}}
	for _, m := range history {
		if m.Role == "system" {
			continue
		}
		msgs = append(msgs, chatMsg{Role: m.Role, Content: m.Content})
	}
	body, _ := json.Marshal(map[string]any{
		"model":       o.Model,
		"messages":    msgs,
		"stream":      true,
		"temperature": 0.3,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.openai.com/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+o.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	resp, err := o.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return "", fmt.Errorf("openai %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}

	var full strings.Builder
	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 64*1024), 256*1024)
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "[DONE]" {
			break
		}
		var ev struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(payload), &ev); err != nil {
			continue
		}
		if len(ev.Choices) > 0 && ev.Choices[0].Delta.Content != "" {
			full.WriteString(ev.Choices[0].Delta.Content)
			if delta != nil {
				delta(ev.Choices[0].Delta.Content)
			}
		}
	}
	return full.String(), sc.Err()
}

// BuildSystemPrompt assembles the grounded context (RAG) for the model.
// Guardrail: the model is explicitly forbidden from inventing prices or
// stock, and is told to route to WhatsApp when uncertain.
func BuildSystemPrompt(prods []models.Producto) string {
	var b strings.Builder
	b.WriteString(`Eres el asistente virtual de Menguz Vinos y Licores, una vinatería en CDMX.
Reglas estrictas:
- Usa SOLOS los precios y existencias del catálogo incluido abajo; nunca inventes precios.
- Si no hay información suficiente, di que un asesor confirmará por WhatsApp (55 1132 2231).
- Responde en español, cordial y breve (máximo 120 palabras).
- Horario: L-V 9:00-18:30, S 9:00-13:30. Entregas en CDMX en 24h. Pago por transferencia con referencia.
- Si el cliente pide un humano, indica el enlace de WhatsApp.

Catálogo relevante:
`)
	for _, p := range prods {
		price := p.PrecioBase
		if p.PrecioOferta > 0 && p.PrecioOferta < price {
			price = p.PrecioOferta
		}
		fmt.Fprintf(&b, "- %s | %s | $%.2f | existencia %.0f | %s\n",
			p.CveArt, p.Nombre, price, p.Existencia, p.Categoria)
	}
	return b.String()
}
