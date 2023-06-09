package main

import starknetid "github.com/dipdup-io/starknet-id/internal/starknet-id"

// Config -
type Config struct {
	StarknetId starknetid.ApiConfig `yaml:"starknet_id" validate:"required"`
	GraphQlApi GraphQlApiConfig     `yaml:"graphql" validate:"required"`
	LogLevel   string               `yaml:"log_level" validate:"omitempty,oneof=debug trace info warn error fatal panic"`
	Start      int                  `yaml:"start" validate:"omitempty,min=0"`
}

// Substitute -
func (c *Config) Substitute() error {
	return nil
}
