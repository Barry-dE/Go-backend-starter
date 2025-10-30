package service

import (
	"github.com/Barry-dE/go-backend-boilerplate/internal/lib/job"
	"github.com/Barry-dE/go-backend-boilerplate/internal/repository"
	"github.com/Barry-dE/go-backend-boilerplate/internal/server"
)

type Services struct {
	AuthService *AuthService
	Job         *job.JobService
}

func NewService(s *server.Server, repos *repository.Repositories) (*Services, error) {
	authService := NewAuthService(s)

	return &Services{
		AuthService: authService,
		Job:         s.Job,
	}, nil
}
