package internal

import (
	"os"
	"strconv"
)

type Config struct {
	GuiEnabled bool
	ListenSMTP string
	ListenHTTP string
	StoreFile  string
	AppriseURL string
}

func LoadConfig() Config {
	guiEnabled, _ := strconv.ParseBool(getEnv("GUI_ENABLED", "false"))
	return Config{
		GuiEnabled: guiEnabled,
		ListenSMTP: getEnv("LISTEN_SMTP", "25"),
		ListenHTTP: getEnv("LISTEN_HTTP", "8080"),
		StoreFile:  getEnv("STORE_FILE", "records.db"),
		AppriseURL: getEnv("APPRISE_URL", "http://apprise:8000/notify"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
