package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"

	"menguz/internal/badger"
	"menguz/internal/config"
	"menguz/internal/middleware"
	"menguz/internal/models"
)

// ---------- Token service: access JWT (15m) + rotating refresh (7d) ----------

type refreshRec struct {
	Sub   string    `json:"sub"`
	Exp   time.Time `json:"exp"`
	Token string    `json:"token"`
}

func (s *Server) issueAccess(sub, email, rol string, lista int) (string, error) {
	claims := middleware.Claims{
		Sub:   sub,
		Email: email,
		Rol:   rol,
		Lista: lista,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(nowUTC().Add(s.Cfg.AccessTTL)),
			IssuedAt:  jwt.NewNumericDate(nowUTC()),
			Issuer:    "menguz",
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(s.Cfg.JWTSecret))
}

func (s *Server) issueRefresh(sub string) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	tok := hex.EncodeToString(b)
	rec := refreshRec{Sub: sub, Exp: nowUTC().Add(s.Cfg.RefreshTTL), Token: tok}
	if err := s.Store.PutJSON(badger.PrefRefresh+tok, rec); err != nil {
		return "", err
	}
	return tok, nil
}

// rotateRefresh consumes a refresh token and returns a new pair. Old token is deleted.
func (s *Server) rotateRefresh(tok string) (refreshRec, error) {
	var rec refreshRec
	key := badger.PrefRefresh + tok
	if err := s.Store.GetJSON(key, &rec); err != nil {
		return rec, errors.New("refresh inválido")
	}
	if nowUTC().After(rec.Exp) {
		_ = s.Store.Delete(key)
		return rec, errors.New("refresh expirado")
	}
	_ = s.Store.Delete(key) // single use
	return rec, nil
}

// ---------- Admin auth ----------

type loginReq struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

func (s *Server) adminLogin(c echo.Context) error {
	var req loginReq
	if err := c.Bind(&req); err != nil {
		return badRequest("JSON inválido")
	}
	id, err := s.Store.GetString(badger.PrefUserEmail + req.Email)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "credenciales inválidas")
	}
	var u models.User
	if err := s.Store.GetJSON(badger.PrefUser+id, &u); err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "credenciales inválidas")
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)) != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "credenciales inválidas")
	}
	return s.respondWithTokens(c, u.ID, u.Email, string(u.Rol), 0)
}

// respondWithTokens issues access+refresh and sets cookies for the admin panel.
func (s *Server) respondWithTokens(c echo.Context, sub, email, rol string, lista int) error {
	access, err := s.issueAccess(sub, email, rol, lista)
	if err != nil {
		return err
	}
	refresh, err := s.issueRefresh(sub)
	if err != nil {
		return err
	}
	setAuthCookies(c, access, refresh, s.Cfg)
	return c.JSON(http.StatusOK, map[string]any{
		"access_token":  access,
		"refresh_token": refresh,
		"token_type":    "Bearer",
		"expires_in":    int(s.Cfg.AccessTTL.Seconds()),
		"rol":           rol,
	})
}

func setAuthCookies(c echo.Context, access, refresh string, cfg *config.Config) {
	secure := cfg.IsSecure()
	c.SetCookie(&http.Cookie{Name: "mgz_token", Value: access, HttpOnly: true, Secure: secure, SameSite: http.SameSiteStrictMode, Path: "/", MaxAge: int(cfg.AccessTTL.Seconds())})
	c.SetCookie(&http.Cookie{Name: "mgz_refresh", Value: refresh, HttpOnly: true, Secure: secure, SameSite: http.SameSiteStrictMode, Path: "/", MaxAge: int(cfg.RefreshTTL.Seconds())})
	c.SetCookie(&http.Cookie{Name: "csrf", Value: randomHex(16), HttpOnly: false, Secure: secure, SameSite: http.SameSiteStrictMode, Path: "/"})
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// refreshHandler rotates both tokens.
func (s *Server) refreshHandler(c echo.Context) error {
	tok := c.FormValue("refresh_token")
	if tok == "" {
		tok = c.QueryParam("refresh_token")
	}
	if tok == "" {
		var body struct {
			RefreshToken string `json:"refresh_token"`
		}
		if c.Bind(&body) == nil {
			tok = body.RefreshToken
		}
	}
	rec, err := s.rotateRefresh(tok)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}
	// load identity for claims
	var u models.User
	if err := s.Store.GetJSON(badger.PrefUser+rec.Sub, &u); err == nil {
		return s.respondWithTokens(c, u.ID, u.Email, string(u.Rol), 0)
	}
	// B2B client refresh
	var cl models.Cliente
	if err := s.Store.GetJSON(badger.PrefCliente+rec.Sub, &cl); err == nil {
		access, err := s.issueAccess(cl.ID, cl.Email, "b2b", cl.ListaPrecio)
		if err != nil {
			return err
		}
		refresh, err := s.issueRefresh(cl.ID)
		if err != nil {
			return err
		}
		return c.JSON(http.StatusOK, map[string]any{
			"access_token": access, "refresh_token": refresh,
			"token_type": "Bearer", "expires_in": int(s.Cfg.AccessTTL.Seconds()), "rol": "b2b",
		})
	}
	return echo.NewHTTPError(http.StatusUnauthorized, "identidad no encontrada")
}

func (s *Server) logout(c echo.Context) error {
	// best-effort: delete refresh cookie token
	if ck, err := c.Cookie("mgz_refresh"); err == nil {
		_ = s.Store.Delete(badger.PrefRefresh + ck.Value)
	}
	c.SetCookie(&http.Cookie{Name: "mgz_token", Value: "", Path: "/", MaxAge: -1})
	c.SetCookie(&http.Cookie{Name: "mgz_refresh", Value: "", Path: "/", MaxAge: -1})
	return c.NoContent(http.StatusNoContent)
}

// ---------- Bootstrap admin ----------

func (s *Server) EnsureAdmin() error {
	_, err := s.Store.GetString(badger.PrefUserEmail + s.Cfg.AdminEmail)
	if err == nil {
		return nil // already bootstrapped
	}
	pass := s.Cfg.AdminPass
	if pass == "" {
		pass = randomHex(6)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(pass), s.Cfg.BcryptCost)
	if err != nil {
		return err
	}
	u := models.User{
		ID: newID("usr"), Email: s.Cfg.AdminEmail, Nombre: "Administrador",
		Rol: models.RolAdmin, PasswordHash: string(hash), CreatedAt: nowUTC(),
	}
	if err := s.Store.PutJSON(badger.PrefUser+u.ID, u); err != nil {
		return err
	}
	if err := s.Store.PutString(badger.PrefUserEmail+u.Email, u.ID); err != nil {
		return err
	}
	// log the generated password once via stderr so it shows in journalctl
	msg, _ := json.Marshal(map[string]string{
		"event": "admin_bootstrapped", "email": u.Email, "password": pass,
	})
	_, _ = os.Stderr.Write(append(msg, '\n'))
	return nil
}
