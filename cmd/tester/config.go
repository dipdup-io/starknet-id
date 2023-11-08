package main

import starknetid "github.com/dipdup-io/starknet-id/internal/starknet-id"

// Config -
type Config struct {
	StarknetId starknetid.ApiConfig `validate:"required"                                                yaml:"starknet_id"`
	GraphQlApi GraphQlApiConfig     `validate:"required"                                                yaml:"graphql"`
	LogLevel   string               `validate:"omitempty,oneof=debug trace info warn error fatal panic" yaml:"log_level"`
	Parts      int                  `validate:"omitempty,min=0"                                         yaml:"parts"`
}

// Substitute -
func (c *Config) Substitute() error {
	return nil
}
