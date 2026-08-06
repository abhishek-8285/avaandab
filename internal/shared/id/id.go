package id

import (
	"fmt"
	"github.com/google/uuid"
	"transport-app/internal/shared/ports"
)

type uuidGenerator struct{}

// NewUUIDGenerator returns an IDGenerator using UUID v4.
func NewUUIDGenerator() ports.IDGenerator {
	return &uuidGenerator{}
}

func (g *uuidGenerator) GenerateUUID() string {
	return uuid.NewString()
}

func (g *uuidGenerator) GenerateDisplayID(prefix string) string {
	id := uuid.NewString()
	return fmt.Sprintf("%s-%s", prefix, id[:8])
}
