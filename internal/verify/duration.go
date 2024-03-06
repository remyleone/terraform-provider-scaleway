package verify

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"time"
)

func ValidateDuration() schema.SchemaValidateFunc {
	return func(i interface{}, _ string) (strings []string, errors []error) {
		str, isStr := i.(string)
		if !isStr {
			return nil, []error{fmt.Errorf("%v is not a string", i)}
		}
		_, err := time.ParseDuration(str)
		if err != nil {
			return nil, []error{fmt.Errorf("cannot parse duration for value %s", str)}
		}
		return nil, nil
	}
}
