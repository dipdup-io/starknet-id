package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/dipdup-net/go-lib/config"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "tester",
		Short: "Checks data correctness",
	}
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: "2006-01-02 15:04:05",
	})
	configPath := rootCmd.PersistentFlags().StringP("config", "c", "dipdup.yml", "path to YAML config file")
	if err := rootCmd.Execute(); err != nil {
		log.Panic().Err(err).Msg("command line execute")
		return
	}
	if err := rootCmd.MarkFlagRequired("config"); err != nil {
		log.Panic().Err(err).Msg("config command line arg is required")
		return
	}

	var cfg Config
	if err := config.Parse(*configPath, &cfg); err != nil {
		log.Panic().Err(err).Msg("parsing config file")
		return
	}

	if cfg.LogLevel == "" {
		cfg.LogLevel = zerolog.LevelInfoValue
	}

	logLevel, err := zerolog.ParseLevel(cfg.LogLevel)
	if err != nil {
		log.Panic().Err(err).Msg("parsing log level")
		return
	}
	zerolog.SetGlobalLevel(logLevel)

	ctx, cancel := context.WithCancel(context.Background())

	tester, err := NewTester(cfg)
	if err != nil {
		log.Panic().Err(err).Msg("create tester")
		return
	}

	tester.Start(ctx)

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	<-signals

	cancel()

	if err := tester.Close(); err != nil {
		log.Panic().Err(err).Msg("closing tester")
	}
}
