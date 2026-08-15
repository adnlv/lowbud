package uuid

import (
	"fmt"

	"github.com/google/uuid"
)

type Generator interface {
	Generate() (uuid.UUID, error)
	Parse(s string) (uuid.UUID, error)
}

type V7Generator struct{}

func NewV7Generator() *V7Generator {
	return &V7Generator{}
}

func (g *V7Generator) Generate() (uuid.UUID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.Nil, fmt.Errorf("V7Generator.Generate: failed to generate V7 UUID: %v", err)
	}
	return id, nil
}

func (g *V7Generator) Parse(s string) (uuid.UUID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, fmt.Errorf("V7Generator.Parse: failed to parse V7 UUID: %v", err)
	}
	return id, nil
}
