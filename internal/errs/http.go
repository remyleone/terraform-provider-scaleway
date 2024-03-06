package errs

import (
	"errors"
	"net/http"

	"github.com/scaleway/scaleway-sdk-go/scw"
)

// isHTTPCodeError returns true if err is an http error with code statusCode
func isHTTPCodeError(err error, statusCode int) bool {
	if err == nil {
		return false
	}

	responseError := &scw.ResponseError{}
	if errors.As(err, &responseError) && responseError.StatusCode == statusCode {
		return true
	}
	return false
}

// Is404Error returns true if err is an HTTP 404 error
func Is404Error(err error) bool {
	notFoundError := &scw.ResourceNotFoundError{}
	return isHTTPCodeError(err, http.StatusNotFound) || errors.As(err, &notFoundError)
}

func Is412Error(err error) bool {
	preConditionFailedError := &scw.PreconditionFailedError{}
	return isHTTPCodeError(err, http.StatusPreconditionFailed) || errors.As(err, &preConditionFailedError)
}

// Is403Error returns true if err is an HTTP 403 error
func Is403Error(err error) bool {
	permissionsDeniedError := &scw.PermissionsDeniedError{}
	return isHTTPCodeError(err, http.StatusForbidden) || errors.As(err, &permissionsDeniedError)
}

// Is409Error return true is err is an HTTP 409 error
func Is409Error(err error) bool {
	// check transient error
	transientStateError := &scw.TransientStateError{}
	return isHTTPCodeError(err, http.StatusConflict) || errors.As(err, &transientStateError)
}

// Is410Error returns true if err is an HTTP 410 error
func Is410Error(err error) bool {
	return isHTTPCodeError(err, http.StatusGone)
}
