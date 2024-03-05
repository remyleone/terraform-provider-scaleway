package verify

import (
	"fmt"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/locality"
	"net"

	"github.com/scaleway/scaleway-sdk-go/validation"
)

// UUID validates the schema is a UUID or the combination of a locality and a UUID
// e.g. "6ba7b810-9dad-11d1-80b4-00c04fd430c8" or "fr-par-1/6ba7b810-9dad-11d1-80b4-00c04fd430c8".
func UUIDorUUIDWithLocality() func(interface{}, string) ([]string, []error) {
	return func(v interface{}, key string) ([]string, []error) {
		return UUID()(locality.ExpandID(v), key)
	}
}

// UUID validates the schema following the canonical UUID format
// "6ba7b810-9dad-11d1-80b4-00c04fd430c8".
func UUID() func(interface{}, string) ([]string, []error) {
	return func(v interface{}, key string) (warnings []string, errors []error) {
		uuid, isString := v.(string)
		if !isString {
			return nil, []error{fmt.Errorf("invalid UUID for key '%s': not a string", key)}
		}

		if !validation.IsUUID(uuid) {
			return nil, []error{fmt.Errorf("invalid UUID for key '%s': '%s' (%d): format should be 'xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx' (36) and contains valid hexadecimal characters", key, uuid, len(uuid))}
		}

		return
	}
}

func UUIDWithLocality() func(interface{}, string) ([]string, []error) {
	return func(v interface{}, key string) (warnings []string, errors []error) {
		uuid, isString := v.(string)
		if !isString {
			errors = []error{fmt.Errorf("invalid UUID for key '%s': not a string", key)}
			return
		}
		_, subUUID, err := locality.ParseLocalizedID(uuid)
		if err != nil {
			errors = []error{fmt.Errorf("invalid UUID with locality for key  '%s': '%s' (%d): format should be 'locality/uuid'", key, uuid, len(uuid))}
			return
		}
		return UUID()(subUUID, key)
	}
}

func Email() func(interface{}, string) ([]string, []error) {
	return func(v interface{}, key string) (warnings []string, errors []error) {
		email, isString := v.(string)
		if !isString {
			return nil, []error{fmt.Errorf("invalid email for key '%s': not a string", key)}
		}

		if !validation.IsEmail(email) {
			return nil, []error{fmt.Errorf("invalid email for key '%s': '%s': should contain valid '@' character", key, email)}
		}

		return
	}
}

func StandaloneIPorCIDR() func(interface{}, string) ([]string, []error) {
	return func(val interface{}, key string) (warns []string, errs []error) {
		ip, isString := val.(string)
		if !isString {
			return nil, []error{fmt.Errorf("invalid input for key '%s': not a string", key)}
		}

		// Check if it's a standalone IP address
		if net.ParseIP(ip) != nil {
			return
		}

		// Check if it's an IP with CIDR notation
		_, _, err := net.ParseCIDR(ip)
		if err != nil {
			errs = append(errs, fmt.Errorf("%q is not a valid IP address or CIDR notation: %s", key, ip))
		}

		return
	}
}
