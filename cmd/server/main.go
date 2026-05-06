package main

import (
	"log"
	"net/http"
	"os_sp26_proj1/internal/api"
	"os_sp26_proj1/internal/elevator"
)

func main() {
	system, err := elevator.NewSystem(20, 5)
	if err != nil {
		log.Fatalf("failed to create elevator system: %v", err)
	}

	server := &api.Server{System: system}

	mux := http.NewServeMux()
	server.RegisterRoutes(mux)

	addr := ":8080"
	log.Printf("server listening on http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
