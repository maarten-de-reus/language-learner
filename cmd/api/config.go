package main

import (
	"errors"
	"net"
	"os"
	"path/filepath"
)

// config komt uit omgevingsvariabelen, zodat dezelfde binary lokaal en in productie draait.
//
//	ADDR           adres om op te luisteren (standaard 127.0.0.1:8080, alleen deze computer)
//	DATA_DIR       map met woorden.json, voortgang.json, antwoorden.jsonl en instellingen.json (standaard de huidige map)
//	AUTH_USER      gebruikersnaam voor basic auth (optioneel, samen met AUTH_PASSWORD)
//	AUTH_PASSWORD  wachtwoord voor basic auth
//	TZ             tijdzone voor de dagindeling van herhalingen (bijv. Europe/Amsterdam)
type config struct {
	Addr         string
	DataDir      string
	AuthUser     string
	AuthPassword string
}

func loadConfig() (config, error) {
	c := config{
		Addr:         envOr("ADDR", "127.0.0.1:8080"),
		DataDir:      envOr("DATA_DIR", "."),
		AuthUser:     os.Getenv("AUTH_USER"),
		AuthPassword: os.Getenv("AUTH_PASSWORD"),
	}
	if (c.AuthUser == "") != (c.AuthPassword == "") {
		return c, errors.New("AUTH_USER en AUTH_PASSWORD moeten allebei gezet zijn, of allebei leeg")
	}
	return c, nil
}

func (c config) wordsPath() string    { return filepath.Join(c.DataDir, "woorden.json") }
func (c config) progressPath() string { return filepath.Join(c.DataDir, "voortgang.json") }
func (c config) reviewsPath() string  { return filepath.Join(c.DataDir, "antwoorden.jsonl") }
func (c config) settingsPath() string { return filepath.Join(c.DataDir, "instellingen.json") }
func (c config) authEnabled() bool    { return c.AuthUser != "" }

// localOnly is true als de server alleen vanaf deze computer bereikbaar is
func (c config) localOnly() bool {
	host, _, err := net.SplitHostPort(c.Addr)
	if err != nil {
		return false
	}
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
