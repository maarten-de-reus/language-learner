package main

import (
	"log"
	"net/http"
	"time"

	"hungarian-language-learner/internal/catalog"
	"hungarian-language-learner/internal/stats"
	"hungarian-language-learner/internal/storage"
)

func (s *server) handleStats(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	reviews, skipped := s.readReviews()
	rep := stats.Compute(s.lib, reviews, time.Now(), s.settings.NewPerDay)
	rep.Skipped = skipped
	writeJSON(w, http.StatusOK, rep)
}

// entries geeft alle concepten voor het beheer, met per concept de tellingen uit het antwoordenlog
func (s *server) entries() []catalog.Entry {
	reviews, _ := s.readReviews()
	counts := stats.PerConcept(reviews)
	entries := catalog.Entries(s.lib)
	for i := range entries {
		c := counts[entries[i].ID]
		entries[i].Attempts, entries[i].Wrong = c.Attempts, c.Wrong
	}
	return entries
}

// readReviews leest het antwoordenlog. Een leesfout maakt de statistiek leeg, maar breekt niets.
func (s *server) readReviews() ([]storage.Review, int) {
	if s.reviewsPath == "" {
		return nil, 0
	}
	reviews, skipped, err := storage.ReadReviews(s.reviewsPath)
	if err != nil {
		log.Printf("Antwoordenlog niet gelezen: %v", err)
	}
	return reviews, skipped
}
