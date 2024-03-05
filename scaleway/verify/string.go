package verify

import (
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// StringInSliceWithWarning helps to only returns warnings in case we got a non public locality passed
func StringInSliceWithWarning(correctValues []string, field string) func(i interface{}, path cty.Path) diag.Diagnostics {
	return func(i interface{}, path cty.Path) diag.Diagnostics {
		_, rawErr := validation.StringInSlice(correctValues, true)(i, field)
		var res diag.Diagnostics
		for _, e := range rawErr {
			res = append(res, diag.Diagnostic{
				Severity:      diag.Warning,
				Summary:       e.Error(),
				AttributePath: path,
			})
		}
		return res
	}
}
