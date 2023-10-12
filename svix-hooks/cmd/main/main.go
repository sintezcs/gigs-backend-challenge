package main

import (
    "context"
    "errors"
    "fmt"
    "github.com/caarlos0/env/v9"
    svix "github.com/svix/svix-webhooks/go"
    "log"
    "net/http"
    "os"
    "os/signal"
    "svix-hooks/pkg/api"
    "svix-hooks/pkg/config"
    "svix-hooks/pkg/models"
    "svix-hooks/pkg/services/hookService"
    "sync"
    "syscall"
)

// main function
func main() {
    // Load config from environment variables
    conf := &config.Config{}
    if err := env.ParseWithOptions(conf, config.ParserOptions); err != nil {
        // If the required environment variables are not set, exit with an error
        panic(err)
    }

    // Create and start the hook service
    var wg sync.WaitGroup
    ctx := context.Background()
    ctx, cancelWorkers := context.WithCancel(ctx)
    svixChannel := make(chan *models.Notification, conf.ChannelBuffer)
    svixRetryChannel := make(chan *models.Notification, conf.ChannelBuffer)
    svixClient := svix.New(conf.SvixApiKey, nil)
    service := hookService.New(conf, svixChannel, svixRetryChannel, &wg, svixClient)
    service.Start(ctx)

    // Create and set up the API handlers
    apiHandlers := api.New(conf, service)
    server := http.Server{Addr: fmt.Sprintf(":%d", conf.Port)}
    http.HandleFunc("/notification", apiHandlers.NotificationHandler)
    http.HandleFunc("/health", apiHandlers.HealthHandler)
    http.HandleFunc("/stats", apiHandlers.StatsHandler)

    // Start the HTTP server
    log.Printf("Starting server on port %d\n", conf.Port)
    go func() {
        err := server.ListenAndServe()
        if err != nil && !errors.Is(err, http.ErrServerClosed) {
            log.Printf("Error starting server: %s\n", err)
        }
    }()

    // Listen to SIGINT and SIGTERM signals
    stop := make(chan os.Signal, 1)
    signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
    <-stop

    // Shutdown the server gracefully if we receive a signal
    log.Println("Received shutdown signal, shutting down.")
    log.Println("Shutting down HTTP server...")
    err := server.Shutdown(ctx)
    if err != nil {
        log.Printf("Error shutting down server: %s\n", err)
    }
    log.Println("HTTP server shutdown completed")

    // Stop the hook service
    service.Stop(cancelWorkers)

    log.Println("Workers finished. App shutdown completed.")
}
