package grpcservice

import (
	"context"
	"log"
	"net"
	"time"

	"google.golang.org/grpc"
)

type Server struct {
}

func StartGRPCServer(port string) {
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Printf("[gRPC ERROR] Failed to listen on port %s: %v", port, err)
		return
	}
	s := grpc.NewServer()
	log.Printf("[gRPC WARNING] gRPC server is a stub: no services are registered on port %s", port)
	go func() {
		if err := s.Serve(lis); err != nil {
			log.Printf("[gRPC ERROR] Failed to serve: %v", err)
		}
	}()
}

func (s *Server) GetDriverLocation(ctx context.Context, driverID string) (map[string]interface{}, error) {
	return map[string]interface{}{
		"driver_id":    driverID,
		"latitude":     18.5204,
		"longitude":    73.8567,
		"last_updated": time.Now().Format(time.RFC3339),
	}, nil
}
