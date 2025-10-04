package job

import (
	"github.com/Barry-dE/go-backend-boilerplate/internal/config"
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"
)

// - Client is used to enqueue tasks
// - server runs worker goroutines that process tasks
// - logger logs start / stop messages
type JobService struct {
	Client *asynq.Client
	logger *zerolog.Logger
	server *asynq.Server
}

func NewJobService(logger *zerolog.Logger, cfg *config.Config) *JobService {
	// Read Redis address from config
	redisAddress := cfg.Redis.Address

	// Create an asynq client that will be used to enqueue tasks
	client := asynq.NewClient(asynq.RedisClientOpt{
		Addr: redisAddress,
	})

	// Create an asynq server which will execute tasks with a given concurrency and queue weights
	server := asynq.NewServer(asynq.RedisClientOpt{
		Addr: redisAddress,
	}, asynq.Config{
		Concurrency: 10,
		Queues: map[string]int{
			"critical": 6, // more capacity for important tasks
			"default":  3, // normal tasks
			"low":      1, // non-urgent tasks
		},
	})
	return &JobService{
		Client: client,
		logger: logger,
		server: server,
	}
}

func (js *JobService) Start() error {
	// create a new multiplexer to route incoming tasks to handlers
	mux := asynq.NewServeMux()

	// register a handler function for each task type
	mux.HandleFunc(TaskWelcomeEmail, js.handleWelcomeEmailTask)

	js.logger.Info().Msg("Starting job server...")

	// if starting the server fails, return the error so caller can handle it
	if err := js.server.Start(mux); err != nil {
		return err
	}

	return nil
}

// graceful shutdown
func (js *JobService) Stop() {
	js.logger.Info().Msg("stopping job server...")
	js.server.Shutdown()
	js.Client.Close()
}
