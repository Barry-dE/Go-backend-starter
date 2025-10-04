package job

import (
	"encoding/json"
	"time"

	"github.com/hibiken/asynq"
)

const TaskWelcomeEmail = "email:welcome"

type WelcomeEmailTaskPayload struct {
	To        string `json:"to"`         // recipient email address
	FirstName string `json:"first_name"` // recipient first name
}

// NewWelcomeEmailTask creates a new task to send a welcome email to a user
func NewWelcomeEmailTask(to string, firstName string) (*asynq.Task, error) {
	jsonPayload, err := json.Marshal(WelcomeEmailTaskPayload{
		To:        to,
		FirstName: firstName,
	})

	if err != nil {
		return nil, err
	}

	return asynq.NewTask(TaskWelcomeEmail, jsonPayload, asynq.Timeout(30*time.Second), asynq.MaxRetry(3), asynq.Queue("default")), nil
}
