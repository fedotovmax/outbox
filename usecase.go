package outbox

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/fedotovmax/pgxtx"
)

type eventUsesace struct {
	ea  *eventPostgres
	txm pgxtx.Manager
}

func newEventUsecase(ea *eventPostgres, txm pgxtx.Manager) *eventUsesace {

	return &eventUsesace{
		ea:  ea,
		txm: txm,
	}
}

func (e *eventUsesace) AddNewEvent(ctx context.Context, ev CreateEvent) (string, error) {
	return e.ea.Create(ctx, ev)
}

func (e *eventUsesace) ConfirmFailed(ctx context.Context, ev *FailedEvent) error {
	const op = "outbox.usecase.ConfirmFailed"

	err := e.txm.Wrap(ctx, func(txCtx context.Context) error {

		err := e.ea.RemoveReserve(txCtx, ev.ID)

		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (e *eventUsesace) ConfirmEvent(ctx context.Context, ev *SuccessEvent) error {

	const op = "outbox.usecase.ConfirmEvent"

	err := e.txm.Wrap(ctx, func(txCtx context.Context) error {

		err := e.ea.RemoveReserve(txCtx, ev.ID)

		if err != nil {
			return err
		}

		err = e.ea.ChangeStatus(txCtx, ev.ID)

		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (e *eventUsesace) ReserveNewEvents(ctx context.Context, limit int, reserveDuration time.Duration) ([]*Event, error) {

	const op = "outbox.usecase.ReserveNewEvents"

	var events []*Event

	err := e.txm.Wrap(ctx, func(txCtx context.Context) error {
		var err error
		events, err = e.ea.FindNewAndNotReserved(txCtx, limit)

		if err != nil {
			return err
		}

		eventsIds := make([]string, len(events))

		for i := 0; i < len(events); i++ {
			eventsIds[i] = events[i].ID
		}

		err = e.ea.SetReservedToByIDs(txCtx, eventsIds, reserveDuration)

		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		if errors.Is(err, ErrNoNewEvents) {
			return nil, fmt.Errorf("%s: %w: %v", op, ErrNoNewEvents, err)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return events, nil
}
