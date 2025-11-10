package main

import (
	"context"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"services/generator-a/internal/generator"
	handler "services/generator-a/internal/http"
	sensorpb "services/ingest-b/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Default values — but now overridable through env variables.
const (
	DEF_SENSOR_TYPE = "temperature"
	DEF_ID1         = "A"
	DEF_ID2         = 1
	DEF_INITIAL_HZ  = 1.0
	DEF_PORT        = ":8081"
	DEF_B_GRPC      = "localhost:9090"
)

// small helper functions
func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getenvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil {
			return n
		}
	}
	return def
}

func getenvFloat(key string, def float64) float64 {
	if v := os.Getenv(key); v != "" {
		f, err := strconv.ParseFloat(v, 64)
		if err == nil {
			return f
		}
	}
	return def
}

func main() {
	// ---------------------------------------------
	// 1) Load settings (env overrides defaults)
	// ---------------------------------------------
	sensorType := getenv("SENSOR_TYPE", DEF_SENSOR_TYPE)
	id1 := getenv("ID1", DEF_ID1)
	id2 := getenvInt("ID2", DEF_ID2)
	initialHz := getenvFloat("HZ", DEF_INITIAL_HZ)
	port := getenv("PORT", DEF_PORT)
	bGrpc := getenv("B_GRPC", DEF_B_GRPC)

	// ---------------------------------------------
	// 2) gRPC connection to Microservice B
	// ---------------------------------------------
	conn, err := grpc.Dial(bGrpc, grpc.WithTransportCredentials(insecure.NewCredentials()))
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
		_, _ = stream.CloseAndRecv()
	}()

	// ---------------------------------------------
	// 3) Sensor generator loop
	// ---------------------------------------------
	loop := generator.NewLoop(id1, id2, sensorType, initialHz)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Each reading is streamed to B
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

	// ---------------------------------------------
	// 4) REST: change frequency
	// ---------------------------------------------
	h := handler.New(loop)
	e := handler.Router(h)

	errCh := make(chan error, 1)
	go func() { errCh <- e.Start(port) }()

	// ---------------------------------------------
	// 5) Graceful shutdown
	// ---------------------------------------------
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-sigs:
	case <-errCh:
	}

	shCtx, cancel2 := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel2()
	_ = e.Shutdown(shCtx)
}
