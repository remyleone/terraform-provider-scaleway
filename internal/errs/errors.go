package errs

import (
	"errors"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"strings"
)

// NotFound returns true if the error represents a "resource not found" condition.
// Specifically, NotFound returns true if the error or a wrapped error is of type
// retry.NotFoundError.
func NotFound(err error) bool {
	var e *retry.NotFoundError // nosemgrep:ci.is-not-found-error
	return errors.As(err, &e)
}

type TooManyResultsError struct {
	Count       int
	LastRequest interface{}
}

func (e *TooManyResultsError) Error() string {
	return fmt.Sprintf("too many results: wanted 1, got %d", e.Count)
}

func (e *TooManyResultsError) Is(err error) bool {
	_, ok := err.(*TooManyResultsError) //nolint:errorlint // Explicitly does *not* match down the error tree
	return ok
}

func (e *TooManyResultsError) As(target interface{}) bool {
	t, ok := target.(**retry.NotFoundError)
	if !ok {
		return false
	}

	*t = &retry.NotFoundError{
		Message:     e.Error(),
		LastRequest: e.LastRequest,
	}

	return true
}

var ErrTooManyResults = &TooManyResultsError{}

func ErrorCheck(err error, message string) bool {
	return strings.Contains(err.Error(), message)
}

// ErrCodeEquals returns true if the error matches all these conditions:
//   - err is of type scw.Error
//   - Error.Error() equals one of the passed codes
func ErrCodeEquals(err error, codes ...string) bool {
	var scwErr scw.SdkError
	if errors.As(err, &scwErr) {
		for _, code := range codes {
			if scwErr.Error() == code {
				return true
			}
		}
	}
	return false
}
