package outbox

import (
	"fmt"
	"strings"
	"time"
)

type Config struct {
	// Limit of events to receive, min = 1, Max = 100
	Limit int
	// Num of workers for publish events, min = 1, max = 10
	Workers int
	// Event processing interval, min = 5s
	Interval time.Duration
	// Event reserve duration, min = 2m
	ReserveDuration time.Duration
	// Timeout for processing method, min = 1100 ms
	ProcessTimeout time.Duration
}

func validateConfig(cfg *Config) error {
	const (
		minWorkers        = 1
		maxWorkers        = 10
		minLimit          = 1
		maxLimit          = 100
		minInterval       = 5 * time.Second
		minReserve        = 2 * time.Minute
		minProcessTimeout = 1100 * time.Millisecond
	)

	var errs []string

	if cfg.Workers < minWorkers || cfg.Workers > maxWorkers {
		errs = append(errs, fmt.Sprintf("workers must be in [%d;%d]", minWorkers, maxWorkers))
	}

	if cfg.Limit < minLimit || cfg.Limit > maxLimit {
		errs = append(errs, fmt.Sprintf("limit must be in [%d;%d]", minLimit, maxLimit))
	}

	if cfg.Interval < minInterval {
		errs = append(errs, fmt.Sprintf("interval must be >= %s", minInterval))
	}

	if cfg.ReserveDuration < minReserve {
		errs = append(errs, fmt.Sprintf("reserveDuration must be >= %s", minReserve))
	}

	if cfg.ProcessTimeout < minProcessTimeout {
		errs = append(errs, fmt.Sprintf("processTimeout must be >= %s", minProcessTimeout))
	}

	if len(errs) > 0 {
		return fmt.Errorf("invalid config:\n - %s", strings.Join(errs, "\n - "))
	}

	return nil
}
