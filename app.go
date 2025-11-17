package outbox

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fedotovmax/pgxtx"
	"github.com/fedotovmax/workerpool"
)

type Outbox struct {
	kafka   *produceKafka
	usecase *eventUsesace
	log     *slog.Logger
	cfg     Config
	// flag for
	inProcess int32
	ctx       context.Context
	stop      context.CancelFunc
	isStopped chan struct{}
}

func New(l *slog.Logger, p Producer, txm pgxtx.Manager, ex pgxtx.Extractor, cfg Config) *Outbox {

	ctx, cancel := context.WithCancel(context.Background())

	validateConfig(&cfg)

	kafka := newProduceKafka(p)

	postgres := newEventPostgres(ex)

	usecase := newEventUsecase(postgres, txm)

	return &Outbox{
		kafka:     kafka,
		log:       l,
		usecase:   usecase,
		cfg:       cfg,
		ctx:       ctx,
		stop:      cancel,
		isStopped: make(chan struct{}),
	}
}

func (a *Outbox) Start() {

	wg := &sync.WaitGroup{}

	a.successesMonitoring(wg)
	a.errorsMonitoring(wg)
	a.processingNewEvents(wg)

	go func() {
		wg.Wait()
		close(a.isStopped)
	}()
}

func (a *Outbox) Stop(ctx context.Context) error {
	const op = "outbox.app.Stop"
	log := a.log.With(slog.String("op", op))
	a.stop()
	select {
	case <-a.isStopped:
		log.Info("Event Processor stopped successfully")
		return nil
	case <-ctx.Done():
		log.Warn("Event Processor stopped by context")
		return fmt.Errorf("%s: %w", op, ctx.Err())
	}
}

func (a *Outbox) AddNewEvent(ctx context.Context, ev CreateEvent) (string, error) {
	return a.usecase.AddNewEvent(ctx, ev)
}

func (a *Outbox) successesMonitoring(wg *sync.WaitGroup) {
	const op = "outbox.app.successesMonitoring"

	log := a.log.With(slog.String("op", op))

	eventsSuccesses := a.kafka.GetSuccesses(a.ctx)

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-a.ctx.Done():
				log.Info("monitoring [successes] stopped: ctx closed")
				return
			case event, ok := <-eventsSuccesses:
				if !ok {
					log.Info("monitoring [successes] stopped: channel closed")
					return
				}
				err := a.confirm(a.ctx, event)
				if err != nil {
					log.Error("error when confirm event, but event is sended",
						slog.String("error", err.Error()))
					continue
				}
				log.Info("event sended", slog.String("event_id", event.ID))
			}
		}
	}()
}

func (a *Outbox) errorsMonitoring(wg *sync.WaitGroup) {
	const op = "outbox.app.errorsMonitoring"

	log := a.log.With(slog.String("op", op))

	eventsErrors := a.kafka.GetErrors(a.ctx)

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-a.ctx.Done():
				log.Info("monitoring [errors] stopped: ctx closed")
				return
			case event, ok := <-eventsErrors:
				if !ok {
					log.Info("monitoring [errors] stopped: channel closed")
					return
				}
				log.Error("event send failed",
					slog.String("event_id", event.ID), slog.String("error", event.Error.Error()))

				err := a.fail(a.ctx, event)

				if err != nil {
					log.Error("error when confirm send fail", slog.String("error", err.Error()))
				}
			}
		}
	}()
}

func (a *Outbox) fail(ctx context.Context, ev *FailedEvent) error {
	const op = "outbox.app.fail"

	queriesCtx, cancelQueriesCtx := context.WithTimeout(ctx, a.cfg.ProcessTimeout)
	defer cancelQueriesCtx()

	err := a.usecase.ConfirmFailed(queriesCtx, ev)

	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (a *Outbox) confirm(ctx context.Context, ev *SuccessEvent) error {

	const op = "outbox.app.confirm"

	queriesCtx, cancelQueriesCtx := context.WithTimeout(ctx, a.cfg.ProcessTimeout)
	defer cancelQueriesCtx()

	err := a.usecase.ConfirmEvent(queriesCtx, ev)

	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil

}

func (a *Outbox) processingNewEvents(wg *sync.WaitGroup) {
	const op = "outbox.app.processingNewEvents"

	log := slog.With(slog.String("op", op))

	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(a.cfg.Interval)
		defer ticker.Stop()
		for {
			select {
			case <-a.ctx.Done():
				log.Info("event processing stopped")
				return
			case <-ticker.C:
				if !atomic.CompareAndSwapInt32(&a.inProcess, 0, 1) {
					continue
				}
				a.process(a.ctx)
				atomic.StoreInt32(&a.inProcess, 0)
			}
		}
	}()
}

func (a *Outbox) process(ctx context.Context) {

	const op = "outbox.app.process"

	log := slog.With(slog.String("op", op))

	queriesCtx, cancelQueriesCtx := context.WithTimeout(ctx, a.cfg.ProcessTimeout)
	defer cancelQueriesCtx()

	events, err := a.usecase.ReserveNewEvents(queriesCtx, a.cfg.Limit, a.cfg.ReserveDuration)

	if err != nil {
		if errors.Is(err, ErrNoNewEvents) {
			log.Info("skip processing, no new events")
			return
		}
		log.Error("error when processing", slog.String("error", err.Error()))
		return
	}

	eventsCh := make(chan *Event, len(events))
	for i := 0; i < len(events); i++ {
		eventsCh <- events[i]
	}
	close(eventsCh)

	workerPoolCtx, workerPoolCtxCancel := context.WithCancel(ctx)
	defer workerPoolCtxCancel()

	publishResults := workerpool.Workerpool(workerPoolCtx, eventsCh, a.cfg.Workers,
		func(ev *Event) error {
			publishCtx, cancelPublishCtx := context.WithTimeout(workerPoolCtx, a.cfg.ProcessTimeout)
			defer cancelPublishCtx()
			return a.kafka.Publish(publishCtx, ev)
		})

	for err := range publishResults {
		if err != nil {
			log.Warn("publish error", slog.String("error", err.Error()))
		}
	}
}
