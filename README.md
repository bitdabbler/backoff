# Backoff

```go
import (
    // ...
    "github.com/bitdabbler/backoff"
)
```

## Purpose

`backoff` is a tiny Go library enabling simple and consistent exponential backoff with jitter.

## Basic Usage

```go
    b := backoff.CoerceNew()

    for i := 0; i < maxRetries; i++ {
        ok := somethingFailableAndRetryable()
        if !ok {
            b.Sleep()
        }
    } 
```

## Details

### Constructors

1. `func CoerceNew(options ...BackoffOption) *Backoff`
2. `func New(options ...BackoffOption) (*Backoff, error)`

The `CoerceNew` constructor clamps option inputs to valid values to guarantee that it returns a valid Backoff.

### Options

| Option                                        | Default        |
| --------------------------------------------- | -------------- |
| `backoff.WithInitialDelay(time.Duration)`     | default 100ms  |
| `backoff.WithBaseDelay(time.Duration)`        | default 100ms  |
| `backoff.WithExponentialLimit(time.Duration)` | default 3 mins |
| `backoff.WithJitterFactor(float64)`           | default 0.3    |

If the initial backoff is 0, then the second backoff will use the base backoff value, and then grow exponentially in each subsequent backoff round.

