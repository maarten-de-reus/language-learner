package main

import (
	"context"
	"embed"
	"errors"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"hungarian-language-learner/internal/quiz"
	"hungarian-language-learner/internal/storage"
)

const (
	RunSize       = 10
	ExtraNewWords = 10 // per klik op "Nog 10 nieuwe woorden"
)

//go:embed web
var webFiles embed.FS

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("Ongeldige configuratie: %v", err)
	}

	lib, err := storage.LoadLibrary(cfg.wordsPath(), cfg.progressPath())
	if err != nil {
		log.Fatalf("Fout bij laden data: %v", err)
	}

	settings, err := storage.LoadSettings(cfg.settingsPath())
	if err != nil {
		log.Fatalf("Fout bij laden instellingen: %v", err)
	}

	session := quiz.New(lib, cfg.progressPath(), RunSize)
	session.LogReviewsTo(storage.NewReviewLog(cfg.reviewsPath()))
	session.SetNewPerDay(settings.NewPerDay)
	s := &server{
		lib: lib, session: session, settings: settings,
		wordsPath: cfg.wordsPath(), reviewsPath: cfg.reviewsPath(), settingsPath: cfg.settingsPath(),
	}
	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           withSecurity(s.routes(), cfg),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	if !cfg.localOnly() && !cfg.authEnabled() {
		log.Printf("⚠️  Let op: %s is bereikbaar vanaf het netwerk zonder wachtwoord. Zet AUTH_USER en AUTH_PASSWORD.", cfg.Addr)
	}

	// Netjes afsluiten bij Ctrl+C of een stopsignaal (bijv. van systemd of Docker):
	// lopende verzoeken, en dus ook het opslaan, worden eerst afgemaakt.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("Fout bij afsluiten: %v", err)
		}
	}()

	log.Printf("🌍 Woordtrainer gestart op http://%s (data: %s)", cfg.Addr, cfg.DataDir)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
	log.Print("Gestopt")
}

func (s *server) routes() *http.ServeMux {
	mux := http.NewServeMux()
	static, _ := fs.Sub(webFiles, "web")
	mux.Handle("/", http.FileServer(http.FS(static)))
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ok\n"))
	})

	// Oefenen
	mux.HandleFunc("GET /api/state", s.handle(func() {}))
	mux.HandleFunc("POST /api/answer", s.handleAnswer)
	mux.HandleFunc("POST /api/next", s.handle(func() { s.session.Next() }))
	mux.HandleFunc("POST /api/run", s.handle(func() { s.session.NewRun() }))
	mux.HandleFunc("POST /api/new/extra", s.handle(func() { s.session.AddExtraNew(ExtraNewWords) }))
	mux.HandleFunc("GET /api/stats", s.handleStats)
	mux.HandleFunc("GET /api/settings", s.handleGetSettings)
	mux.HandleFunc("PUT /api/settings", s.handlePutSettings)

	// Beheer
	mux.HandleFunc("GET /api/concepts", s.handleListConcepts)
	mux.HandleFunc("POST /api/concepts/check", s.handleCheckConcept)
	mux.HandleFunc("POST /api/concepts", s.handleAddConcept)
	mux.HandleFunc("PUT /api/concepts/{id}", s.handleUpdateConcept)
	mux.HandleFunc("DELETE /api/concepts/{id}", s.handleDeleteConcept)
	mux.HandleFunc("POST /api/import/check", s.handleImportCheck)
	mux.HandleFunc("POST /api/import", s.handleImport)
	return mux
}
