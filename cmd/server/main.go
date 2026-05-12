package main

import (
	"context"
	"log"
	"net/http"
	"os_sp26_proj1/internal/api"
	"os_sp26_proj1/internal/elevator"
	"time"
)

func main() {
	const (
		defaultFloorCount       = 20
		defaultElevatorCount    = 5
		defaultTicksPerFloor    = 5
		defaultDoorBaseTicks    = 2
		defaultTickPerPassenger = 1
		defaultDatabasePath     = "data/requests.db"
		defaultAutoStepInterval = 250 * time.Millisecond
	)

	config := elevator.SystemConfig{
		Floors:           defaultFloorCount,
		ElevatorCount:    defaultElevatorCount,
		TicksPerFloor:    defaultTicksPerFloor,
		DoorBaseTicks:    defaultDoorBaseTicks,
		TickPerPassenger: defaultTickPerPassenger,
		DatabasePath:     defaultDatabasePath,
	}

	system, err := elevator.NewSystem(config)
	if err != nil {
		log.Fatalf("failed to create elevator system: %v", err)
	}
	defer system.Close()

	server := api.NewServer(system, config)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	system.StartElevatorRunners(ctx)
	server.StartAutoStep(ctx, defaultAutoStepInterval)

	mux := http.NewServeMux()
	server.RegisterRoutes(mux)

	addr := ":8080"
	log.Printf("server listening on http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
