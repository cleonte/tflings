package exercises

import "context"

type progressKey struct{}

// WithProgress returns a context whose runs report what they are doing, in a
// few words ("terraform init"), to fn. The full-screen view shows it while a
// slow run is going on. fn may be called from another goroutine.
func WithProgress(ctx context.Context, fn func(line string)) context.Context {
	return context.WithValue(ctx, progressKey{}, fn)
}

func report(ctx context.Context, line string) {
	if fn, ok := ctx.Value(progressKey{}).(func(string)); ok && fn != nil {
		fn(line)
	}
}
