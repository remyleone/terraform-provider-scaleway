package locality

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/verify"
)

// RegionalSchema returns a standard schema for a zone
func RegionalSchema() *schema.Schema {
	return &schema.Schema{
		Type:             schema.TypeString,
		Description:      "The region you want to attach the resource to",
		Optional:         true,
		ForceNew:         true,
		Computed:         true,
		ValidateDiagFunc: verify.StringInSliceWithWarning(AllRegions(), "region"),
	}
}

// ZoneComputedSchema returns a standard schema for a zone
func ZoneComputedSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeString,
		Description: "The zone of the resource",
		Computed:    true,
	}
}

// RegionComputedSchema returns a standard schema for a region
func RegionComputedSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeString,
		Description: "The region of the resource",
		Computed:    true,
	}
}
