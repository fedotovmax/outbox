package outbox

import (
	"fmt"
	"strings"
	"time"
)

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

func validateConfig(cfg *Config) error {
	const (
		minWorkers        = 1
		maxWorkers        = 10
		minLimit          = 1
		maxLimit          = 100
		minInterval       = 5 * time.Second
		minReserve        = 25 * time.Second
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
