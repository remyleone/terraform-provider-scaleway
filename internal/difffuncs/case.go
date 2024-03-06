package difffuncs

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"strings"
)

func DiffSuppressFuncIgnoreCase(_, oldValue, newValue string, _ *schema.ResourceData) bool {
	return strings.EqualFold(oldValue, newValue)
}

func DiffSuppressFuncIgnoreCaseAndHyphen(_, oldValue, newValue string, _ *schema.ResourceData) bool {
	return strings.ReplaceAll(strings.ToLower(oldValue), "-", "_") == strings.ReplaceAll(strings.ToLower(newValue), "-", "_")
}
