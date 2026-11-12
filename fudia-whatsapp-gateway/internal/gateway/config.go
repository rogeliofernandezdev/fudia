package gateway

import (
	"errors"
	"net/url"
	"os"
	"strings"
)

type Config struct {
	DatabaseURL, APIKey, ConciergeURL, IngressKey, Address string
	MediaHosts                                             []string
	Development                                            bool
}

func LoadConfig() (Config, error) {
	c := Config{DatabaseURL: os.Getenv("DATABASE_URL"), APIKey: os.Getenv("GATEWAY_API_KEY"), ConciergeURL: strings.TrimRight(os.Getenv("FUDIA_CONCIERGE_URL"), "/"), IngressKey: os.Getenv("CONCIERGE_INGRESS_KEY"), Address: os.Getenv("LISTEN_ADDRESS"), Development: os.Getenv("APP_ENV") == "development"}
	if c.Address == "" {
		c.Address = ":8020"
	}
	for _, host := range strings.Split(os.Getenv("MEDIA_ALLOWED_HOSTS"), ",") {
		if host = strings.TrimSpace(host); host != "" {
			c.MediaHosts = append(c.MediaHosts, host)
		}
	}
	if c.DatabaseURL == "" || len(c.APIKey) < 32 || len(c.IngressKey) < 32 {
		return c, errors.New("database and service authentication must be configured")
	}
	u, err := url.Parse(c.ConciergeURL)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Scheme != "https" && !(c.Development && u.Scheme == "http")) {
		return c, errors.New("Concierge requires an authorized HTTPS URL")
	}
	if !c.Development {
		db, parseErr := url.Parse(c.DatabaseURL)
		if parseErr != nil || (db.Query().Get("sslmode") != "verify-full" && db.Query().Get("sslmode") != "verify-ca") || os.Getenv("SESSION_STORAGE_PROTECTED") != "true" {
			return c, errors.New("production sessions require verified database TLS and protected encrypted storage")
		}
	}
	return c, nil
}
