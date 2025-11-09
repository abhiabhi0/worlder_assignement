package main

import (
	"log"
	"net"
	"net/http"

	"google.golang.org/grpc"

	"services/ingest-b/internal/db"
	inggrpc "services/ingest-b/internal/grpc"
	httph "services/ingest-b/internal/http"
	"services/ingest-b/internal/repo"
	sensorpb "services/ingest-b/proto"
)

func main() {
	// DB
	gdb := db.ConnectAndMigrate()
	store := repo.NewStore(gdb)

	// HTTP (REST) on :8080
	h := httph.New(store)
	httpSrv := httph.Router(h)
	go func() {
		log.Println("HTTP listening on :8080")
		if err := httpSrv.Start(":8080"); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http failed: %v", err)
		}
	}()

	// gRPC on :9090
	gsrv := grpc.NewServer()
	ing := inggrpc.New(store)
	sensorpb.RegisterIngestServiceServer(gsrv, ing)

	lis, err := net.Listen("tcp", ":9090")
	if err != nil {
		log.Fatalf("grpc listen: %v", err)
	}
	log.Println("gRPC listening on :9090")
	if err := gsrv.Serve(lis); err != nil {
		log.Fatalf("grpc serve: %v", err)
	}
}
