package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func secured(t *testing.T, cfg config) http.Handler {
	t.Helper()
	s, _ := newTestServer(t)
	return withSecurity(s.routes(), cfg)
}

func do(h http.Handler, method, path, contentType, body string, mutate ...func(*http.Request)) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	if contentType != "" {
		r.Header.Set("Content-Type", contentType)
	}
	for _, m := range mutate {
		m(r)
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, r)
	return rr
}

func TestWritesMustBeJSON(t *testing.T) {
	h := secured(t, config{})
	// Een "simpel" formulier of text/plain kan elke website zonder preflight versturen
	for _, ct := range []string{"", "text/plain", "application/x-www-form-urlencoded", "multipart/form-data; boundary=x"} {
		if rr := do(h, "POST", "/api/answer", ct, `{"input":"x"}`); rr.Code != http.StatusUnsupportedMediaType {
			t.Errorf("Content-Type %q: verwacht 415, kreeg %d", ct, rr.Code)
		}
	}
	if rr := do(h, "POST", "/api/next", "application/json; charset=utf-8", `{}`); rr.Code != http.StatusOK {
		t.Errorf("JSON-verzoek moet slagen, kreeg %d", rr.Code)
	}
	if rr := do(h, "GET", "/api/state", "", ""); rr.Code != http.StatusOK {
		t.Errorf("GET heeft geen Content-Type nodig, kreeg %d", rr.Code)
	}
}

func TestCrossSiteRequestsAreRejected(t *testing.T) {
	h := secured(t, config{})
	rr := do(h, "POST", "/api/concepts", "application/json", `{"dutch":"x","hungarian":"y"}`, func(r *http.Request) {
		r.Header.Set("Sec-Fetch-Site", "cross-site")
	})
	if rr.Code != http.StatusForbidden {
		t.Errorf("verzoek vanaf een andere website: verwacht 403, kreeg %d", rr.Code)
	}
	rr = do(h, "POST", "/api/next", "application/json", `{}`, func(r *http.Request) {
		r.Header.Set("Sec-Fetch-Site", "same-origin")
	})
	if rr.Code != http.StatusOK {
		t.Errorf("verzoek vanaf de eigen pagina moet slagen, kreeg %d", rr.Code)
	}
}

func TestBodySizeIsLimited(t *testing.T) {
	h := secured(t, config{})
	huge := `{"dutch":"` + strings.Repeat("a", maxBodyBytes) + `","hungarian":"b"}`
	if rr := do(h, "POST", "/api/concepts", "application/json", huge); rr.Code != http.StatusBadRequest {
		t.Errorf("te grote body: verwacht 400, kreeg %d", rr.Code)
	}
}

func TestBasicAuth(t *testing.T) {
	h := secured(t, config{AuthUser: "maarten", AuthPassword: "geheim"})
	if rr := do(h, "GET", "/api/state", "", ""); rr.Code != http.StatusUnauthorized || rr.Header().Get("WWW-Authenticate") == "" {
		t.Errorf("zonder inloggen: verwacht 401 met WWW-Authenticate, kreeg %d", rr.Code)
	}
	if rr := do(h, "GET", "/", "", "", func(r *http.Request) { r.SetBasicAuth("maarten", "fout") }); rr.Code != http.StatusUnauthorized {
		t.Errorf("verkeerd wachtwoord: verwacht 401, kreeg %d", rr.Code)
	}
	if rr := do(h, "GET", "/api/state", "", "", func(r *http.Request) { r.SetBasicAuth("maarten", "geheim") }); rr.Code != http.StatusOK {
		t.Errorf("goed wachtwoord: verwacht 200, kreeg %d", rr.Code)
	}
	if rr := do(h, "GET", "/healthz", "", ""); rr.Code != http.StatusOK {
		t.Errorf("/healthz moet zonder inloggen werken, kreeg %d", rr.Code)
	}
}

func TestSecurityHeaders(t *testing.T) {
	rr := do(secured(t, config{}), "GET", "/", "", "")
	for _, h := range []string{"Content-Security-Policy", "X-Content-Type-Options", "X-Frame-Options", "Referrer-Policy"} {
		if rr.Header().Get(h) == "" {
			t.Errorf("header %s ontbreekt", h)
		}
	}
}

func TestConfig(t *testing.T) {
	for addr, local := range map[string]bool{
		"127.0.0.1:8080": true, "localhost:8080": true, "[::1]:8080": true,
		":8080": false, "0.0.0.0:8080": false, "192.168.1.10:8080": false,
	} {
		if got := (config{Addr: addr}).localOnly(); got != local {
			t.Errorf("%s: localOnly = %v, verwacht %v", addr, got, local)
		}
	}

	t.Setenv("ADDR", "")
	t.Setenv("DATA_DIR", "/data")
	t.Setenv("AUTH_USER", "maarten")
	t.Setenv("AUTH_PASSWORD", "")
	if _, err := loadConfig(); err == nil {
		t.Error("alleen AUTH_USER zonder wachtwoord moet een fout geven")
	}
	t.Setenv("AUTH_PASSWORD", "geheim")
	cfg, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Addr != "127.0.0.1:8080" || cfg.wordsPath() != "/data/woorden.json" || !cfg.authEnabled() {
		t.Errorf("config: %+v", cfg)
	}
}
