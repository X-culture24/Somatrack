package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort         string
	ServerAddr         string
	DjangoSecretKey    string
	Debug              bool
	AllowedHosts       []string
	DatabaseURL        string
	PortalURL          string
	NovaURL            string
	PortalOrigin       string
	NovaOrigin         string
	CORSAllowedOrigins []string
	CSRFTrustedOrigins []string
	TimeZone           string
	StaticURL          string
	MediaURL           string
	MediaRoot          string
	ServeMediaInDebug  bool
	JWTSecret          string
	JWTAccessTTL       time.Duration
	JWTRefreshTTL      time.Duration
	JWTRotateRefresh   bool
	SchoolName         string
	SchoolAddress      string
	SchoolPhone        string
	BankAccountName    string
	BankName           string
	BankAccountNo      string
	MpesaPaybill       string
	MpesaAccountPrefix string
	MpesaSharedSecret  string
	MpesaAllowedIPs    []string
	NovaMaxUploadBytes int64
	UseRedis           bool
	RedisURL           string
	EmailBackend       string
	DefaultFromEmail   string
	FingerprintEncKey  string
	FingerprintEncSalt string
}

var Loaded *Config

func Load(envPath ...string) (*Config, error) {
	if len(envPath) > 0 {
		_ = godotenv.Load(envPath[0])
	} else {
		_ = godotenv.Load()
	}

	debug := strings.ToLower(getEnv("DJANGO_DEBUG", "true")) == "true"
	jwtAccessTTL, _ := time.ParseDuration(getEnv("JWT_ACCESS_TTL", "8h"))
	jwtRefreshTTL, _ := time.ParseDuration(getEnv("JWT_REFRESH_TTL", "168h"))
	novaMaxUpload, _ := strconv.ParseInt(getEnv("NOVA_MAX_UPLOAD_BYTES", "10485760"), 10, 64)
	useRedis := strings.ToLower(getEnv("USE_CELERY", "false")) == "true" ||
		strings.ToLower(getEnv("USE_REDIS", "false")) == "true"

	cfg := &Config{
		ServerPort:         getEnv("SERVER_PORT", "8000"),
		ServerAddr:         fmt.Sprintf(":%s", getEnv("SERVER_PORT", "8000")),
		DjangoSecretKey:    getEnv("DJANGO_SECRET_KEY", "insecure-dev-key"),
		Debug:              debug,
		AllowedHosts:       splitCSV(getEnv("DJANGO_ALLOWED_HOSTS", "localhost,127.0.0.1")),
		DatabaseURL:        getEnv("DATABASE_URL", "postgres://st_mary:James_Bond007%21@127.0.0.1:5432/st_marys_db"),
		PortalURL:          strings.TrimRight(getEnv("PORTAL_URL", "http://localhost:5173"), "/"),
		NovaURL:            strings.TrimRight(getEnv("NOVA_URL", "http://localhost:5174"), "/"),
		CORSAllowedOrigins: dedupe(append(splitCSV(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5173")),
			strings.TrimRight(getEnv("PORTAL_URL", "http://localhost:5173"), "/"),
			strings.TrimRight(getEnv("NOVA_URL", "http://localhost:5174"), "/"))),
		CSRFTrustedOrigins: dedupe(append(splitCSV(getEnv("CSRF_TRUSTED_ORIGINS", "")),
			strings.TrimRight(getEnv("PORTAL_URL", "http://localhost:5173"), "/"),
			strings.TrimRight(getEnv("NOVA_URL", "http://localhost:5174"), "/"))),
		TimeZone:           getEnv("TIME_ZONE", "Africa/Nairobi"),
		MediaURL:           getEnv("MEDIA_URL", "media/"),
		MediaRoot:          getEnv("MEDIA_ROOT", "media"),
		ServeMediaInDebug:  strings.ToLower(getEnv("SERVE_MEDIA_IN_DEBUG", "false")) == "true",
		JWTSecret:          getEnv("JWT_SECRET", getEnv("DJANGO_SECRET_KEY", "insecure-dev-key")),
		JWTAccessTTL:       jwtAccessTTL,
		JWTRefreshTTL:      jwtRefreshTTL,
		JWTRotateRefresh:   true,
		SchoolName:         getEnv("SCHOOL_NAME", "ACK St. Mary's School Kabete"),
		SchoolAddress:      getEnv("SCHOOL_ADDRESS", "P.O BOX 29190-00625, Nairobi"),
		SchoolPhone:        getEnv("SCHOOL_PHONE", "0746714946"),
		BankAccountName:    getEnv("BANK_ACCOUNT_NAME", "ACK ST. MARY'S SCHOOL KABETE"),
		BankName:           getEnv("BANK_NAME", "Equity Bank Kangemi"),
		BankAccountNo:      getEnv("BANK_ACCOUNT_NO", "1370263402101"),
		MpesaPaybill:       getEnv("MPESA_PAYBILL", "247247"),
		MpesaAccountPrefix: getEnv("MPESA_ACCOUNT_PREFIX", "137101"),
		MpesaSharedSecret:  getEnv("MPESA_SHARED_SECRET", ""),
		MpesaAllowedIPs:    splitCSV(getEnv("MPESA_ALLOWED_IPS", "")),
		NovaMaxUploadBytes: novaMaxUpload,
		UseRedis:           useRedis,
		RedisURL:           getEnv("REDIS_URL", "redis://127.0.0.1:6379/0"),
		EmailBackend:       getEnv("EMAIL_BACKEND", "console"),
		DefaultFromEmail:   getEnv("DEFAULT_FROM_EMAIL", "noreply@stmaryskabete.ac.ke"),
		FingerprintEncKey:  getEnv("FINGERPRINT_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"),
		FingerprintEncSalt: getEnv("FINGERPRINT_ENCRYPTION_SALT", "stmarys-fp-salt"),
	}
	cfg.PortalOrigin = cfg.PortalURL
	cfg.NovaOrigin = cfg.NovaURL
	if err := cfg.checkInsecureDefaults(); err != nil {
		return nil, err
	}
	Loaded = cfg
	return cfg, nil
}

// insecureDefaultSecrets are the fallback values baked into docker-compose.yml
// and .env.example so the stack boots out of the box for local dev. If any
// of these are still in effect once DJANGO_DEBUG=false (i.e. "production"),
// every JWT, inter-service call, and encrypted fingerprint template is
// protected by a secret anyone can read directly from this repo — refuse to
// start instead of silently running insecurely.
var insecureDefaultSecrets = []string{
	"change-me-in-production",
	"change-me-in-production-0123456789abcdef",
	"insecure-dev-key",
	"dev-stmarys-kabete-change-me",
	"dev-local-secret-key-0123456789abcdef",
}

const insecureFingerprintKey = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func (c *Config) checkInsecureDefaults() error {
	if c.Debug {
		return nil
	}
	for _, bad := range insecureDefaultSecrets {
		if c.DjangoSecretKey == bad {
			return fmt.Errorf("refusing to start with DJANGO_DEBUG=false while DJANGO_SECRET_KEY is still the checked-in default (%q) — set a real secret", bad)
		}
		if c.JWTSecret == bad {
			return fmt.Errorf("refusing to start with DJANGO_DEBUG=false while JWT_SECRET is still the checked-in default (%q) — anyone can forge valid tokens", bad)
		}
	}
	if c.FingerprintEncKey == insecureFingerprintKey {
		return fmt.Errorf("refusing to start with DJANGO_DEBUG=false while FINGERPRINT_ENCRYPTION_KEY is still the checked-in default — stored biometric templates would be encrypted with a publicly-known key")
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func splitCSV(s string) []string {
	if s == "" {
		return []string{}
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func dedupe(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if _, ok := seen[s]; !ok {
			seen[s] = struct{}{}
			out = append(out, s)
		}
	}
	return out
}

func NairobiLocation() *time.Location {
	loc, err := time.LoadLocation(Loaded.TimeZone)
	if err != nil {
		loc = time.FixedZone("EAT", 3*3600)
	}
	return loc
}
