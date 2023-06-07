package main

import (
	"github.com/dipdup-io/starknet-indexer/pkg/grpc"
	"github.com/dipdup-net/go-lib/config"
)

// Config -
type Config struct {
	config.Config `yaml:",inline"`

	LogLevel string             `yaml:"log_level" validate:"omitempty,oneof=debug trace info warn error fatal panic"`
	GRPC     *grpc.ClientConfig `yaml:"grpc" validate:"required"`
}

// Substitute -
func (c *Config) Substitute() error {
	return nil
}
