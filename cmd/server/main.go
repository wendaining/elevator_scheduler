package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os_sp26_proj1/internal/api"
	"os_sp26_proj1/internal/elevator"
	"time"
)

func main() {
	const (
		defaultFloorCount         = 20
		defaultElevatorCount      = 5
		defaultTicksPerFloor      = 2
		defaultDoorBaseTicks      = 4
		defaultTickPerPassenger   = 1
		defaultEmergencyStopTicks = 20
		defaultAutoStepInterval   = 250 * time.Millisecond
	)

	dbPath := fmt.Sprintf("data/requests_%de_%df_scan_%d.db", defaultElevatorCount, defaultFloorCount, time.Now().Unix())

	config := elevator.SystemConfig{
		Floors:             defaultFloorCount,
		ElevatorCount:      defaultElevatorCount,
		TicksPerFloor:      defaultTicksPerFloor,
		DoorBaseTicks:      defaultDoorBaseTicks,
		TickPerPassenger:   defaultTickPerPassenger,
		EmergencyStopTicks: defaultEmergencyStopTicks,
		DatabasePath:       dbPath,
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
