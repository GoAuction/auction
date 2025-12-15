package grpc

import (
	"context"
	"log"

	"google.golang.org/grpc"
)

func loggingInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	log.Printf("Received request: %s", info.FullMethod)
	resp, err := handler(ctx, req)
	if err != nil {
		log.Printf("Error handling request: %s", err)
	}
	return resp, err
}

func recoveryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic: %v", r)
		}
	}()
	return handler(ctx, req)
}
