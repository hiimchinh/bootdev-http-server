package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) printHits() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "text/html")
		io.WriteString(w, fmt.Sprintf(`<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`, cfg.fileserverHits.Load()))
	})
}
func (cfg *apiConfig) resetHits() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Store(0)
	})
}

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, "OK")
}

type parameters struct {
	Body string `json:"body"`
}

type validRes struct {
	Valid bool `json:"valid"`
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	type errRes struct {
		Error string `json:"error"`
	}
	response := errRes{
		Error: msg,
	}

	body, err := json.Marshal(response)
	if err != nil {
		respondWithError(w, 500, "Something went wrong")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(body)

}

func respondWithJSON(w http.ResponseWriter, code int, payload any) {
	type validResponse struct {
		Valid bool `json:"valid"`
	}

	body, err := json.Marshal(payload)
	if err != nil {
		respondWithError(w, 400, "Something went wrong")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(body)

}

func validateChirpHandler(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	params := parameters{}

	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, 500, "Something went wrong")

		return
	}
	if len(params.Body) > 140 {
		respondWithError(w, 400, "Chirp is too long")
		return

	}

	valid := validRes{
		Valid: true,
	}
	respondWithJSON(w, 200, valid)

}

func main() {
	mux := http.NewServeMux()
	config := &apiConfig{}
	mux.Handle("/app/", config.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(".")))))
	mux.HandleFunc("GET /api/healthz", healthCheckHandler)
	mux.Handle("GET /admin/metrics", config.printHits())
	mux.Handle("POST /admin/reset", config.resetHits())
	mux.HandleFunc("POST /api/validate_chirp", validateChirpHandler)
	server := &http.Server{Addr: ":8080", Handler: mux}
	server.ListenAndServe()
}
