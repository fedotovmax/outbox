package outbox

import "errors"

var ErrInternal = errors.New("internal error")

var ErrNoNewEvents = errors.New("err no new events")

var ErrMissUpdate = errors.New("unable to update all events")
