package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rafikurnia/measurement-measurer/api"
	"github.com/rafikurnia/measurement-measurer/utils/logger"
)

func main() {
	log.SetFlags(0)

	router, err := api.SetupRouter()
	if err != nil {
		log.Println(logger.Entry{
			Severity:  "CRITICAL",
			Message:   fmt.Errorf("api.SetupRouter -> %w", err).Error(),
			Component: "main",
		})
	}

	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Println(logger.Entry{
				Severity:  "CRITICAL",
				Message:   fmt.Errorf("srv.ListenAndServe -> %w", err).Error(),
				Component: "main",
			})
		}
	}()
	log.Println(logger.Entry{
		Severity:  "INFO",
		Message:   "Server started",
		Component: "main",
	})

	signal := <-done
	log.Println(logger.Entry{
		Severity:  "INFO",
		Message:   fmt.Sprintf("Received signal: %s", signal.String()),
		Component: "main",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		log.Println(logger.Entry{
			Severity:  "INFO",
			Message:   "Cancelling context",
			Component: "main",
		})
		cancel()
	}()

	if err := srv.Shutdown(ctx); err != nil {
		log.Println(logger.Entry{
			Severity:  "CRITICAL",
			Message:   fmt.Errorf("srv.Shutdown -> %w", err).Error(),
			Component: "main",
		})
	}
	log.Println(logger.Entry{
		Severity:  "INFO",
		Message:   "Server Exited Properly",
		Component: "main",
	})
}
