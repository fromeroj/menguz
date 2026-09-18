// Package middleware provides HTTP middleware: JWT auth, CORS, request
// logging, rate limiting and security headers.
package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

// ---- JWT claims ----

type Claims struct {
	Sub   string `json:"sub"` // user id or cliente id
	Email string `json:"email"`
	Rol   string `json:"rol"`             // admin | ventas | b2b
	Lista int    `json:"lista,omitempty"` // SAE price list for B2B
	jwt.RegisteredClaims
}

// RequireAuth validates the Bearer token and injects claims into the context.
func RequireAuth(secret string, roles ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			h := c.Request().Header.Get("Authorization")
			if !strings.HasPrefix(h, "Bearer ") {
				return echo.NewHTTPError(http.StatusUnauthorized, "token requerido")
			}
			tok := strings.TrimPrefix(h, "Bearer ")
			claims := &Claims{}
			_, err := jwt.ParseWithClaims(tok, claims, func(t *jwt.Token) (any, error) {
				return []byte(secret), nil
			}, jwt.WithValidMethods([]string{"HS256"}))
			if err != nil || claims.Sub == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "token inválido o expirado")
			}
			if len(roles) > 0 {
				ok := false
				for _, r := range roles {
					if claims.Rol == r {
						ok = true
						break
					}
				}
				if !ok {
					return echo.NewHTTPError(http.StatusForbidden, "rol no autorizado")
				}
			}
			c.Set("claims", claims)
			return next(c)
		}
	}
}

func ClaimsOf(c echo.Context) *Claims {
	cl, _ := c.Get("claims").(*Claims)
	return cl
}

// ---- Security headers ----

func SecurityHeaders(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		h := c.Response().Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		// Tailwind/HTMX admin panel needs CDN scripts + inline styles
		h.Set("Content-Security-Policy",
			"default-src 'self'; script-src 'self' https://cdn.tailwindcss.com https://unpkg.com 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; connect-src 'self'")
		return next(c)
	}
}

// ---- CORS ----

func CORS(origins []string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			origin := c.Request().Header.Get("Origin")
			allowed := origin == "" // same-origin / curl
			for _, o := range origins {
				if subtle.ConstantTimeCompare([]byte(origin), []byte(o)) == 1 {
					allowed = true
					break
				}
			}
			if allowed && origin != "" {
				c.Response().Header().Set("Access-Control-Allow-Origin", origin)
				c.Response().Header().Set("Vary", "Origin")
				c.Response().Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-CSRF")
				c.Response().Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				c.Response().Header().Set("Access-Control-Allow-Credentials", "true")
			}
			if c.Request().Method == http.MethodOptions {
				return c.NoContent(http.StatusNoContent)
			}
			return next(c)
		}
	}
}

// ---- Rate limiting (in-memory token bucket per key) ----

type bucket struct {
	tokens float64
	last   time.Time
}

type RateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	rate    float64 // tokens per second
	burst   float64
}

func NewRateLimiter(ratePerMin float64, burst float64) *RateLimiter {
	return &RateLimiter{
		buckets: map[string]*bucket{},
		rate:    ratePerMin / 60,
		burst:   burst,
	}
}

func (r *RateLimiter) Allow(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	b, ok := r.buckets[key]
	now := time.Now()
	if !ok {
		b = &bucket{tokens: r.burst, last: now}
		r.buckets[key] = b
	}
	b.tokens += now.Sub(b.last).Seconds() * r.rate
	if b.tokens > r.burst {
		b.tokens = r.burst
	}
	b.last = now
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// Limit middleware: per-IP global bucket. Use tighter buckets on checkout/chat.
func (r *RateLimiter) Limit() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ip := c.RealIP()
			if !r.Allow(ip) {
				return echo.NewHTTPError(http.StatusTooManyRequests, "demasiadas solicitudes, intenta más tarde")
			}
			return next(c)
		}
	}
}

// ---- CSRF-lite for cookie-based admin sessions (double submit) ----

// CSRF validates that cookie "csrf" equals header/form "csrf" on unsafe methods.
// The login page is exempt (no session exists yet, and it holds no secrets).
func CSRF() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			m := c.Request().Method
			if m == http.MethodGet || m == http.MethodHead || m == http.MethodOptions {
				return next(c)
			}
			if c.Request().URL.Path == "/admin/login" {
				return next(c)
			}
			cookie, err := c.Cookie("csrf")
			if err != nil {
				return echo.NewHTTPError(http.StatusForbidden, "csrf: falta cookie")
			}
			submitted := c.Request().Header.Get("X-CSRF")
			if submitted == "" {
				submitted = c.FormValue("csrf")
			}
			if subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(submitted)) != 1 {
				return echo.NewHTTPError(http.StatusForbidden, "csrf inválido")
			}
			return next(c)
		}
	}
}
