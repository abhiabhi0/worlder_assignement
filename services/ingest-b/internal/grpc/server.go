package ingestgrpc

import (
	"log"
	"net"
	"time"

	"services/ingest-b/internal/repo"
	sensorpb "services/ingest-b/proto"

	"google.golang.org/grpc"
)

type IngestServer struct {
	sensorpb.UnimplementedIngestServiceServer
	Store *repo.Store
}

func New(store *repo.Store) *IngestServer { return &IngestServer{Store: store} }

func (s *IngestServer) StreamReadings(stream sensorpb.IngestService_StreamReadingsServer) error {
	var batch []repo.Reading
	const flushN = 200

	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		err := s.Store.Insert(stream.Context(), batch)
		batch = batch[:0]
		return err
	}

	for {
		msg, err := stream.Recv()
		if err != nil {
			// client closed or stream ended
			if err := flush(); err != nil {
				return err
			}
			return stream.SendAndClose(&sensorpb.IngestAck{Count: 0})
		}
		r := repo.Reading{
			Value:      msg.Value,
			SensorType: msg.SensorType,                            // generated Go name
			ID1:        msg.Id1,                                   // generated Go name
			ID2:        int(msg.Id2),                              // generated Go name
			Timestamp:  time.UnixMilli(msg.TimestampUnixMs).UTC(), // generated Go name
		}
		batch = append(batch, r)
		if len(batch) >= flushN {
			if err := flush(); err != nil {
				return err
			}
		}
	}
}

func StartGRPCServer(s *grpc.Server, addr string) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("grpc listen: %v", err)
	}
	log.Printf("gRPC listening on %s", addr)
	if err := s.Serve(l); err != nil {
		log.Fatalf("grpc serve: %v", err)
	}
}
