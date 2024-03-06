package errs

import "github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

// TimedOut returns true if the error represents a "wait timed out" condition.
// Specifically, TimedOut returns true if the error matches all these conditions:
//   - err is of type resource.TimeoutError
//   - TimeoutError.LastError is nil
func TimedOut(err error) bool {
	// This explicitly does *not* match wrapped TimeoutErrors
	timeoutErr, ok := err.(*resource.TimeoutError) //nolint:errorlint // Explicitly does *not* match wrapped TimeoutErrors
	return ok && timeoutErr.LastError == nil
}
