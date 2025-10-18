package repository

import "github.com/Barry-dE/go-backend-boilerplate/internal/server"

type Repositories struct{}

func NewRepositories(s *server.Server) *Repositories{
return &Repositories{}
}