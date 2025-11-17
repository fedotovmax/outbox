package outbox

import "time"

type Config struct {
	// Limit of events to receive, min = 1, Max = 100
	Limit int
	// Min = 1, Max = 10
	Workers  int
	Interval time.Duration
	// Event reserve duration
	ReserveDuration time.Duration
	// Timeout for methods process, confirmSuccess, confirmFailed
	ProcessTimeout time.Duration
}

func validateConfig(cfg *Config) {
	const minWorkers = 1
	const maxWorkers = 10

	const minLimit = 1
	const maxLimit = 100

	const minInterval = time.Second * 5

	const minReserve = time.Second * 25

	const minProcessTimeout = time.Millisecond * 1100

	if cfg.Workers < minWorkers {
		cfg.Workers = minWorkers
	}

	if cfg.Workers > maxWorkers {
		cfg.Workers = maxWorkers
	}

	if cfg.Limit < minLimit {
		cfg.Limit = minLimit
	}

	if cfg.Limit > maxLimit {
		cfg.Limit = maxLimit
	}

	if cfg.Interval < minInterval {
		cfg.Interval = minInterval
	}

	if cfg.ReserveDuration < minReserve {
		cfg.ReserveDuration = minReserve
	}

	if cfg.ProcessTimeout < minProcessTimeout {
		cfg.ProcessTimeout = minProcessTimeout
	}
}
