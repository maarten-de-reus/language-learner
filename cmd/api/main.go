package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"

	"hungarian-language-learner/internal/quiz"
	"hungarian-language-learner/internal/storage"
)

const (
	WordsPath    = "woorden.json"
	ProgressPath = "voortgang.json"
	RunSize      = 10
	Addr         = ":8080"
)

//go:embed web
var webFiles embed.FS

func main() {
	lib, err := storage.LoadLibrary(WordsPath, ProgressPath)
	if err != nil {
		log.Fatalf("Fout bij laden data: %v", err)
	}

	s := &server{lib: lib, wordsPath: WordsPath, session: quiz.New(lib, ProgressPath, RunSize)}

	log.Printf("🌍 Web frontend gestart op http://localhost%s", Addr)
	log.Fatal(http.ListenAndServe(Addr, s.routes()))
}

func (s *server) routes() *http.ServeMux {
	mux := http.NewServeMux()
	static, _ := fs.Sub(webFiles, "web")
	mux.Handle("/", http.FileServer(http.FS(static)))

	// Oefenen
	mux.HandleFunc("GET /api/state", s.handle(func() {}))
	mux.HandleFunc("POST /api/answer", s.handleAnswer)
	mux.HandleFunc("POST /api/next", s.handle(func() { s.session.Next() }))
	mux.HandleFunc("POST /api/run", s.handle(func() { s.session.NewRun() }))

	// Beheer
	mux.HandleFunc("GET /api/concepts", s.handleListConcepts)
	mux.HandleFunc("POST /api/concepts/check", s.handleCheckConcept)
	mux.HandleFunc("POST /api/concepts", s.handleAddConcept)
	mux.HandleFunc("PUT /api/concepts/{id}", s.handleUpdateConcept)
	return mux
}
