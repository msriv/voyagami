// Command api runs the Voyagami HTTP API.
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
	// Distroless has no zoneinfo; trip validation loads IANA zones.
	_ "time/tzdata"

	"github.com/msriv/voyagami/services/api/internal/config"
	"github.com/msriv/voyagami/services/api/internal/logging"
	"github.com/msriv/voyagami/services/api/internal/server"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "api:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load(os.Getenv)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	logger := logging.New(os.Stdout, cfg.LogLevel, cfg.Env)

	deps, err := buildSignIn(context.Background(), cfg, logger)
	if err != nil {
		return fmt.Errorf("sign-in: %w", err)
	}
	defer deps.close()

	srv := &http.Server{
		Addr:              ":" + strconv.Itoa(cfg.Port),
		Handler:           http.TimeoutHandler(server.NewHandler(logger, deps.verifier, deps.options...), cfg.RequestTimeout, `{"code":"timeout","message":"The request took too long."}`),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       cfg.RequestTimeout,
		WriteTimeout:      cfg.RequestTimeout + 5*time.Second,
		IdleTimeout:       2 * time.Minute,
		MaxHeaderBytes:    1 << 16,
	}

	// Cloud Run sends SIGTERM and allows a short window to finish in-flight work.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errc := make(chan error, 1)
	go func() {
		logger.Info("listening", "port", cfg.Port)
		errc <- srv.ListenAndServe()
	}()

	select {
	case err := <-errc:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve: %w", err)
		}
		return nil
	case <-ctx.Done():
	}

	logger.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	logger.Info("stopped")
	return nil
}
