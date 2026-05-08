package main

import (
	"log"
	"net/http"
	"os_sp26_proj1/internal/api"
	"os_sp26_proj1/internal/elevator"
)

func main() {
	const (
		defaultFloorCount       = 20
		defaultElevatorCount    = 5
		defaultTicksPerFloor    = 5
		defaultDoorBaseTicks    = 2
		defaultTickPerPassenger = 1
	)

	system, err := elevator.NewSystemWithDatabase(
		defaultFloorCount,
		defaultElevatorCount,
		defaultTicksPerFloor,
		defaultDoorBaseTicks,
		defaultTickPerPassenger,
		"data/requests.db",
	)
	if err != nil {
		log.Fatalf("failed to create elevator system: %v", err)
	}
	defer system.Close()

	server := &api.Server{System: system}

	mux := http.NewServeMux()
	server.RegisterRoutes(mux)

	addr := ":8080"
	log.Printf("server listening on http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
