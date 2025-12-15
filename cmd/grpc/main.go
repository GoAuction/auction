package main

import (
	"auction/infra/grpc"
	"auction/infra/postgres"
	"auction/pkg/config"
	itemv1 "auction/proto/gen"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	// Initialize logger
	zapConfig := zap.NewDevelopmentConfig()
	zapConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	logger, _ := zapConfig.Build()
	zap.ReplaceGlobals(logger)
	defer logger.Sync()

	zap.L().Info("Auction gRPC Service starting...")

	appConfig := config.Read()

	grpcServer, err := grpc.NewServer(appConfig)
	if err != nil {
		zap.L().Error("failed to create grpc server", zap.Error(err))
		os.Exit(1)
	}

	// Initialize PostgreSQL repository
	pgRepository := postgres.NewPgRepository(
		appConfig.PostgresHost,
		appConfig.PostgresDatabase,
		appConfig.PostgresUsername,
		appConfig.PostgresPassword,
		appConfig.PostgresPort,
	)

	itemService := grpc.NewItemServiceServer(pgRepository)
	itemv1.RegisterItemServiceServer(grpcServer.GetGRPCServer(), itemService)

	zap.L().Info("starting gRPC server...", zap.String("port", appConfig.GRPCPort))
	go func() {
		if err := grpcServer.Start(); err != nil {
			zap.L().Error("failed to start grpc server", zap.Error(err))
			os.Exit(1)
		}
	}()

	gracefulShutdown(grpcServer)
}

func gracefulShutdown(grpcServer *grpc.Server) {
	// Create channel for shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Wait for shutdown signal
	<-sigChan
	zap.L().Info("Shutting down server...")

	// Shutdown with 5 second timeout
	if err := grpcServer.GracefulStop(); err != nil {
		zap.L().Error("Error during server shutdown", zap.Error(err))
	}

	zap.L().Info("Server gracefully stopped")
}
