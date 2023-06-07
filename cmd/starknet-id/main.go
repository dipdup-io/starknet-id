package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/dipdup-io/starknet-id/internal/storage"
	"github.com/dipdup-io/starknet-id/internal/storage/postgres"
	"github.com/dipdup-io/starknet-indexer/pkg/grpc"
	"github.com/dipdup-net/go-lib/config"
	"github.com/dipdup-net/go-lib/hasura"
	"github.com/dipdup-net/indexer-sdk/pkg/modules"
	grpcSDK "github.com/dipdup-net/indexer-sdk/pkg/modules/grpc"
	"github.com/dipdup-net/indexer-sdk/pkg/modules/printer"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "indexer",
		Short: "DipDup indexer",
	}
)

func main() {
	defer func() {
		if err := recover(); err != nil {
			log.Error().Interface("err", err).Msg("panic occurred")
		}
	}()

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

	pg, err := postgres.Create(ctx, cfg.Database)
	if err != nil {
		log.Panic().Err(err).Msg("database creation")
		return
	}
	views, err := createViews(ctx, pg)
	if err != nil {
		log.Panic().Err(err).Msg("create views")
		return
	}

	if cfg.Hasura != nil {
		models := make([]any, len(storage.Models))
		for i := range storage.Models {
			models[i] = storage.Models[i]
		}

		if err := hasura.Create(ctx, hasura.GenerateArgs{
			Config:         cfg.Hasura,
			DatabaseConfig: cfg.Database,
			Models:         models,
			Views:          views,
		}); err != nil {
			log.Panic().Err(err).Msg("hasura initialization")
			return
		}
	}

	client := grpc.NewClient(*cfg.GRPC)
	indexer := NewIndexer(pg, client)

	if err := modules.Connect(client, indexer, grpc.OutputMessages, printer.InputName); err != nil {
		log.Panic().Err(err).Msg("module connect")
		return
	}

	log.Info().Msg("connecting to gRPC...")
	if err := client.Connect(ctx,
		grpcSDK.WaitServer(),
		grpcSDK.WithUserAgent("starknet-id"),
	); err != nil {
		log.Panic().Err(err).Msg("grpc connect")
		return
	}
	log.Info().Msg("connected")

	client.Start(ctx)
	indexer.Start(ctx)

	if err := indexer.Subscribe(ctx, cfg.GRPC.Subscriptions); err != nil {
		log.Panic().Err(err).Msg("subscribe")
		return
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	<-signals

	if err := indexer.Unsubscribe(ctx); err != nil {
		log.Panic().Err(err).Msg("unsubscribe")
		return
	}

	cancel()

	if err := indexer.Close(); err != nil {
		log.Panic().Err(err).Msg("closing indexer")
	}
	if err := client.Close(); err != nil {
		log.Panic().Err(err).Msg("closing grpc server")
	}
	if err := pg.Close(); err != nil {
		log.Panic().Err(err).Msg("closing database connection")
	}

	close(signals)
}
