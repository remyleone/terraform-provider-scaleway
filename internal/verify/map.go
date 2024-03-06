package verify

import (
	"fmt"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/types"
	"strings"
)

func MapKeyLowerCase() schema.SchemaValidateDiagFunc {
	return func(i interface{}, _ cty.Path) diag.Diagnostics {
		m := types.ExpandMapStringStringPtr(i)
		for k := range m {
			if strings.ToLower(k) != k {
				return diag.Diagnostics{diag.Diagnostic{
					Severity:      diag.Error,
					AttributePath: cty.IndexStringPath(k),
					Summary:       "Invalid map content",
					Detail:        fmt.Sprintf("key (%s) should be lowercase", k),
				}}
			}
		}
		return nil
	}
}
