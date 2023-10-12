package models

import (
	"fmt"
	"math"
	"svix-hooks/pkg/utils/constants"
	"time"
)

// Notification struct holds the data of a notification received from the API /notifications endpoint
type Notification struct {
	Object          string         `json:"object"`
	Id              string         `json:"id"`
	Data            map[string]any `json:"data"`
	DataContentType string         `json:"datacontenttype"`
	Project         string         `json:"project"`
	Source          string         `json:"source"`
	SpecVersion     string         `json:"specversion"`
	Time            string         `json:"time"`
	Type            string         `json:"type"`
	Version         string         `json:"version"`
	Metadata        NotificationMetadata
}

// NotificationMetadata struct holds the metadata of a notification
type NotificationMetadata struct {
	RetryCount int
	RetryTime  int64
}

// String method for Notification struct returns a string representation of the notification
func (n Notification) String() string {
	return fmt.Sprintf("id:%s type:%s retries:%d", n.Id, n.Type, n.Metadata.RetryCount)
}

// SetRetryTime method for Notification struct sets the retry time to the current time plus the retry interval
func (n *Notification) SetRetryTime() {
	// Increase the retry time exponentially
	n.Metadata.RetryTime = int64(time.Duration(math.Pow(2, float64(n.Metadata.RetryCount))) * time.Second)
}

// ReadyToRetry method for Notification struct returns true if the retry time has passed
func (n *Notification) ReadyToRetry() bool {
	return time.Now().Unix() >= n.Metadata.RetryTime
}

// IncrementRetryCount method for Notification struct increments the retry count,
// and returns the new counter value
func (n *Notification) IncrementRetryCount() (int, error) {
	if n.Metadata.RetryCount < constants.MAX_RETRIES {
		n.Metadata.RetryCount++
	} else {
		return constants.MAX_RETRIES, fmt.Errorf(
			"Notification %s can not be tried more than MAX_RETRIES: %d times", n.Id, constants.MAX_RETRIES,
		)
	}
	return n.Metadata.RetryCount, nil
}

// CanRetry method for Notification struct returns true if the notification can be retried
func (n *Notification) CanRetry() bool {
	return n.Metadata.RetryCount < constants.MAX_RETRIES
}
