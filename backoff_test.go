package backoff

import (
	"math"
	"reflect"
	"testing"
	"time"
)

func TestNewConstructor(t *testing.T) {
	tests := map[string]struct {
		inputs    Backoff
		expectErr bool
	}{
		"ok with default inputs":            {Backoff{defaultInitDelay, defaultBaseDelay, defaultExpLimit, defaultJitterFactor}, false},
		"ok with 0 init delay":              {Backoff{0, defaultBaseDelay, defaultExpLimit, defaultJitterFactor}, false},
		"ok with 0 exp limit":               {Backoff{defaultInitDelay, defaultBaseDelay, 0, defaultJitterFactor}, false},
		"ok with 0 jitter factor":           {Backoff{defaultInitDelay, defaultBaseDelay, defaultExpLimit, 0}, false},
		"fails with negative init delay":    {Backoff{-1, defaultBaseDelay, defaultExpLimit, defaultJitterFactor}, true},
		"fails with negative base delay":    {Backoff{defaultInitDelay, -1, defaultExpLimit, defaultJitterFactor}, true},
		"fails with 0 base delay":           {Backoff{defaultInitDelay, 0, defaultExpLimit, defaultJitterFactor}, true},
		"fails with negative exp limit":     {Backoff{defaultInitDelay, defaultBaseDelay, -1, defaultJitterFactor}, true},
		"fails with negative jitter factor": {Backoff{defaultInitDelay, defaultBaseDelay, defaultExpLimit, -1}, true},
		"fails with jitter factor == 1":     {Backoff{defaultInitDelay, defaultBaseDelay, defaultExpLimit, 1}, true},
		"fails with jitter factor > 1":      {Backoff{defaultInitDelay, defaultBaseDelay, defaultExpLimit, 1.3}, true},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := New(
				WithInitialDelay(tc.inputs.delay),
				WithBaseDelay(tc.inputs.baseDelay),
				WithExponentialLimit(tc.inputs.expLimit),
				WithJitterFactor(tc.inputs.jitterFactor),
			)
			if err == nil && tc.expectErr {
				t.Fatalf("expected error but received none")
			} else if err != nil && !tc.expectErr {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}

}

func TestCoerceNewConstructor(t *testing.T) {
	tests := map[string]struct {
		inputs  Backoff
		outputs Backoff
	}{
		"with default inputs": {
			Backoff{defaultInitDelay, defaultBaseDelay, defaultExpLimit, defaultJitterFactor},
			Backoff{defaultInitDelay, defaultBaseDelay, defaultExpLimit, defaultJitterFactor},
		},
		"with 0 init delay": {
			Backoff{0, defaultBaseDelay, defaultExpLimit, defaultJitterFactor},
			Backoff{0, defaultBaseDelay, defaultExpLimit, defaultJitterFactor},
		},
		"with 0 exp limit": {
			Backoff{defaultInitDelay, defaultBaseDelay, 0, defaultJitterFactor},
			Backoff{defaultInitDelay, defaultBaseDelay, 0, defaultJitterFactor},
		},
		"with 0 jitter factor": {
			Backoff{defaultInitDelay, defaultBaseDelay, defaultExpLimit, 0},
			Backoff{defaultInitDelay, defaultBaseDelay, defaultExpLimit, 0},
		},
		"coerce negative init delay to 0": {
			Backoff{-1, defaultBaseDelay, defaultExpLimit, defaultJitterFactor},
			Backoff{0, defaultBaseDelay, defaultExpLimit, defaultJitterFactor},
		},
		"coerce negative base delay to the default": {
			Backoff{defaultInitDelay, -1, defaultExpLimit, defaultJitterFactor},
			Backoff{defaultInitDelay, defaultBaseDelay, defaultExpLimit, defaultJitterFactor},
		},
		"coerce 0 base delay to the default": {
			Backoff{defaultInitDelay, 0, defaultExpLimit, defaultJitterFactor},
			Backoff{defaultInitDelay, defaultBaseDelay, defaultExpLimit, defaultJitterFactor},
		},
		"coerce negative exp limit to 0": {
			Backoff{defaultInitDelay, defaultBaseDelay, -1, defaultJitterFactor},
			Backoff{defaultInitDelay, defaultBaseDelay, 0, defaultJitterFactor},
		},
		"coerce negative jitter factor to zero": {
			Backoff{defaultInitDelay, defaultBaseDelay, defaultExpLimit, -1},
			Backoff{defaultInitDelay, defaultBaseDelay, defaultExpLimit, 0},
		},
		"coerce jitter factor == 1 to the default": {
			Backoff{defaultInitDelay, defaultBaseDelay, defaultExpLimit, 1},
			Backoff{defaultInitDelay, defaultBaseDelay, defaultExpLimit, defaultJitterFactor},
		},
		"coerce jitter factor > 1 to the default": {
			Backoff{defaultInitDelay, defaultBaseDelay, defaultExpLimit, 1.3},
			Backoff{defaultInitDelay, defaultBaseDelay, defaultExpLimit, defaultJitterFactor},
		},
	}
	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			b := CoerceNew(
				WithInitialDelay(tc.inputs.delay),
				WithBaseDelay(tc.inputs.baseDelay),
				WithExponentialLimit(tc.inputs.expLimit),
				WithJitterFactor(tc.inputs.jitterFactor),
			)
			if !reflect.DeepEqual(tc.outputs, *b) {
				t.Fatalf("got: %+v, want: %+v", b, &tc.outputs)
			}
		})
	}

}

func TestBaseDelay(t *testing.T) {
	tests := map[string]struct {
		inputs      Backoff
		round2Delay time.Duration
	}{
		"uses baseDelay if initial delay is 0": {
			Backoff{0, 200, defaultExpLimit, defaultJitterFactor},
			200,
		},
		"ignores baseDelay if initial delay is not 0": {
			Backoff{1, 200, defaultExpLimit, defaultJitterFactor},
			2,
		},
	}
	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			b := CoerceNew(
				WithInitialDelay(tc.inputs.delay),
				WithBaseDelay(tc.inputs.baseDelay),
				WithExponentialLimit(tc.inputs.expLimit),
				WithJitterFactor(tc.inputs.jitterFactor),
			)
			b.computeDelay()
			r2d := b.delay
			if r2d != tc.round2Delay {
				t.Fatalf("got: %+v, want: %+v", r2d, tc.round2Delay)
			}
		})
	}
}

func TestGrowthAndJitter(t *testing.T) {
	var lim time.Duration = 64
	b := CoerceNew(
		WithInitialDelay(2),
		WithExponentialLimit(lim),
	)
	bNoJitter := CoerceNew(
		WithInitialDelay(2),
		WithExponentialLimit(lim),
		WithJitterFactor(0.0),
	)
	delays := make([]time.Duration, 10)
	delaysWithJitter := make([]time.Duration, 10)
	delaysWithJitter0 := make([]time.Duration, 10)
	for i := 0; i < 10; i++ {
		delays[i] = b.delay
		delaysWithJitter[i] = b.computeDelay()
		delaysWithJitter0[i] = bNoJitter.computeDelay()

		// jitter = 0 same as the base growth before applying jitter
		if delaysWithJitter0[i] != delays[i] {
			t.Fatalf("with jitterFactor of 0, expected: %v, got %v", delays[i], delaysWithJitter0[i])
		}

	}

	reachesLimitAt := int(math.Log2(float64(lim))) - 1

	// confirm exponential growth; 2 4 8 16 32 64
	for i := 1; i <= reachesLimitAt; i++ {
		if expected := 2 * delays[i-1]; delays[i] != expected {
			t.Fatalf("exponential growth violation, expected: %v, got %v", expected, delays[i])
		}
	}

	nSame := 0
	nSameWithJitter := 0
	for i := reachesLimitAt; i < 10; i++ {
		if delays[i] == lim {
			nSame++
		}
		if delaysWithJitter[i] == lim {
			nSameWithJitter++
		}
	}

	if nSame != 10-reachesLimitAt {
		t.Fatalf("exponential growth expected to stop after round %d, all delays: %v", reachesLimitAt, delays)
	}

	if nSameWithJitter == nSame {
		t.Fatalf("jitter failure: all delays with jitter applied: %v", delaysWithJitter)
	}
}
