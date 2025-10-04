// This handler unctions process the task
package job

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Barry-dE/go-backend-boilerplate/internal/config"
	"github.com/Barry-dE/go-backend-boilerplate/internal/lib/email"
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"
)


var emailClient *email.Client

func (j *JobService) InitHandlers(config *config.Config, logger *zerolog.Logger) {
	emailClient = email.NewClient(config, logger)
}

func (j *JobService) handleWelcomeEmailTask(ctx context.Context, t *asynq.Task) error {
	var p WelcomeEmailTaskPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("failed to unmarshal welcome email payload: %w", err)
	}

	j.logger.Info().Str("type", "welcome").Str("to", p.To).Msg("processing welcome email task")

	err := emailClient.SendWelcomeEmail(p.To, p.FirstName)
	if err != nil {
		j.logger.Error().Str("type", "welcome").Str("to", p.To).Err(err).Msg("welcome email sending failed")
		return err
	}

	j.logger.Info().Str("type", "welcome").Str("to", p.To).Msg("successfully sent welcome email")

	return nil
}
