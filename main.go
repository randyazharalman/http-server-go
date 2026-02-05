package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/randyazharalman/http-server-go/internal/database"

	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	db             *sql.DB
	dbQueries      *database.Queries
	platform       string
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	dbUrl := os.Getenv("DB_URL")
	if dbUrl == "" {
		log.Fatal("DB_URL environment variable is not set")
	}

	platform := os.Getenv("PLATFORM")
	if platform == "" {
		log.Fatal("PLATFORM environment variable is not set")
	}

	db, err := sql.Open("postgres", dbUrl)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}

	defer db.Close()

	err = db.Ping()
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}
	log.Println("Successfully connected to database")

	dbQueries := database.New(db)

	apiCfg := &apiConfig{
		db:        db,
		dbQueries: dbQueries,
		platform:  platform,
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/api/healthz", healthzHandler)
	mux.HandleFunc("/api/users", apiCfg.usersHandler)
	mux.HandleFunc("/api/chirps", apiCfg.chirpsHandler)
	mux.HandleFunc("/api/chirps/{chirpID}", apiCfg.getChirpByIDHandler)
	mux.HandleFunc("/api/validate_chirp", validateChirpHandler)
	mux.HandleFunc("/admin/metrics", apiCfg.metricsHandler)
	mux.HandleFunc("/admin/reset", apiCfg.resetHandler)

	fileServer := http.FileServer(http.Dir("."))
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app", fileServer)))

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	log.Println("Server starting on http://localhost:8080")
	err = server.ListenAndServe()
	if err != nil {
		log.Fatalf("Error starting server: %v\n", err)
	}
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) metricsHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	hits := cfg.fileserverHits.Add(1)
	template := fmt.Sprintf(`<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`, hits)
	w.Write([]byte(template))
}
func (cfg *apiConfig) resetHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if cfg.platform != "dev" {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	err := cfg.dbQueries.DeleteAllUsers(context.Background())
	if err != nil {
		log.Printf("Error deleting users: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to reset database")
		return
	}

	cfg.fileserverHits.Store(0)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Hits: reset to 0"))
}

func healthzHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (cfg *apiConfig) usersHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		cfg.getUsersHandler(w, r)
	case http.MethodPost:
		cfg.createUsersHandler(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (cfg *apiConfig) createUsersHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	type requestBody struct {
		Email string `json:"email"`
	}

	decoder := json.NewDecoder(r.Body)
	params := requestBody{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	userParams := database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Email:     params.Email,
	}
	user, err := cfg.dbQueries.CreateUser(context.Background(), userParams)
	if err != nil {
		log.Printf("Error creating user: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to create user")
		return
	}

	respondWithJSON(w, http.StatusCreated, user)
}

func (cfg *apiConfig) getUsersHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {

	}
	users, err := cfg.dbQueries.GetAllUsers(context.Background())
	if err != nil {
		log.Printf("Error getting users: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to get users")
		return
	}

	respondWithJSON(w, http.StatusOK, users)
}

func (cfg *apiConfig) chirpsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		cfg.createChirpsHandler(w, r)
	case http.MethodGet:
		cfg.getChirpsHandler(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (cfg *apiConfig) createChirpsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	type requestBody struct {
		Body    string `json:"body"`
		User_id string `json:"user_id"`
	}

	decoder := json.NewDecoder(r.Body)
	params := requestBody{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	userUUID, err := uuid.Parse(params.User_id)
	if err != nil {
		log.Println("invalid user id:", err)
		return
	}
	chirpParams := database.CreateChirpParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Body:      params.Body,
		UserID:    userUUID,
	}

	chirp, err := cfg.dbQueries.CreateChirp(context.Background(), chirpParams)
	if err != nil {
		log.Printf("Error creating chirp: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to create chirp")
		return
	}

	respondWithJSON(w, http.StatusCreated, chirp)
}

func (cfg *apiConfig) getChirpsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	chirp, err := cfg.dbQueries.GetAllChirps(context.Background())
	if err != nil {
		log.Printf("Error getting chirps: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to get chirps")
		return
	}

	respondWithJSON(w, http.StatusOK, chirp)
}

func (cfg *apiConfig) getChirpByIDHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	chirpIDStr := r.PathValue("chirpID")
	if chirpIDStr == "" {
		respondWithError(w, http.StatusBadRequest, "Missing chirp ID")
		return
	}

	chirpID, err := uuid.Parse(chirpIDStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid chirp ID format")
		return
	}

	chirp, err := cfg.dbQueries.GetChirpByID(context.Background(), chirpID)
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusNotFound, "Chirp not found")
			return
		}
		log.Printf("Error getting chirp: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to get chirp")
		return
	}

	respondWithJSON(w, http.StatusOK, chirp)

}

func validateChirpHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	type requestBody struct {
		Body string `json:"body"`
	}

	decoder := json.NewDecoder(r.Body)
	params := requestBody{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid Request Body")
		return
	}

	if len(params.Body) > 140 {
		respondWithError(w, http.StatusBadRequest, "Chirp is too long")
		return
	}

	cleanBody := cleanProfanity(params.Body)

	respondWithJSON(w, http.StatusOK, map[string]string{"cleaned_body": cleanBody})

}

func cleanProfanity(text string) string {
	profaneWords := []string{"kerfuffle", "sharbert", "fornax"}
	words := strings.Split(text, " ")
	for i, word := range words {
		lowercaseWord := strings.ToLower(word)
		for _, profane := range profaneWords {
			if lowercaseWord == profane {
				words[i] = "****"
				break
			}
		}
	}

	return strings.Join(words, " ")
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	respondWithJSON(w, code, map[string]string{"error": msg})
}
func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(payload)
}
