# Backoff

## Purpose

Backoff is a tiny Go library enabling simple and consistent exponential backoff with jitter.

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


```go
Ex.
b, err := NewBackoff(
  WithInitialDelay(0),
  WithBaseDelay(time.Millisecond * 500),
  WithExponentialLimit(time.Second * 60), // stop growing exponentially after 1 min
)
if err != nil {
    log.Fatalln(err)
}
b.Sleep() // immediately retry because initial delay = 0
b.Sleep() // wait 425~575ms, 500ms +/- 15%, the default jitter
b.Sleep() // wait 850~1150ms  		~1s
b.Sleep() // wait 1700~2300ms 		~2s
b.Sleep() // wait 3400-4600ms 		~4s
b.Sleep() // wait 6800~9200ms 		~8s
b.Sleep() // wait 1360~1840ms 		~16s
b.Sleep() // wait 27200~36800ms 	~32s
b.Sleep() // wait 54400~73600ms 	~64s
b.Sleep() // wait 54400~73600ms 	~64s (stopped growing, jitter still applied)
```