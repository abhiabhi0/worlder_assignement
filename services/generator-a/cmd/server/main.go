package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"services/generator-a/internal/generator"
	handler "services/generator-a/internal/http"
	sensorpb "services/ingest-b/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// ---- FIXED INSTANCE PROPERTIES (each Microservice A runs one sensor type) ----
const (
	SENSOR_TYPE = "temperature"
	ID1         = "A"
	ID2         = 1

	INITIAL_HZ = 1.0              // initial readings/sec
	PORT       = ":8081"          // A's REST port
	B_GRPC     = "localhost:9090" // B's gRPC address
)

func main() {
	// 1) Dial Microservice B (gRPC)
	conn, err := grpc.Dial(B_GRPC, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	client := sensorpb.NewIngestServiceClient(conn)
	stream, err := client.StreamReadings(context.Background())
	if err != nil {
		panic(err)
	}
	defer func() {
		// close stream on exit (ignore ack here)
		_, _ = stream.CloseAndRecv()
	}()

	// 2) Build generator loop that sends each reading to B
	loop := generator.NewLoop(ID1, ID2, SENSOR_TYPE, INITIAL_HZ)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Use RunWithSink (see section 2 to add it) so each tick is sent to B
	go func() {
		_ = loop.RunWithSink(ctx, func(r generator.Reading) {
			_ = stream.Send(&sensorpb.SensorReading{
				Value:           r.Value,
				SensorType:      r.SensorType,
				Id1:             r.ID1,
				Id2:             int32(r.ID2),
				TimestampUnixMs: r.Timestamp.UnixMilli(),
			})
		})
	}()

	// 3) Minimal REST server to change frequency (as required)
	h := handler.New(loop)
	e := handler.Router(h)

	// start REST
	errCh := make(chan error, 1)
	go func() { errCh <- e.Start(PORT) }()

	// 4) Graceful shutdown on signal or server error
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-sigs:
	case <-errCh:
	}

	// shut down HTTP cleanly
	shCtx, cancel2 := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel2()
	_ = e.Shutdown(shCtx)
}
