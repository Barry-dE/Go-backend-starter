package handler

import (
	"github.com/Barry-dE/go-backend-boilerplate/internal/server"
	"github.com/Barry-dE/go-backend-boilerplate/internal/service"
)

type Handlers struct{}

func NewHandler(s *server.Server, services *service.Services) *Handlers {
return &Handlers{}
}