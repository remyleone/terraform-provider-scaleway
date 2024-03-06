package verify

import (
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"time"
)

// ValidateDate will validate that field is a valid ISO 8601
// It is the same as RFC3339
func ValidateDate() schema.SchemaValidateDiagFunc {
	return func(i interface{}, _ cty.Path) diag.Diagnostics {
		date, isStr := i.(string)
		if !isStr {
			return diag.Errorf("%v is not a string", date)
		}
		_, err := time.Parse(time.RFC3339, date)
		if err != nil {
			return diag.FromErr(err)
		}
		return nil
	}
}
