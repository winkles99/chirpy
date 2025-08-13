package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"unicode"
)

// Profanity list
var profanity = []string{"kerfuffle", "sharbert", "fornax"}

// Helper to respond with JSON
func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(payload)
}

// Strip punctuation for word matching
func stripPunct(word string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			return r
		}
		return -1
	}, word)
}

// Replace profane words in a chirp
func cleanChirp(body string) string {
	words := strings.Split(body, " ")
	for i, word := range words {
		lowerWord := strings.ToLower(stripPunct(word))
		for _, profane := range profanity {
			if lowerWord == profane {
				words[i] = "****"
				break
			}
		}
	}
	return strings.Join(words, " ")
}

// API config to track hits
type apiConfig struct {
	fileserverHits atomic.Int32
}

// Request struct
type chirpRequest struct {
	Body string `json:"body"`
}

// Error response struct
type errorResponse struct {
	Error string `json:"error"`
}

// Middleware to increment hit counter
func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

// Metrics handler
func (cfg *apiConfig) handlerMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	hits := cfg.fileserverHits.Load()
	html := fmt.Sprintf(`
<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>
`, hits)
	fmt.Fprint(w, html)
}

// Reset metrics
func (cfg *apiConfig) handlerReset(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits.Store(0)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Hits reset to 0"))
}

// Validate chirp handler
func (cfg *apiConfig) handlerValidateChirp(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req chirpRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithJSON(w, http.StatusBadRequest, errorResponse{Error: "Something went wrong"})
		return
	}

	if len(req.Body) > 140 {
		respondWithJSON(w, http.StatusBadRequest, errorResponse{Error: "Chirp is too long"})
		return
	}

	cleaned := cleanChirp(req.Body)
	respondWithJSON(w, http.StatusOK, map[string]string{"cleaned_body": cleaned})
}

func main() {
	apiCfg := &apiConfig{}
	mux := http.NewServeMux()

	// File server
	fsHandler := http.StripPrefix("/app/", http.FileServer(http.Dir(".")))
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(fsHandler))

	// Health check
	mux.HandleFunc("/api/healthz", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Admin endpoints
	mux.HandleFunc("/admin/metrics", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		apiCfg.handlerMetrics(w, r)
	})

	mux.HandleFunc("/admin/reset", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		apiCfg.handlerReset(w, r)
	})

	// Root redirect
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/app/", http.StatusFound)
	})

	// Chirp validation
	mux.HandleFunc("/api/validate_chirp", apiCfg.handlerValidateChirp)

	server := http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	fmt.Println("Server listening on http://localhost:8080")
	server.ListenAndServe()
}
