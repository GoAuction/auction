package grpc

import (
	"auction/pkg/config"
	"fmt"
	"net"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

type Server struct {
	server   *grpc.Server
	listener net.Listener
}

func (s *Server) GetGRPCServer() grpc.ServiceRegistrar {
	return s.server
}

func NewServer(cfg *config.AppConfig) (*Server, error) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.GRPCPort))
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %w", err)
	}

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			loggingInterceptor,
			recoveryInterceptor,
		),
	)

	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)

	return &Server{
		server:   grpcServer,
		listener: lis,
	}, nil
}

func (s *Server) Start() error {
	zap.L().Info("gRPC server started successfully",
		zap.String("address", s.listener.Addr().String()))
	return s.server.Serve(s.listener)
}

func (s *Server) GetListener() net.Listener {
	return s.listener
}

func (s *Server) GracefulStop() error {
	s.server.GracefulStop()
	return s.listener.Close()
}
