/*
Package backoff provides a backoff object that makes it easy to consistently
apply exponential backoff with jitter.

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
	b.Sleep() // wait 425~575ms, base delay of 500ms +/- 15%, the default jitter
	b.Sleep() // wait 850~1150ms  		~1s
	b.Sleep() // wait 1700~2300ms 		~2s
	b.Sleep() // wait 3400-4600ms 		~4s
	b.Sleep() // wait 6800~9200ms 		~8s
	b.Sleep() // wait 1360~1840ms 		~16s
	b.Sleep() // wait 27200~36800ms 	~32s
	b.Sleep() // wait 54400~73600ms 	~64s
	b.Sleep() // wait 54400~73600ms 	~64s (stopped growing, jitter still applied)
*/
package backoff

import (
	"errors"
	"math"
	"math/rand"
	"time"
)

type backoffOption func(*Backoff, bool) error

// Backoff provides exponential backoff with jitter. By default, the initial
// backoff is 100ms, the jitter factor is 0.3 (so +/- 15%), and exponential grow
// stops once the backoff reaches 3 minutes.
type Backoff struct {
	delay        time.Duration
	baseDelay    time.Duration
	expLimit     time.Duration
	jitterFactor float64
}

var (
	defaultInitDelay = time.Millisecond * 100
	defaultBaseDelay = time.Millisecond * 100
	defaultExpLimit  = time.Minute * 3
)

const defaultJitterFactor = 0.3

func defaultBackoff() *Backoff {
	return &Backoff{
		delay:        defaultInitDelay,
		baseDelay:    defaultBaseDelay,
		expLimit:     defaultExpLimit,
		jitterFactor: defaultJitterFactor,
	}
}

// New creates a new exponential backoff object. Use the Sleep() method to pause
// using exponential backoff with jitter. The default initial delay is 100ms,
// with a jitter of +/- 30%, and exponential growth until the delay reaches 3
// minutes.
func New(options ...backoffOption) (*Backoff, error) {
	b := defaultBackoff()

	var errs error
	for i := 0; i < len(options); i++ {
		errs = errors.Join(errs, options[i](b, false))
	}
	if errs != nil {
		return nil, errs
	}

	return b, nil
}

// CoerceNew creates a new exponential backoff object, coercing invalid options
// to valid values, to guarantee that it returns a valid backoff.
func CoerceNew(options ...backoffOption) *Backoff {
	b := defaultBackoff()

	for i := 0; i < len(options); i++ {
		options[i](b, true)
	}

	return b
}

// WithInitialDelay configuration BackoffOption allows customization of the
// initial backoff delay (before jitter). It is safe to set this to 0, allowing
// the first retry to occur immediately, then after the first delay it will
// still grow exponentially from the `BaseDelay`. The default initial backoff
// delay is 100ms.
func WithInitialDelay(d time.Duration) backoffOption {
	return func(b *Backoff, coerce bool) error {
		if d >= 0 {
			b.delay = d
			return nil
		}
		if !coerce {
			return errors.New("the initial delay must be >= 0")
		}
		// assume caller wanted immediate initial retry
		b.delay = 0
		return nil
	}
}

// WithBaseDelay configuration BackoffOption allows customization of the backoff
// delay (before jitter), used after the initial delay, if the initial delay is
// 0 (so, on the second call to `backoff.Sleep()` in that case). The default is
// 100ms.
//
// The base delay must be > 0, and is only used if the initial delay is 0.
func WithBaseDelay(d time.Duration) backoffOption {
	return func(b *Backoff, coerce bool) error {
		if d > 0 {
			b.baseDelay = d
			return nil

		}
		if !coerce {
			return errors.New("the base delay must be > 0")
		}

		// keep the default value
		return nil
	}
}

// WithExponentialLimit configuration BackoffOption allows customization of the
// backoff delay beyond it stops growing exponentially. It is possible to set
// the limit to 0, in which case the it will never grow beyond the base delay.
// though jitter will still be applied in call cases. The default is 3 minutes.
func WithExponentialLimit(d time.Duration) backoffOption {
	return func(b *Backoff, coerce bool) error {
		if d >= 0 {
			b.expLimit = d
			return nil
		}
		if !coerce {
			return errors.New("the exponential backoff limit must be >= 0")
		}
		// assume caller wanted zero exponential growth in the backoff
		b.expLimit = 0
		return nil
	}
}

// WithJitterFactor configuration BackoffOption allows customization of the
// jitter factor. The value must be in the range [0,1). Jitter is applied
// uniformly randomly about the backoff delay, so 0.3 represents the backoff
// delay being adjusted by +/- 15%. The default is 0.3.
func WithJitterFactor(jitterFactor float64) backoffOption {
	return func(b *Backoff, coerce bool) error {
		if jitterFactor >= 0 && jitterFactor < 1.0 {
			b.jitterFactor = jitterFactor
			return nil
		}
		if !coerce {
			return errors.New("the jitterFactor must be in the range [0,1)")
		}
		if jitterFactor < 0 {
			// assume caller wanted to disable jitter
			b.jitterFactor = 0.0
			return nil
		}

		// keep default value
		return nil
	}
}

// Sleep pauses execution on the current thread. The duration of the sleep
// increases exponentially, up to a limit, and random jitter is applied to
// mitigate the thundering herd problem.
func (b *Backoff) Sleep() {
	time.Sleep(b.computeDelay())
}

// PeekDelay allows the caller to query the hext delay without performing the
// backoff (i.e. without pausing execution or growing the backoff delay).
func (b *Backoff) PeekDelay() time.Duration {
	return b.delay
}

func (b *Backoff) computeDelay() time.Duration {
	// compute current backoff by adding jitter
	j := 1.0 + (rand.Float64()-0.5)*b.jitterFactor
	d := float64(b.delay.Nanoseconds()) * j

	// update state for the next backoff round
	if b.delay == 0.0 {
		b.delay = b.baseDelay
	} else if b.delay < b.expLimit {
		b.delay *= 2.0
	}

	return time.Duration(int(math.Round(d)))
}
