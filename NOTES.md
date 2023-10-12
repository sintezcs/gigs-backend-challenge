# Svix-Hooks - Gigs backend challenge

## Customer facing ideas
Here is a number of ideas that I came up with for improving the customer experience for the webhook service:

1. Reporting and monitoring. Show a detailed dashboard with the number of events sent to Svix, the number of errors, 
the number of retries, etc.
2. It will be really helpful if we track the actual webhook delivery success rate and details, by using the Svix API. 
We might need to store Svix-generated notification IDs for this, but it will be really helpful for clients to 
debug issues with their webhooks.
3. We can provide a way for clients to configure the retry policy for their webhooks, how many times to retry, 
the timeouts, etc.
4. We need to give them the ability to filter the events that they want to receive, by event type, payload, etc.
5. Probably some of the customers may want to use their own Svix account. So we may give them a way to provide 
their own Svix API key and application ID.


## Code structure

### Packages 
The code is split into the following packages:
- `main` — contains the main function and the code for starting the server
- `api` — contains the HTTP handlers for the endpoints
- `config` — contains the configuration values for the application
- `models` — contains the models used in the application
- `hookService` — contains the logic for sending the events to Svix
- `utils` — contains utility functions 
- `constants` — contains constants used in the application

### Architecture and code flow
The application is using `http.server` from the standard library to start a server and listen for incoming requests. 
Currently, it exposes three endpoints: 
- `POST /notifications` — for receiving the events
- `GET /health` — for checking the health of the application
- `GET /stats` — for checking the basic metrics of the application

The logic for sending the events to Svix is implemented in the `hookService` package. The service starts a configurable 
number of goroutines that are listening on a channel for events. The `/notifications` API endpoint is sending the events 
to the channel and the goroutines are processing them. When an event is received, the goroutine will send it 
to Svix and if the request fails, it will retry it a configurable number of times, and use an exponential backoff. 
The service will retry sending the event to Svix only if the request fails with a 429 status code or any internal 
server error code (>=500). For retrying the events, the service is using a separate channel.

The requests to Svix API are being rate-limited using [x/time/rate](https://pkg.go.dev/golang.org/x/time/rate) package.
The rate limit is set to 5 messages per second, and the burst is set to 10 messages. The rate limit is configurable.

For ensuring idempotency, the service is using a Svix built-in feature — [idempotency keys](https://docs.svix.com/docs/idempotency-keys).
`Notification.Id` is being used as an idempotency key, that is added to the request headers.  

The service is using a simple counter for tracking the number of events sent to Svix. The counter is exposed via the 
`/stats` API endpoint. The counter is being reset when the number is approaching the maximum value of an unsigned 
64-bit, so it's not for production use.

### Important assumptions

#### 1. Idempotency

Incoming `Notification.Id` is currently being used as an idempotency key. I've assumed that it's guaranteed that the 
`Notification.Id` is unique for each event. If this is not the case, we could use a combination of `Notification.Id` 
and `Notification.CreatedAt` as an idempotency key, or we could generate a UUID for each event and use it as an 
idempotency key.
 
#### 2. A single Svix APP ID for all the events

Currently, the application is using a single Svix APP ID for all the events. In a real-life scenario, we would need to 
determine the Svix APP ID based on the events' user or account. Because each Gigs user/account should have a separate 
Svix Application configured.

### Testing

For the sake of time and simplicity, I've decided to write only a single tests for the `hookService` package. But this 
test covers a number of aspects:
- Starting multiple worker goroutines
- Shutting down the workers gracefully
- Ensuring that the stats are being updated correctly (the number of active workers)
Please see `hookService_test.go` for more details.

### Production readiness and things to improve and implement

The following features and improvements should be implemented in order to make the service production-ready.

1. **Logic for determining Svix APP ID.** We need to determine the Svix APP ID based on the events' user or account. Or an 
upstream service/application should enrich the events with the Svix APP ID. The choice depends on the architecture of 
the system, and there are multiple ways to implement this.
2. **Logging.** The application already contains basic logging. The logs should be sent to a centralized 
logging system, like ELK or Splunk. Also, the logs should be enriched with additional information, like the tracking UUID,
the user ID, etc. The logs should be structured and in JSON format. Currently, the application is using the standard 
library logger, which is limited in functionality, so it should be replaced with a more advanced logging library.
3. **Monitoring and reporting.** The application should expose metrics and health checks. The current implementation 
already exposes a simple `/stats` endpoint and `/health` endpoint. But we need to make sure that the metrics are in a 
format that can be easily consumed by a monitoring system, like Prometheus. Also, integration with a system like Sentry 
will be really helpful for tracking unhandled errors and unexpected application faults.
4. **Testing.** The application should have a good test coverage. The current implementation has only a single test.  
The tests should be split into unit and integration tests. The unit tests should cover most of the critical parts of 
the application, like the `hookService` package (the logic for handling retries, determining Svix APP ID, 
rate-limiting, handling svix API errors, graceful restart and shutdown, etc.). The integration tests should cover the 
API endpoints and the integration with Svix API.
5. **Robustness.** If the request to Svix API fails after all the retries, the event should be sent to a dead letter queue. 
The application should have a mechanism for retrying the events from the dead letter queue. Also, we need to have a 
solution to monitor the DLQ and alert if the number of events in the queue is increasing.
6. **Deployment.** The application should be deployed in a containerized environment. So we need to create a Dockerfile 
for the application. Also, we need to create a CI/CD pipeline for building and deploying the application. 
7. **Documentation.** The application should have a detailed documentation for the API endpoints, configuration,
deployment, etc.
