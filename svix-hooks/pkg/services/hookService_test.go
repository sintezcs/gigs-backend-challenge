package services

import (
    "context"
    "github.com/caarlos0/env/v9"
    svix "github.com/svix/svix-webhooks/go"
    "log"
    "os"
    "svix-hooks/pkg/config"
    "svix-hooks/pkg/models"
    "svix-hooks/pkg/services/hookService"
    "sync"
    "testing"
)

func setup(ctx context.Context, wg *sync.WaitGroup) (*hookService.HookService, *config.Config) {
    // set the required environment variables
    err := os.Setenv("SVIX_API_KEY", "test")
    if err != nil {
        log.Printf("Error setting environment variable: %s\n", err)
        return nil, nil
    }
    err = os.Setenv("SVIX_APP_ID", "test")
    if err != nil {
        log.Printf("Error setting environment variable: %s\n", err)
        return nil, nil
    }
    // load the configuration
    conf := &config.Config{}
    if err := env.ParseWithOptions(conf, config.ParserOptions); err != nil {
        panic(err)
    }
    // create the hook service
    svixChannel := make(chan *models.Notification, conf.ChannelBuffer)
    svixRetryChannel := make(chan *models.Notification, conf.ChannelBuffer)
    svixClient := svix.New(conf.SvixApiKey, nil)
    service := hookService.New(conf, svixChannel, svixRetryChannel, wg, svixClient)
    return service, conf
}

func teardown() {
    log.Printf("teardown")
}

// TestHookService_RunWorkers tests the GetStats method of the HookService
// It creates a HookService instance and starts it.
// 1. Checks that the number of active workers is equal to the number of workers configured
// 2. Stops the service
// 3. Checks that the number of active workers is zero
func TestHookService_GetStats(t *testing.T) {
    var wg sync.WaitGroup
    ctx := context.Background()
    ctx, cancel := context.WithCancel(ctx)

    service, conf := setup(ctx, &wg)
    defer teardown()

    service.Start(ctx)
    stats := service.GetStats()
    log.Printf("Stats: %v\n", stats)
    if stats == nil {
        t.Errorf("Stats is nil")
    }
    messagesProcessed, ok := stats["messagesProcessed"].(uint64)
    if !ok {
        t.Errorf("MessagesProcessed is not uint64")
    }
    if messagesProcessed != 0 {
        t.Errorf("MessagesProcessed is not 0")
    }
    numWorkers, ok := stats["numWorkers"].(int32)
    if !ok {
        t.Errorf("NumWorkers is not int")
    }
    if numWorkers != conf.NumWorkers {
        t.Errorf("NumWorkers is not %d", conf.NumWorkers)
    }
    if stats["lastMessageProcessed"] != "" {
        t.Errorf("LastMessageProcessedTime is not empty")
    }
    service.Stop(cancel)
    stats = service.GetStats()
    log.Printf("Stats: %v\n", stats)
    numWorkers, _ = stats["numWorkers"].(int32)
    if numWorkers != 0 {
        t.Errorf("NumWorkers is not zero after shutting down the service")
    }
}
