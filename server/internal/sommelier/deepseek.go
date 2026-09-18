package sommelier

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
)

// Client is a minimal OpenAI-compatible chat client (DeepSeek, OpenAI,
// proxies) supporting streaming and tool/function calling. Hand-rolled to
// keep the binary dependency-free.
type Client struct {
	APIKey  string
	Model   string
	BaseURL string
	HTTP    *http.Client
}

func NewClient(apiKey, model, baseURL string) *Client {
	if baseURL == "" {
		baseURL = "https://api.deepseek.com"
	}
	return &Client{
		APIKey: apiKey, Model: model, BaseURL: strings.TrimRight(baseURL, "/"),
		HTTP: &http.Client{Timeout: 90 * time.Second},
	}
}

// Tool is the OpenAI tools schema entry.
type Tool struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

type ToolFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

// Message is one chat message. ToolCalls is set on assistant messages that
// request tool execution; ToolCallID on the resulting tool messages.
type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function CallFunction `json:"function"`
}

type CallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// Reply is the assembled model turn: streamed text plus any tool calls.
type Reply struct {
	Text      string
	ToolCalls []ToolCall
}

// Chat streams one completion. Content deltas are forwarded to delta (may be
// nil). When the model requests tool calls they are assembled and returned;
// content is still streamed as it arrives (usually empty on tool turns).
func (cl *Client) Chat(ctx context.Context, system string, history []Message, tools []Tool, delta func(string)) (*Reply, error) {
	msgs := append([]Message{{Role: "system", Content: system}}, history...)
	body, _ := json.Marshal(map[string]any{
		"model":       cl.Model,
		"messages":    msgs,
		"tools":       tools,
		"stream":      true,
		"temperature": 0.4,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cl.BaseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+cl.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	resp, err := cl.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("llm %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}

	var text strings.Builder
	pending := map[int]*ToolCall{} // assembled by tool-call index
	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 64*1024), 1<<20)
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
					Content   string `json:"content"`
					ToolCalls []struct {
						Index    int    `json:"index"`
						ID       string `json:"id"`
						Type     string `json:"type"`
						Function struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						} `json:"function"`
					} `json:"tool_calls"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(payload), &ev); err != nil {
			continue
		}
		if len(ev.Choices) == 0 {
			continue
		}
		d := ev.Choices[0].Delta
		if d.Content != "" {
			text.WriteString(d.Content)
			if delta != nil {
				delta(d.Content)
			}
		}
		for _, tc := range d.ToolCalls {
			acc, ok := pending[tc.Index]
			if !ok {
				acc = &ToolCall{Type: "function"}
				pending[tc.Index] = acc
			}
			if tc.ID != "" {
				acc.ID = tc.ID
			}
			if tc.Type != "" {
				acc.Type = tc.Type
			}
			if tc.Function.Name != "" {
				acc.Function.Name = tc.Function.Name
			}
			acc.Function.Arguments += tc.Function.Arguments
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	out := &Reply{Text: text.String()}
	for i := 0; i < len(pending); i++ {
		if tc, ok := pending[i]; ok && tc.ID != "" {
			out.ToolCalls = append(out.ToolCalls, *tc)
		}
	}
	return out, nil
}

// Tools exposes the sommelier's inventory tools to the model.
func Tools() []Tool {
	str := map[string]any{"type": "string"}
	num := map[string]any{"type": "number"}
	boolT := map[string]any{"type": "boolean"}
	return []Tool{
		{Type: "function", Function: ToolFunction{
			Name:        "buscar_vinos",
			Description: "Busca vinos y licores en el inventario VIVO de Menguz (precios y existencias reales). Usa este tool SIEMPRE antes de recomendar o citar precios. Filtra por tipo, uva, bodega, región/país, ocasión o rango de precio (MXN).",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"q":              withDesc(str, "texto libre: nombre, estilo, sabor, comida a maridar"),
					"tipo":           withDesc(str, "Tinto | Blanco | Rosado | Espumoso | Champagne | Licor"),
					"uva":            withDesc(str, "varietal, ej. Malbec, Tempranillo, Chardonnay"),
					"bodega":         withDesc(str, "productor/bodega"),
					"region":         withDesc(str, "región o denominación de origen"),
					"pais":           withDesc(str, "país, ej. México, España, Francia, Argentina, Italia"),
					"ocasion":        withDesc(str, "ocasión: regalo, cena romántica, celebración, asado, verano, digestivo, brindis"),
					"tag":            withDesc(str, "característica: frutal, seco, dulce, tánico, añejo, artesanal, cremoso"),
					"precio_max":     withDesc(num, "precio máximo MXN por botella"),
					"precio_min":     withDesc(num, "precio mínimo MXN"),
					"disponibles":    withDesc(boolT, "true = solo con existencia > 0"),
					"max_resultados": withDesc(num, "cuántos resultados (default 8, máx 12)"),
				},
			},
		}},
		{Type: "function", Function: ToolFunction{
			Name:        "ficha_vino",
			Description: "Obtiene la ficha completa de UN producto por su clave SAE (descripción, notas de cata, maridaje, uvas, bodega, imagen, precio y existencia). Úsala cuando el cliente pregunte por una botella específica o quiera más detalle de una recomendación.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"cve_art": withDesc(str, "clave SAE del artículo, ej. VINO-0123"),
				},
				"required": []string{"cve_art"},
			},
		}},
	}
}

func withDesc(t map[string]any, desc string) map[string]any {
	m := map[string]any{}
	for k, v := range t {
		m[k] = v
	}
	m["description"] = desc
	return m
}
