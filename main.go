package main

import (
	"auction/app/item"
	"auction/infra/postgres"
	"auction/pkg/config"
	"auction/pkg/httperror"
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type Request any
type Response any

type HandlerInterface[R Request, Res Response] interface {
	Handle(ctx context.Context, req *R) (*Res, error)
}

func handle[R Request, Res Response](handler HandlerInterface[R, Res]) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req R

		if err := c.BodyParser(&req); err != nil && !errors.Is(err, fiber.ErrUnprocessableEntity) {
			return writeError(c, httperror.BadRequest(
				"request.invalid_body",
				"Invalid body",
				fiber.Map{"error": err.Error()},
			))
		}

		if err := c.ParamsParser(&req); err != nil {
			return writeError(c, httperror.BadRequest(
				"request.invalid_path_params",
				"Invalid path params",
				fiber.Map{"error": err.Error()},
			))
		}

		if err := c.QueryParser(&req); err != nil {
			return writeError(c, httperror.BadRequest(
				"request.invalid_query_params",
				"Invalid query params",
				fiber.Map{"error": err.Error()},
			))
		}

		if err := c.ReqHeaderParser(&req); err != nil {
			return writeError(c, httperror.BadRequest(
				"request.invalid_headers",
				"Invalid headers",
				fiber.Map{"error": err.Error()},
			))
		}

		ctx := c.UserContext()

		res, err := handler.Handle(ctx, &req)
		if err != nil {
			return writeError(c, err)
		}

		return c.JSON(res)
	}
}

func main() {
	appConfig := config.Read()
	defer zap.L().Sync()
	zap.L().Info("app starting...")
	zap.L().Info("app config", zap.Any("appConfig", appConfig))

	app := fiber.New(fiber.Config{
		IdleTimeout:  5 * time.Second,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		Concurrency:  256 * 1024,
	})

	pgRepository := postgres.NewPgRepository(
		appConfig.PostgresHost,
		appConfig.PostgresDatabase,
		appConfig.PostgresUsername,
		appConfig.PostgresPassword,
		appConfig.PostgresPort,
	)

	createItemHadler := item.NewCreateItemHandler(pgRepository)

	publicRoutes := app.Group("/api/v1")
	publicRoutes.Get("/items", handle[item.CreateItemRequest, item.CreateItemResponse](createItemHadler))
	publicRoutes.Get("/items/:item", handle[item.CreateItemRequest, item.CreateItemResponse](createItemHadler))
	publicRoutes.Post("/items", handle[item.CreateItemRequest, item.CreateItemResponse](createItemHadler))
	publicRoutes.Put("/items/:item", handle[item.CreateItemRequest, item.CreateItemResponse](createItemHadler))
	publicRoutes.Delete("/items/:item", handle[item.CreateItemRequest, item.CreateItemResponse](createItemHadler))

	// Start server in a goroutine
	go func() {
		if err := app.Listen(fmt.Sprintf("0.0.0.0:%s", appConfig.Port)); err != nil {
			zap.L().Error("Failed to start server", zap.Error(err))
			os.Exit(1)
		}
	}()

	zap.L().Info("Server started on port", zap.String("port", appConfig.Port))

	gracefulShutdown(app)
}

func gracefulShutdown(app *fiber.App) {
	// Create channel for shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Wait for shutdown signal
	<-sigChan
	zap.L().Info("Shutting down server...")

	// Shutdown with 5 second timeout
	if err := app.ShutdownWithTimeout(5 * time.Second); err != nil {
		zap.L().Error("Error during server shutdown", zap.Error(err))
	}

	zap.L().Info("Server gracefully stopped")
}

func writeError(c *fiber.Ctx, err error) error {
	var httpErr *httperror.Error
	if errors.As(err, &httpErr) {
		payload := fiber.Map{
			"code":    httpErr.Code,
			"message": httpErr.Message,
		}

		if httpErr.Details != nil {
			payload["details"] = httpErr.Details
		}

		if httpErr.Status >= fiber.StatusInternalServerError {
			zap.L().Error("Handler returned server error", zap.String("code", httpErr.Code), zap.Error(httpErr))
		} else {
			zap.L().Warn("Handler returned client error", zap.String("code", httpErr.Code), zap.Error(httpErr))
		}

		return c.Status(httpErr.Status).JSON(payload)
	}

	var fiberErr *fiber.Error
	if errors.As(err, &fiberErr) {
		zap.L().Warn("Fiber validation error", zap.String("message", fiberErr.Message), zap.Error(err))
		return c.Status(fiberErr.Code).JSON(fiber.Map{
			"code":    "request.invalid",
			"message": fiberErr.Message,
		})
	}

	zap.L().Error("Unhandled error", zap.Error(err))
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
		"code":    "internal_server_error",
		"message": "Internal server error.",
	})
}
