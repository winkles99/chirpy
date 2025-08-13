package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"
)

// api configuration to manage file server metrics
type apiConfig struct {
	fileserverHits atomic.Int32
}

// request struct for chirp validation
type chirpRequest struct {
	Body string `json:"body"`
}

// response struct for error messages
type errorResponse struct {
	Error string `json:"error"`
}

// response struct for valid chirps
type validResponse struct {
	Valid bool `json:"valid"`
}

// Middleware to increment hit counter
func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1) // thread-safe increment
		next.ServeHTTP(w, r)
	})
}

// Hit Shower
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

// reset metrics
func (cfg *apiConfig) handlerReset(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits.Store(0)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Hits reset to 0"))
}

// ✅ Now a method on *apiConfig
func (cfg *apiConfig) handlerValidateChirp(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req chirpRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errorResponse{Error: "Something went wrong"})
		return
	}

	if len(req.Body) > 140 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errorResponse{Error: "Chirp is too long"})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(validResponse{Valid: true})
}

func main() {
	apiCfg := &apiConfig{}
	mux := http.NewServeMux()

	// file server
	fsHandler := http.StripPrefix("/app/", http.FileServer(http.Dir(".")))
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(fsHandler))

	// readiness endpoint
	mux.HandleFunc("/api/healthz", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

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

	mux.HandleFunc("/api/validate_chirp", apiCfg.handlerValidateChirp)

	server := http.Server{
		Addr:    ":8080",
		Handler: mux,
	}
	server.ListenAndServe()
}
