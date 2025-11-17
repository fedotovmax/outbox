package outbox

import (
	"context"
	"fmt"
	"sync"

	"github.com/IBM/sarama"
)

type Producer interface {
	GetInput() chan<- *sarama.ProducerMessage
	GetSuccesses() <-chan *sarama.ProducerMessage
	GetErrors() <-chan *sarama.ProducerError
}

type produceKafka struct {
	producer Producer

	onceSuccess sync.Once
	onceErrors  sync.Once

	successes chan *SuccessEvent
	errors    chan *FailedEvent
}

func newProduceKafka(p Producer) *produceKafka {
	return &produceKafka{
		producer:  p,
		successes: make(chan *SuccessEvent),
		errors:    make(chan *FailedEvent),
	}
}

func (p *produceKafka) Publish(ctx context.Context, ev *Event) error {
	const op = "outbox.kafka.Publish"

	metadata := &messageMetadata{
		ID:   ev.ID,
		Type: ev.Type,
	}

	msg := &sarama.ProducerMessage{
		Topic: ev.Topic,
		Key:   sarama.StringEncoder(ev.AggregateID),
		Value: sarama.ByteEncoder(ev.Payload),
		Headers: []sarama.RecordHeader{
			{
				Key:   []byte("event_id"),
				Value: []byte(ev.ID),
			},
			{
				Key:   []byte("event_type"),
				Value: []byte(ev.Type),
			},
		},
		Metadata: metadata,
	}

	select {
	case <-ctx.Done():
		return fmt.Errorf("%s: event_id: %s: %w", op, ev.ID, ctx.Err())
	case p.producer.GetInput() <- msg:
		return nil
	}
}

func (p *produceKafka) GetSuccesses(ctx context.Context) <-chan *SuccessEvent {
	p.onceSuccess.Do(func() {
		go func() {
			defer close(p.successes)
			for {
				select {
				case <-ctx.Done():
					return
				case msg, ok := <-p.producer.GetSuccesses():
					if !ok {
						return
					}
					m, ok := msg.Metadata.(*messageMetadata)
					if !ok {
						continue
					}
					select {
					case <-ctx.Done():
						return
					case p.successes <- &SuccessEvent{ID: m.ID, Type: m.Type}:
					}
				}
			}
		}()
	})

	return p.successes
}

func (p *produceKafka) GetErrors(ctx context.Context) <-chan *FailedEvent {

	const op = "outbox.kafka.GetErrors"

	p.onceErrors.Do(func() {
		go func() {
			defer close(p.errors)
			for {
				select {
				case <-ctx.Done():
					return
				case produceErr, ok := <-p.producer.GetErrors():
					if !ok {
						return
					}
					m, ok := produceErr.Msg.Metadata.(*messageMetadata)
					if !ok {
						continue
					}
					select {
					case <-ctx.Done():
						return
					case p.errors <- &FailedEvent{ID: m.ID, Type: m.Type, Error: fmt.Errorf("%s:%w", op,
						produceErr.Err)}:
					}
				}
			}
		}()
	})

	return p.errors
}
