package errs

import (
	"errors"
	"net/http"
	"testing"

	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/stretchr/testify/assert"
)

func TestIsHTTPCodeError(t *testing.T) {
	assert.True(t, isHTTPCodeError(&scw.ResponseError{StatusCode: http.StatusBadRequest}, http.StatusBadRequest))
	assert.False(t, isHTTPCodeError(nil, http.StatusBadRequest))
	assert.False(t, isHTTPCodeError(&scw.ResponseError{StatusCode: http.StatusBadRequest}, http.StatusNotFound))
	assert.False(t, isHTTPCodeError(errors.New("not an http error"), http.StatusNotFound))
}

func TestIs404Error(t *testing.T) {
	assert.True(t, Is404Error(&scw.ResponseError{StatusCode: http.StatusNotFound}))
	assert.False(t, Is404Error(nil))
	assert.False(t, Is404Error(&scw.ResponseError{StatusCode: http.StatusBadRequest}))
}

func TestIs403Error(t *testing.T) {
	assert.True(t, Is403Error(&scw.ResponseError{StatusCode: http.StatusForbidden}))
	assert.False(t, Is403Error(nil))
	assert.False(t, Is403Error(&scw.ResponseError{StatusCode: http.StatusBadRequest}))
}
