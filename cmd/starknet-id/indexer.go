package main

import (
	"context"
	"sync"
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
	client         *grpc.Client
	storage        postgres.Storage
	input          *modules.Input
	channels       map[uint64]Channel
	channelsByName map[string]Channel
	subscriptions  map[string]grpc.Subscription
	subdomains     map[string]string

	wg *sync.WaitGroup
}

// NewIndexer -
func NewIndexer(pg postgres.Storage, client *grpc.Client, subdomains map[string]string) *Indexer {
	indexer := &Indexer{
		client:         client,
		storage:        pg,
		input:          modules.NewInput(InputName),
		channels:       make(map[uint64]Channel),
		channelsByName: make(map[string]Channel),
		subscriptions:  make(map[string]grpc.Subscription),
		subdomains:     subdomains,
		wg:             new(sync.WaitGroup),
	}

	return indexer
}

// Start -
func (indexer *Indexer) Start(ctx context.Context) {
	if err := indexer.init(ctx); err != nil {
		log.Err(err).Msg("state initialization")
		return
	}

	indexer.client.Start(ctx)

	indexer.wg.Add(1)
	go indexer.reconnectThread(ctx)

	indexer.wg.Add(1)
	go indexer.listen(ctx)
}

// Name -
func (indexer *Indexer) Name() string {
	return "starknet_id_indexer"
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

// Input - returns input by name
func (indexer *Indexer) Input(name string) (*modules.Input, error) {
	switch name {
	case InputName:
		return indexer.input, nil
	default:
		return nil, errors.Wrap(modules.ErrUnknownInput, name)
	}
}

func (indexer *Indexer) listen(ctx context.Context) {
	defer indexer.wg.Done()

	input, err := indexer.Input(InputName)
	if err != nil {
		log.Err(err).Msg("unknown input")
		return
	}

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
	defer indexer.wg.Done()

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

// Output - returns output by name
func (indexer *Indexer) Output(name string) (*modules.Output, error) {
	return nil, errors.Wrap(modules.ErrUnknownOutput, name)
}

// AttachTo - attach input to output with name
func (indexer *Indexer) AttachTo(name string, input *modules.Input) error {
	output, err := indexer.Output(name)
	if err != nil {
		return err
	}
	output.Attach(input)
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
	indexer.wg.Wait()

	for _, channel := range indexer.channels {
		if err := channel.Close(); err != nil {
			return err
		}
	}

	if err := indexer.input.Close(); err != nil {
		return err
	}

	return nil
}
