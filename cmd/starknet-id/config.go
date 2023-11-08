package main

import (
	"github.com/dipdup-io/starknet-indexer/pkg/grpc"
	"github.com/dipdup-net/go-lib/config"
)

// Config -
type Config struct {
	config.Config `yaml:",inline"`

	LogLevel   string             `validate:"omitempty,oneof=debug trace info warn error fatal panic" yaml:"log_level"`
	GRPC       *grpc.ClientConfig `validate:"required"                                                yaml:"grpc"`
	Subdomains map[string]string  `validate:"required"                                                yaml:"subdomains"`
}

// Substitute -
func (c *Config) Substitute() error {
	return nil
}
