// Package config loads configuration from CLI flags and environment variables.
// Flags take precedence over environment variables.
package config

import (
	"flag"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port        int
	BadgerPath  string
	SQLitePath  string
	StaticDir   string // storefront build (design/app/dist); served if present
	DataDir     string
	JWTSecret   string
	AccessTTL   time.Duration
	RefreshTTL  time.Duration
	AdminEmail  string
	AdminPass   string
	BcryptCost  int
	CORSOrigins []string

	// SAE / Firebird
	SAEHost    string
	SAEPort    int
	SAEUser    string
	SAEPass    string
	SAEDB      string
	SAEMock    bool   // generate demo catalog instead of connecting to Firebird
	SyncAt     string // daily sync time "03:00"
	SyncOnBoot bool

	// Payments: bank transfer reference (B2C checkout per proposal PDF)
	BankName   string
	BankCLABE  string
	BankHolder string

	// WhatsApp Business API (Meta)
	WAVerifyToken string
	WAAccessToken string
	WAPhoneID     string

	// OpenAI chatbot
	OpenAIKey   string
	OpenAIModel string

	// DeepSeek sommelier (Hedonism-style wine advisor; OpenAI-compatible API)
	DeepSeekKey     string
	DeepSeekModel   string
	DeepSeekBaseURL string
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func parse() *Config {
	c := &Config{}
	fs := flag.NewFlagSet("menguz-server", flag.ExitOnError)
	fs.IntVar(&c.Port, "port", 8080, "HTTP port")
	fs.StringVar(&c.BadgerPath, "badger-path", getenv("BADGER_PATH", "./data/badger"), "Badger data dir")
	fs.StringVar(&c.SQLitePath, "sqlite-path", getenv("SQLITE_PATH", "./data/analytics.db"), "SQLite analytics DB path")
	fs.StringVar(&c.StaticDir, "static-dir", getenv("STATIC_DIR", "../design/app/dist"), "storefront static build dir")
	fs.StringVar(&c.DataDir, "data-dir", getenv("DATA_DIR", "./data"), "data root (uploads, backups)")
	fs.StringVar(&c.JWTSecret, "jwt-secret", getenv("JWT_SECRET", ""), "JWT signing secret (required in prod)")
	fs.DurationVar(&c.AccessTTL, "access-ttl", 15*time.Minute, "access token TTL")
	fs.DurationVar(&c.RefreshTTL, "refresh-ttl", 7*24*time.Hour, "refresh token TTL")
	fs.StringVar(&c.AdminEmail, "admin-email", getenv("ADMIN_EMAIL", "admin@menguz.local"), "bootstrap admin email")
	fs.StringVar(&c.AdminPass, "admin-pass", getenv("ADMIN_PASS", ""), "bootstrap admin password (prompted if empty)")
	fs.IntVar(&c.BcryptCost, "bcrypt-cost", 12, "bcrypt cost")
	fs.StringVar(&c.SAEHost, "sae-host", getenv("SAE_HOST", ""), "Firebird/SAE host (empty → mock mode)")
	fs.IntVar(&c.SAEPort, "sae-port", 3050, "Firebird port")
	fs.StringVar(&c.SAEUser, "sae-user", getenv("SAE_USER", "SYSDBA"), "Firebird read-only user")
	fs.StringVar(&c.SAEPass, "sae-pass", getenv("SAE_PASS", ""), "Firebird password")
	fs.StringVar(&c.SAEDB, "sae-db", getenv("SAE_DB", ""), "path to .fdb on SAE server")
	fs.BoolVar(&c.SAEMock, "sae-mock", getenv("SAE_MOCK", "true") == "true", "use mock SAE data source")
	fs.StringVar(&c.SyncAt, "sync-at", getenv("SYNC_AT", "03:00"), "daily sync time HH:MM")
	fs.BoolVar(&c.SyncOnBoot, "sync-on-boot", true, "run sync at startup if data is stale")

	fs.StringVar(&c.BankName, "bank-name", getenv("BANK_NAME", "BBVA"), "bank name for transfer references")
	fs.StringVar(&c.BankCLABE, "bank-clabe", getenv("BANK_CLABE", "000000000000000000"), "CLABE for transfers")
	fs.StringVar(&c.BankHolder, "bank-holder", getenv("BANK_HOLDER", "Menguz Vinos y Licores SA de CV"), "account holder")

	fs.StringVar(&c.WAVerifyToken, "wa-verify-token", getenv("WA_VERIFY_TOKEN", ""), "Meta webhook verify token")
	fs.StringVar(&c.WAAccessToken, "wa-access-token", getenv("WA_ACCESS_TOKEN", ""), "Meta WhatsApp access token")
	fs.StringVar(&c.WAPhoneID, "wa-phone-id", getenv("WA_PHONE_ID", ""), "Meta WhatsApp phone number id")

	fs.StringVar(&c.OpenAIKey, "openai-key", getenv("OPENAI_API_KEY", ""), "OpenAI API key (empty → rule-based chatbot fallback)")
	fs.StringVar(&c.OpenAIModel, "openai-model", getenv("OPENAI_MODEL", "gpt-4o-mini"), "OpenAI chat model")

	fs.StringVar(&c.DeepSeekKey, "deepseek-key", getenv("DEEPSEEK_API_KEY", ""), "DeepSeek API key for the sommelier (empty → rule-based sommelier fallback)")
	fs.StringVar(&c.DeepSeekModel, "deepseek-model", getenv("DEEPSEEK_MODEL", "deepseek-chat"), "DeepSeek chat model")
	fs.StringVar(&c.DeepSeekBaseURL, "deepseek-base-url", getenv("DEEPSEEK_BASE_URL", "https://api.deepseek.com"), "DeepSeek API base URL")
	_ = fs.Parse(os.Args[1:])

	c.CORSOrigins = []string{} // same-origin by default; extend via CORS_ORIGINS="a,b"
	if v := os.Getenv("CORS_ORIGINS"); v != "" {
		for _, o := range splitComma(v) {
			c.CORSOrigins = append(c.CORSOrigins, o)
		}
	}
	if c.SAEHost != "" {
		c.SAEMock = false
	}
	if c.JWTSecret == "" {
		// dev default; refuse empty in production via README guidance
		c.JWTSecret = "menguz-dev-secret-change-me"
	}
	return c
}

func splitComma(s string) []string {
	var out []string
	cur := ""
	for _, ch := range s {
		if ch == ',' {
			if cur != "" {
				out = append(out, cur)
			}
			cur = ""
			continue
		}
		cur += string(ch)
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

func atoi(s string, def int) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

func (c *Config) IsSecure() bool { return c.Port == 443 }

var loaded *Config

// Load returns the singleton config, parsed on first call.
func Load() *Config {
	if loaded == nil {
		loaded = parse()
	}
	return loaded
}
