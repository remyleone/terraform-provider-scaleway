package verify_test

import (
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/verify"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidationUUIDWithInvalidUUIDReturnError(t *testing.T) {
	assert := assert.New(t)

	for _, uuid := range []string{"fr-par/wrong-uuid/resource", "fr-par/wrong-uuid", "wrong-uuid"} {
		warnings, errors := verify.UUID()(uuid, "key")
		assert.Empty(warnings)
		assert.Len(errors, 1)
	}
}

func TestValidationUUIDWithValidUUIDReturnNothing(t *testing.T) {
	assert := assert.New(t)

	warnings, errors := verify.UUID()("6ba7b810-9dad-11d1-80b4-00c04fd430c8", "key")

	assert.Empty(warnings)
	assert.Empty(errors)
}

func TestValidationUUIDorUUIDWithLocalityWithValidUUIDReturnNothing(t *testing.T) {
	assert := assert.New(t)

	for _, uuid := range []string{"fr-par/6ba7b810-9dad-11d1-80b4-00c04fd430c8", "6ba7b810-9dad-11d1-80b4-00c04fd430c8"} {
		warnings, errors := locality.UUIDorUUIDWithLocality()(uuid, "key")
		assert.Empty(warnings)
		assert.Empty(errors)
	}
}

func TestValidationUUIDorUUIDWithLocalityWithInvalidUUIDReturnError(t *testing.T) {
	assert := assert.New(t)

	for _, uuid := range []string{"fr-par/wrong-uuid/resource", "fr-par/wrong-uuid", "wrong-uuid"} {
		warnings, errors := locality.UUIDorUUIDWithLocality()(uuid, "key")
		assert.Empty(warnings)
		assert.Len(errors, 1)
	}
}

func TestValidationUUIDWithLocalityWithValidUUIDReturnNothing(t *testing.T) {
	assert := assert.New(t)

	for _, uuid := range []string{"fr-par/6ba7b810-9dad-11d1-80b4-00c04fd430c8"} {
		warnings, errors := locality.UUIDWithLocality()(uuid, "key")
		assert.Empty(warnings)
		assert.Empty(errors)
	}
}

func TestValidationUUIDWithLocalityWithInvalidUUIDReturnError(t *testing.T) {
	assert := assert.New(t)

	for _, uuid := range []string{"fr-par/wrong-uuid/resource", "fr-par/wrong-uuid", "wrong-uuid", "6ba7b810-9dad-11d1-80b4-00c04fd430c8"} {
		warnings, errors := locality.UUIDWithLocality()(uuid, "key")
		assert.Empty(warnings)
		assert.Len(errors, 1, uuid)
	}
}

func TestValidateStandaloneIPorCIDRWithValidIPReturnNothing(t *testing.T) {
	assert := assert.New(t)

	for _, ip := range []string{"192.168.1.1", "2001:0db8:85a3:0000:0000:8a2e:0370:7334", "10.0.0.0/24", "2001:0db8:85a3::8a2e:0370:7334/64"} {
		warnings, errors := verify.StandaloneIPorCIDR()(ip, "key")
		assert.Empty(warnings)
		assert.Empty(errors)
	}
}

func TestValidateStandaloneIPorCIDRWithInvalidIPReturnError(t *testing.T) {
	assert := assert.New(t)

	for _, ip := range []string{"10.0.0", "256.256.256.256", "2001::85a3::8a2e:0370:7334", "10.0.0.0/34"} {
		warnings, errors := verify.StandaloneIPorCIDR()(ip, "key")
		assert.Empty(warnings)
		assert.Len(errors, 1)
	}
}
