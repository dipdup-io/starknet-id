package main

import (
	"context"
	"time"

	"github.com/dipdup-io/starknet-go-api/pkg/data"
	"github.com/dipdup-io/starknet-id/internal/storage/postgres"
	"github.com/dipdup-io/starknet-indexer/pkg/grpc"
	"github.com/dipdup-io/starknet-indexer/pkg/grpc/pb"
	"github.com/dipdup-net/indexer-sdk/pkg/modules"
	"github.com/dipdup-net/indexer-sdk/pkg/storage"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// variables
var (
	ZeroAddress = data.Felt("0x0").Bytes()
)

// input name
const (
	InputName = "input"
)

// Indexer -
type Indexer struct {
	modules.BaseModule

	client         *grpc.Client
	storage        postgres.Storage
	channels       map[uint64]Channel
	channelsByName map[string]Channel
	subscriptions  map[string]grpc.Subscription
	subdomains     map[string]string
}

// NewIndexer -
func NewIndexer(pg postgres.Storage, client *grpc.Client, subdomains map[string]string) *Indexer {
	indexer := &Indexer{
		BaseModule:     modules.New("starknet_id_indexer"),
		client:         client,
		storage:        pg,
		channels:       make(map[uint64]Channel),
		channelsByName: make(map[string]Channel),
		subscriptions:  make(map[string]grpc.Subscription),
		subdomains:     subdomains,
	}

	indexer.CreateInput(InputName)

	return indexer
}

// Start -
func (indexer *Indexer) Start(ctx context.Context) {
	if err := indexer.init(ctx); err != nil {
		log.Err(err).Msg("state initialization")
		return
	}

	indexer.client.Start(ctx)

	indexer.G.GoCtx(ctx, indexer.reconnectThread)
	indexer.G.GoCtx(ctx, indexer.listen)
}

// Subscribe -
func (indexer *Indexer) Subscribe(ctx context.Context, subscriptions map[string]grpc.Subscription) error {
	indexer.subscriptions = subscriptions

	for name, sub := range subscriptions {
		ch, ok := indexer.channelsByName[name]
		if !ok {
			ch = NewChannel(name, indexer.storage, indexer.subdomains)
		}

		ch.Start(ctx)

		if err := indexer.actualFilters(ctx, ch, &sub); err != nil {
			return errors.Wrap(err, "filters modifying")
		}

		log.Info().Str("topic", name).Msg("subscribing...")
		req := sub.ToGrpcFilter()
		subId, err := indexer.client.Subscribe(ctx, req)
		if err != nil {
			return errors.Wrap(err, "subscribing error")
		}
		indexer.channels[subId] = ch
	}
	return nil
}

func (indexer *Indexer) init(ctx context.Context) error {
	states, err := indexer.storage.State.List(ctx, 10, 0, storage.SortOrderAsc)
	switch {
	case err == nil:
		for i := range states {
			ch := NewChannel(states[i].Name, indexer.storage, indexer.subdomains)
			ch.blockCtx.state = states[i]
			indexer.channelsByName[states[i].Name] = ch
		}
		return nil
	case indexer.storage.State.IsNoRows(err):
		return nil
	default:
		return err
	}
}

func (indexer *Indexer) listen(ctx context.Context) {
	input := indexer.MustInput(InputName)

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("close listen thread")
			return

		case msg, ok := <-input.Listen():
			if !ok {
				continue
			}

			switch typ := msg.(type) {
			case *pb.Subscription:
				channel, ok := indexer.channels[typ.Response.Id]
				if !ok {
					log.Error().Uint64("id", typ.Response.Id).Msg("unknown subscription")
					continue
				}
				channel.Add(typ)
			default:
				log.Info().Msgf("unknown message: %T", typ)
			}
		}
	}
}

func (indexer *Indexer) reconnectThread(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("close reconnect thread")
			return
		case subscriptionId, ok := <-indexer.client.Reconnect():
			if !ok {
				continue
			}

			if err := indexer.resubscribe(ctx, subscriptionId); err != nil {
				log.Err(err).Msg("resubscribe")
			}
		}
	}
}

func (indexer *Indexer) resubscribe(ctx context.Context, id uint64) error {
	channel, ok := indexer.channels[id]
	if !ok {
		return errors.Errorf("unknown subscription: %d", id)
	}

	for !channel.IsEmpty() {
		select {
		case <-ctx.Done():
			return nil
		default:
			time.Sleep(time.Second)
		}
	}

	delete(indexer.channels, id)

	sub, ok := indexer.subscriptions[channel.Name()]
	if !ok {
		return errors.Errorf("unknown subscription request: %d", id)
	}

	if err := indexer.actualFilters(ctx, channel, &sub); err != nil {
		return errors.Wrap(err, "filters modifying")
	}

	log.Info().Str("topic", channel.Name()).Msg("resubscribing...")
	req := sub.ToGrpcFilter()
	subId, err := indexer.client.Subscribe(ctx, req)
	if err != nil {
		return errors.Wrap(err, "resubscribing error")
	}
	indexer.channels[subId] = channel

	return nil
}

func (indexer *Indexer) actualFilters(ctx context.Context, ch Channel, sub *grpc.Subscription) error {
	if sub.EventFilter != nil {
		for i := range sub.EventFilter {
			sub.EventFilter[i].Height = &grpc.IntegerFilter{
				Gt: ch.blockCtx.state.LastHeight,
			}
			sub.EventFilter[i].Time = &grpc.TimeFilter{
				Gt: 1701088623,
			}
		}

	}
	if sub.AddressFilter != nil {
		lastId, err := indexer.storage.Addresses.LastID(ctx)
		if err != nil {
			if !indexer.storage.Addresses.IsNoRows(err) {
				return errors.Wrap(err, "get last address id")
			}
		}
		for i := range sub.AddressFilter {
			sub.AddressFilter[i].Id = &grpc.IntegerFilter{
				Gt: lastId,
			}
		}
	}

	return nil
}

// Unsubscribe -
func (indexer *Indexer) Unsubscribe(ctx context.Context) error {
	for subId, channel := range indexer.channels {
		log.Info().Str("subscription", channel.Name()).Uint64("id", subId).Msg("unsubscribing...")
		if err := indexer.client.Unsubscribe(ctx, subId); err != nil {
			return errors.Wrap(err, "unsubscribing")
		}

	}
	return nil
}

// Close - gracefully stops module
func (indexer *Indexer) Close() error {
	indexer.G.Wait()

	for _, channel := range indexer.channels {
		if err := channel.Close(); err != nil {
			return err
		}
	}

	return nil
}
