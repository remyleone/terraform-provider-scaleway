package transport

import (
	"context"
	"github.com/hashicorp/aws-sdk-go-base/tfawserr"
)

// RetryWhenAWSErrCodeEquals retries a function when it returns a specific AWS error
func RetryWhenAWSErrCodeEquals[T any](ctx context.Context, codes []string, config *RetryWhenConfig[T]) (T, error) { //nolint: ireturn
	return retryWhen(ctx, config, func(err error) bool {
		return tfawserr.ErrCodeEquals(err, codes...)
	})
}

// RetryWhenAWSErrCodeNotEquals retries a function until it returns a specific AWS error
func RetryWhenAWSErrCodeNotEquals[T any](ctx context.Context, codes []string, config *RetryWhenConfig[T]) (T, error) { //nolint: ireturn
	return retryWhen(ctx, config, func(err error) bool {
		if err == nil {
			return true
		}

		return !tfawserr.ErrCodeEquals(err, codes...)
	})
}
