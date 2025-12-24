package pipeline

import "errors"

var ErrCircuitOpen = errors.New("circuit breaker open")
