package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"services/generator-a/internal/generator"
	handler "services/generator-a/internal/http"
)

// ---- FIXED INSTANCE PROPERTIES (each Microservice A should have a fixed sensor type) ----
const (
	// Change these constants when you run another instance of Microservice A.
	// Example instances:
	//  - temperature sensor: SENSOR_TYPE="temperature", ID1="A", ID2=1
	//  - humidity sensor:    SENSOR_TYPE="humidity",    ID1="B", ID2=2
	//  - pressure sensor:    SENSOR_TYPE="pressure",    ID1="C", ID2=3
	SENSOR_TYPE = "temperature"
	ID1         = "A"
	ID2         = 1

	// initial frequency (readings per second)
	INITIAL_HZ = 1.0

	// HTTP server port (only to expose the one required endpoint)
	PORT = ":8081"
)

func main() {
	loop := generator.NewLoop(ID1, ID2, SENSOR_TYPE, INITIAL_HZ)

	// Minimal HTTP server exposing ONLY the required endpoint.
	h := handler.New(loop)
	e := handler.Router(h)

	srvErr := make(chan error, 1)
	go func() { srvErr <- e.Start(PORT) }()

	// Run the generator loop.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = loop.Run(ctx) }()

	// Graceful shutdown on signal or server error.
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-sigs:
	case <-srvErr:
	}

	_ = e.Shutdown(ctx)
}
