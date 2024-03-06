package locality

import (
	"fmt"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/verify"
)

func UUIDWithLocality() func(interface{}, string) ([]string, []error) {
	return func(v interface{}, key string) (warnings []string, errors []error) {
		uuid, isString := v.(string)
		if !isString {
			errors = []error{fmt.Errorf("invalid UUID for key '%s': not a string", key)}
			return
		}
		_, subUUID, err := ParseLocalizedID(uuid)
		if err != nil {
			errors = []error{fmt.Errorf("invalid UUID with locality for key  '%s': '%s' (%d): format should be 'locality/uuid'", key, uuid, len(uuid))}
			return
		}
		return verify.UUID()(subUUID, key)
	}
}

// UUID validates the schema is a UUID or the combination of a locality and a UUID
// e.g. "6ba7b810-9dad-11d1-80b4-00c04fd430c8" or "fr-par-1/6ba7b810-9dad-11d1-80b4-00c04fd430c8".
func UUIDorUUIDWithLocality() func(interface{}, string) ([]string, []error) {
	return func(v interface{}, key string) ([]string, []error) {
		return verify.UUID()(ExpandID(v), key)
	}
}
