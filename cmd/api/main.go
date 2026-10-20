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
	wordsPool, conceptsDB, err := storage.LoadData(WordsPath, ProgressPath)
	if err != nil {
		log.Fatalf("Fout bij laden data: %v", err)
	}

	s := &server{session: quiz.New(wordsPool, conceptsDB, ProgressPath, RunSize)}

	static, _ := fs.Sub(webFiles, "web")
	http.Handle("/", http.FileServer(http.FS(static)))
	http.HandleFunc("GET /api/state", s.handle(func() {}))
	http.HandleFunc("POST /api/answer", s.handleAnswer)
	http.HandleFunc("POST /api/next", s.handle(func() { s.session.Next() }))
	http.HandleFunc("POST /api/run", s.handle(func() { s.session.NewRun() }))

	log.Printf("🌍 Web frontend gestart op http://localhost%s", Addr)
	log.Fatal(http.ListenAndServe(Addr, nil))
}
