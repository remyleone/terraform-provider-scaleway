package project

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/verify"
)

// ProjectIDSchema returns a standard schema for a project_id
func ProjectIDSchema() *schema.Schema {
	return &schema.Schema{
		Type:         schema.TypeString,
		Description:  "The project_id you want to attach the resource to",
		Optional:     true,
		ForceNew:     true,
		Computed:     true,
		ValidateFunc: verify.UUID(),
	}
}
