package api

// API handlers for POST /notification and GET /health endpoints

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"svix-hooks/pkg/config"
	"svix-hooks/pkg/models"
	"svix-hooks/pkg/services/hookService"
	"svix-hooks/pkg/utils"
)

// Api struct holds the API handlers and the configuration
type Api struct {
	Config      *config.Config
	HookService *hookService.HookService
}

// New creates a new Api struct instance
func New(config *config.Config, hookService *hookService.HookService) *Api {
	return &Api{
		Config:      config,
		HookService: hookService,
	}
}

// NotificationHandler is an API handler for POST /notification endpoint
// It receives notifications and sends them to the hook service, which will send them to Svix
func (api *Api) NotificationHandler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		utils.BadRequestResponse(w, err, "Error reading request body")
		return
	}
	var notification models.Notification
	err = json.Unmarshal(body, &notification)
	if err != nil {
		utils.BadRequestResponse(w, err, "Error parsing request body")
		return
	}

	// TODO: it will be nice to validate the payload here using JSON schema

	log.Printf("Received notification: %s\n", notification)
	api.HookService.SvixChannel <- &notification
	log.Printf("Notification %s sent to Svix service\n", notification.Id)
	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte(`{"result": "ok"}`))
	if err != nil {
		log.Printf("Error writing response: %s\n", err)
	}
}

// HealthHandler is an API handler for GET /health endpoint
// It returns the health status of the hook service
func (api *Api) HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, err := w.Write([]byte(`{"status": "ok"}`))
	if err != nil {
		log.Printf("Error writing response: %s\n", err)
	}
}

// StatsHandler is an API handler for GET /stats endpoint
// It returns the stats of the hook service
func (api *Api) StatsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	stats := api.HookService.GetStats()
	statsJson, err := json.Marshal(stats)
	if err != nil {
		utils.ServerErrorResponse(w, err, "Error marshalling stats")
		return
	}
	_, err = w.Write(statsJson)
	if err != nil {
		log.Printf("Error writing response: %s\n", err)
	}
}
