package main

import (
	"log"
	"net/http"
)

func main() {
	mux := http.NewServeMux()

    mux.HandleFunc("/healthz", healthzHandler)

    fileServer := http.FileServer(http.Dir("."))
    mux.Handle("/app/",http.StripPrefix("/app", fileServer))

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	log.Println("Server starting on http://localhost:8080")
	err := server.ListenAndServe()
	if err != nil {
		log.Fatalf("Error starting server: %v\n", err)
	}
}

func healthzHandler(w http.ResponseWriter, r *http.Request){
    w.Header().Set("Content-Type", "text/plain; charset=utf-8")
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("OK"))
}