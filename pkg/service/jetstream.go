package service

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/cosmos/btcutil/base58"
	"github.com/nats-io/nats.go"
)

var ErrNotAvailable = fmt.Errorf("not available")

func (b *Service) jsMakeHash(subjects ...string) string {
	config := strings.Join(subjects, ",")
	sum := sha256.Sum256([]byte(config))
	return base58.Encode(sum[:])
}

func (b *Service) jsStreamName() string {
	if b.StreamName != "" {
		return b.StreamName
	}
	return fmt.Sprintf("%s-%s", b.Prefix, b.Name)
}

func (b *Service) jsConsumerName(hash string) string {
	return fmt.Sprintf("%s-%s", b.Identity, hash)
}

// AddStream is an experimental feature that creates a durable stream. It is possible to
// subscribe to this durable stream using regular Subscribe or SubscribeTo methods given that
// the subject is included in the created stream.
//
// The interface for this feature is experimental and it should be expected to change.
//
// This method will create a stream with maxMsgs, maxBytes, and age for a list of subjects on JetStream if it does not exist.
// This is a temporary solution so that the stream doesn't have to be created manually. However,
// this will change in the near future, therefore users will have to make sure that the stream exists
// before calling this method(maxMsgs, maxBytes, and age parameters will be removed).
//
// NOTE: Messages are automatically acknowledged after handler returns.
func (b *Service) AddStream(maxMsgs, maxBytes uint64, age time.Duration, subjects ...string) error {
	if b.js == nil {
		return ErrNotAvailable
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	for i, subject := range subjects {
		s := Subject(subject)

		if err := Subject(s).Validate(); err != nil {
			return fmt.Errorf("%s validation failed: %w", s, err)
		}

		if match, _, found := b.streamSubjects.SymmetricSearch(s); found {
			return fmt.Errorf("%s already configured as %s", s, match)
		}

		for _, s1 := range subjects[i+1:] {
			if s.SymmetricMatch(Subject(s1)) {
				return fmt.Errorf("overlapping subjects: %s and %s", s, s1)
			}
		}
	}

	hash := b.jsMakeHash(subjects...)
	streamName := b.jsStreamName()

	connCtx, connCancelFn := context.WithTimeout(b.Context, 10*time.Second)
	defer connCancelFn()

	cfg := &nats.StreamConfig{
		Name:     streamName,
		Subjects: subjects,
		MaxMsgs:  int64(maxMsgs),
		MaxBytes: int64(maxBytes),
		MaxAge:   age,
	}
	si, err := b.js.AddStream(cfg, nats.Context(connCtx))
	if err != nil {
		return fmt.Errorf("AddStream failed: %w", err)
	}

	ccfg := &nats.ConsumerConfig{
		Durable:       b.jsConsumerName(hash),
		AckPolicy:     nats.AckExplicitPolicy,
		DeliverPolicy: nats.DeliverAllPolicy,
	}
	ci, err := b.js.AddConsumer(streamName, ccfg)
	if errors.Is(err, nats.ErrConsumerNameAlreadyInUse) {
		ci, err = b.js.UpdateConsumer(streamName, ccfg)
	}
	if err != nil {
		return fmt.Errorf("AddConsumer failed: %w", err)
	}

	b.streams = append(b.streams, jsStream{
		cfgStream:    cfg,
		cfgConsumer:  ccfg,
		streamInfo:   si,
		consumerInfo: ci,
	})

	for _, s := range subjects {
		b.streamSubjects.Add(Subject(s), len(b.streams)-1)
	}

	return nil
}

// RemoveStream will attempt to remove consumers based on a list of subjects.
// List of subjects must be exactly the same as was used in AddStream since js Consumer names
// are based on the subjects.
func (b *Service) RemoveStream(subjects ...string) error {
	if b.js == nil {
		return ErrNotAvailable
	}

	hash := b.jsMakeHash(subjects...)
	streamName := b.jsStreamName()
	consumerName := b.jsConsumerName(hash)

	b.mu.Lock()
	defer b.mu.Unlock()

	ctx, cancel := context.WithTimeout(b.Context, 30*time.Second)
	defer cancel()

	err := b.js.DeleteConsumer(streamName, consumerName, nats.Context(ctx))
	if err != nil {
		return fmt.Errorf("DeleteConsumer failed: %w", err)
	}

	b.removeStreamsFromMap(subjects)
	return nil
}

func (b *Service) removeStreamsFromMap(subjects []string) {
	idsToDeleteSet := make(map[int]struct{}, len(subjects))
	for _, subject := range subjects {
		s := Subject(subject)
		if ssIdx, ok := b.streamSubjects.Get(s); ok {
			delete(b.streamSubjects, s)
			idsToDeleteSet[ssIdx] = struct{}{}
		}
	}

	idsToDelete := make([]int, 0, len(idsToDeleteSet))
	for id := range idsToDelete {
		idsToDelete = append(idsToDelete, id)
	}
	slices.SortFunc(idsToDelete, func(a, b int) int { return b - a })

	for id := range idsToDelete {
		b.streams = slices.Delete(b.streams, id, id+1)
	}
}

func (b *Service) attemptJSConsume(handler nats.MsgHandler, subject string) (*nats.Subscription, error) {
	if b.js == nil {
		return nil, ErrNotAvailable
	}

	var info jsStream
	streamName, consumerName := "", ""
	if _, id, ok := b.streamSubjects.Search(Subject(subject)); !ok {
		return nil, ErrNotAvailable
	} else {
		info = b.streams[id]
		streamName = info.cfgStream.Name
		consumerName = info.consumerInfo.Config.Durable
	}

	sub, err := b.js.PullSubscribe(streamName, consumerName, nats.ManualAck(), nats.Bind(streamName, consumerName))
	if err != nil {
		return nil, err
	}

	b.Group.Go(func() error {
		b.Logger.Info("PullSubscribe loop start", "subject", subject)
		defer b.Logger.Info("PullSubscribe loop exit", "subject", subject)

		for {
			select {
			case <-b.Context.Done():
				return b.Context.Err()
			default:
			}

			ctx, cancel := context.WithTimeout(b.Context, 5*time.Second)
			defer cancel()

			msgs, err := sub.Fetch(10, nats.Context(ctx))
			if err != nil {
				if errors.Is(err, context.Canceled) {
					return fmt.Errorf("context cancelled during pulling next message: %w", err)
				}
				if errors.Is(err, context.DeadlineExceeded) {
					continue
				}
				if errors.Is(err, nats.ErrBadSubscription) {
					b.Logger.Info("subscription closed", "subject", subject)
					return nil
				}
				return fmt.Errorf("pulling message failed: %w", err)
			}
			for _, msg := range msgs {
				handler(msg)
				if err := msg.Ack(); err != nil {
					b.Logger.Warn("message ack failed", "err", err)
				}
			}
		}
	})

	return sub, nil
}
