package hookService

import (
    "context"
    "errors"
    svix "github.com/svix/svix-webhooks/go"
    "golang.org/x/time/rate"
    "log"
    "net/http"
    "svix-hooks/pkg/config"
    "svix-hooks/pkg/models"
    "sync"
    "sync/atomic"
    "time"
)

// HookService struct holds the configuration and state of the hook service
type HookService struct {
    Config                   *config.Config
    SvixChannel              chan *models.Notification
    svixRetryChannel         chan *models.Notification
    SvixClient               *svix.Svix
    waitGroup                *sync.WaitGroup
    messagesProcessed        uint64
    activeWorkers            int32
    lastMessageProcessedTime string
    rateLimiter              *rate.Limiter
}

// New creates a new HookService struct instance
func New(
    config *config.Config,
    svixChannel chan *models.Notification,
    svixRetryChannel chan *models.Notification,
    waitGroup *sync.WaitGroup,
    svixClient *svix.Svix,
) *HookService {
    limitRate := rate.Every(time.Second / time.Duration(config.SvixApiMaxRate))
    return &HookService{
        Config:           config,
        SvixChannel:      svixChannel,
        svixRetryChannel: svixRetryChannel,
        SvixClient:       svixClient,
        waitGroup:        waitGroup,
        rateLimiter:      rate.NewLimiter(limitRate, config.SvixApiMaxBurst),
    }
}

// worker is the main worker function that takes messages from SvixChannel and SvixRetryChannel and sends them to Svix
func (hs *HookService) worker(workerId int32, context context.Context) {
    defer hs.waitGroup.Done()
    for {
        select {
        case <-context.Done():
            log.Printf("Context done, shutting down worker %d.", workerId)
            atomic.AddInt32(&hs.activeWorkers, -1)
            return
        case n, ok := <-hs.SvixChannel:
            if !ok {
                log.Printf("SvixChannel closed.")
                continue
            }
            hs.updateProcessedMessagesCount()
            hs.sendNotification(n)
        case n, ok := <-hs.svixRetryChannel:
            if !ok {
                log.Printf("svixRetryChannel closed.")
                continue
            }
            if n.ReadyToRetry() {
                _, err := n.IncrementRetryCount()
                if err != nil {
                    // TODO: implement logic to handle notifications that can not be retried, some kind of a DLQ
                    log.Printf("Retry count exceeded for notification %s: %s\n", n.Id, err)
                    continue
                }
                hs.sendNotification(n)
            } else {
                // Put the notification back to the channel, it's not ready to retry yet
                hs.svixRetryChannel <- n
            }
        }
    }
}

// sendNotification function sends the notification to Svix
// The function rate limits the requests to Svix and retries the request if it fails with 429 or 5xx
func (hs *HookService) sendNotification(n *models.Notification) {

    // TODO: implement logic to determine SvixAppId for the notification. See NOTES.md for more details.
    svixAppId := hs.Config.SvixAppId

    // Rate limit the requests to Svix. 5 requests per second by default.
    if err := hs.rateLimiter.Wait(context.Background()); err != nil {
        log.Printf("Error waiting for rate limiter: %s\n", err)
        return
    }

    // Create a message with the notification data and send it to Svix
    _, err := hs.SvixClient.Message.CreateWithOptions(nil, svixAppId, &svix.MessageIn{
        EventType: n.Type,
        EventId:   *svix.NullableString(&n.Id),
        Payload:   n.Data,
    },
        &svix.PostOptions{
            IdempotencyKey: &n.Id, // Use the notification id as the idempotency key
        },
    )
    log.Printf("Sent message to Svix: %s\n", n)

    if err != nil {
        // Cast the error to a Svix error to get the status code
        var svixErr *svix.Error
        if errors.As(err, &svixErr) {
            // If it's a Svix error, retry it.
            hs.retryNotification(n, svixErr)
        } else {
            // If it's not a Svix error, log the error and don't retry. We can not determine if it is safe to retry.
            log.Printf("Error sending notification and can't retry %s: %s\n", n.Id, err)
            // TODO: put this message to a DLQ
        }

    }
}

// retryNotification function retries sending the notification to Svix
// if the notification can be retried, or log the error if it can't
func (hs *HookService) retryNotification(n *models.Notification, err *svix.Error) {
    // Retry if the error is 429 too many requests or generic 5xx server error
    if err.Status() == http.StatusTooManyRequests || (err.Status() >= http.StatusInternalServerError) && n.CanRetry() {
        // Update the retry timestamp to calculate the next retry time
        n.SetRetryTime()
        hs.svixRetryChannel <- n
        log.Printf("Retrying notification %s, retry count: %d\n", n.Id, n.Metadata.RetryCount)
    } else {
        log.Printf("Error sending notification %s: %s\n", n.Id, err)
        // TODO: put this message to a DLQ
    }
}

// updateProcessedMessagesCount function updates the number of processed messages
// and the time of the last processed message. It should be called after each message is processed.
// It will reset the counter if it reaches the maximum value for uint64.
func (hs *HookService) updateProcessedMessagesCount() {
    if hs.messagesProcessed == ^uint64(0) { // Check if the value is the maximum for uint64
        hs.messagesProcessed = 0 // Reset to zero if it's the maximum
        log.Println("Resetting messagesProcessed counter to zero, reached maximum value for uint64")
    } else {
        // We are using atomic.AddUint64 to increment the counter because it's used in multiple goroutines
        atomic.AddUint64(&hs.messagesProcessed, 1) // Increment otherwise
    }
    hs.lastMessageProcessedTime = time.Now().Format(time.RFC3339)
}

// GetStats function returns the statistics of the hook service - number of workers, number of messages in channels
func (hs *HookService) GetStats() map[string]any {
    stats := map[string]any{
        "numWorkers":           hs.activeWorkers,
        "svixChannelSize":      len(hs.SvixChannel),
        "svixRetryChannelSize": len(hs.svixRetryChannel),
        "messagesProcessed":    hs.messagesProcessed,
        "lastMessageProcessed": hs.lastMessageProcessedTime,
    }
    return stats
}

// Start function starts the number of workers specified in the config
func (hs *HookService) Start(context context.Context) {
    log.Println("Starting workers...")
    for i := int32(0); i < hs.Config.NumWorkers; i++ {
        hs.waitGroup.Add(1)
        go hs.worker(i, context)
        atomic.AddInt32(&hs.activeWorkers, 1)
        log.Printf("Started worker %d\n", i)
    }
    log.Println("All workers started.")
}

// Stop function stops the workers gracefully
func (hs *HookService) Stop(cancelWorkers context.CancelFunc) {
    log.Println("Stopping workers...")
    // Close the swixChannel
    log.Println("Closing SvixChannel and svixRetryChannel")
    close(hs.SvixChannel)
    close(hs.svixRetryChannel)
    log.Println("Channels closed. waiting for workers to finish...")
    // Cancel the context
    cancelWorkers()
    hs.waitGroup.Wait()
    log.Println("All Workers finished.")
}
