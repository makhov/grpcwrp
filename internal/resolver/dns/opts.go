package dns

import "time"

type BuilderOpts func(*dnsBuilder)

// WithMinFreq sets min dns resolve frequency
func WithMinFreq(minFreq time.Duration) BuilderOpts {
	return func(b *dnsBuilder) {
		b.minFreq = minFreq
	}
}
