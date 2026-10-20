package main

import (
	"crypto/sha256"
	"crypto/subtle"
	"mime"
	"net/http"
	"strings"
)

const maxBodyBytes = 1 << 20 // 1 MB is ruim genoeg voor elk verzoek van de frontend

// withSecurity zet alle beschermingslagen om de routes heen, van buiten naar binnen:
// security headers, optioneel wachtwoord, CSRF-bescherming en alleen-JSON voor wijzigingen.
func withSecurity(next http.Handler, cfg config) http.Handler {
	h := requireJSON(next)
	h = http.NewCrossOriginProtection().Handler(h)
	if cfg.authEnabled() {
		h = basicAuth(h, cfg.AuthUser, cfg.AuthPassword)
	}
	return securityHeaders(h)
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		// 'unsafe-inline' alleen voor stijlen: de frontend zet kleuren en breedtes via style-attributen
		h.Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; frame-ancestors 'none'; base-uri 'none'; form-action 'self'")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}

// basicAuth vraagt om een gebruikersnaam en wachtwoord; /healthz blijft vrij voor monitoring
func basicAuth(next http.Handler, user, password string) http.Handler {
	wantUser, wantPass := sha256.Sum256([]byte(user)), sha256.Sum256([]byte(password))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			next.ServeHTTP(w, r)
			return
		}
		u, p, ok := r.BasicAuth()
		gotUser, gotPass := sha256.Sum256([]byte(u)), sha256.Sum256([]byte(p))
		// Altijd beide vergelijken, zodat de duur niets verraadt
		userOK := subtle.ConstantTimeCompare(gotUser[:], wantUser[:]) == 1
		passOK := subtle.ConstantTimeCompare(gotPass[:], wantPass[:]) == 1
		if !ok || !userOK || !passOK {
			w.Header().Set("WWW-Authenticate", `Basic realm="Magyar woordtrainer", charset="UTF-8"`)
			http.Error(w, "inloggen vereist", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// requireJSON accepteert wijzigende API-verzoeken alleen als JSON. Een andere website kan zonder
// toestemming van de browser geen JSON sturen, dus dit sluit ook "simpele" cross-site POSTs uit.
// Daarnaast wordt de grootte van de body begrensd.
func requireJSON(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead || !strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}
		if mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type")); err != nil || mt != "application/json" {
			writeJSON(w, http.StatusUnsupportedMediaType, map[string]string{"error": "verzoek moet JSON zijn (Content-Type: application/json)"})
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		next.ServeHTTP(w, r)
	})
}
