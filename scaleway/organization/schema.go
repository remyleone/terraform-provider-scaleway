package organization

import "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

func OrganizationIDOptionalSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Computed:    true,
		Description: "ID of organization the resource is associated to.",
	}
}

// OrganizationIDSchema returns a standard schema for a organization_id
func OrganizationIDSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeString,
		Description: "The organization_id you want to attach the resource to",
		Computed:    true,
	}
}
