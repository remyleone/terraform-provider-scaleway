package zonal

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/verify"
)

// Schema returns a standard schema for a zone
func Schema() *schema.Schema {
	return &schema.Schema{
		Type:             schema.TypeString,
		Description:      "The zone you want to attach the resource to",
		Optional:         true,
		ForceNew:         true,
		Computed:         true,
		ValidateDiagFunc: verify.StringInSliceWithWarning(All(), "zone"),
	}
}

func All() []string {
	zones := make([]string, 0, len(scw.AllZones))
	for _, z := range scw.AllZones {
		zones = append(zones, z.String())
	}

	return zones
}
