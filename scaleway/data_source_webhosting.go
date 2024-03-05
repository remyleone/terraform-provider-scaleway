package scaleway

import (
	"context"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/organization"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/types"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/verify"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	webhosting "github.com/scaleway/scaleway-sdk-go/api/webhosting/v1alpha1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func dataSourceScalewayWebhosting() *schema.Resource {
	dsSchema := datasourceSchemaFromResourceSchema(resourceScalewayWebhosting().Schema)

	addOptionalFieldsToSchema(dsSchema, "domain")

	dsSchema["domain"].ConflictsWith = []string{"webhosting_id"}
	dsSchema["webhosting_id"] = &schema.Schema{
		Type:          schema.TypeString,
		Optional:      true,
		Description:   "The ID of the Webhosting",
		ValidateFunc:  verify.UUIDorUUIDWithLocality(),
		ConflictsWith: []string{"domain"},
	}
	dsSchema["organization_id"] = organization.OrganizationIDOptionalSchema()
	dsSchema["project_id"] = &schema.Schema{
		Type:         schema.TypeString,
		Optional:     true,
		Description:  "The project ID the resource is associated to",
		ValidateFunc: verify.UUID(),
	}

	return &schema.Resource{
		Schema:      dsSchema,
		ReadContext: dataSourceScalewayWebhostingRead,
	}
}

func dataSourceScalewayWebhostingRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, err := webhostingAPIWithRegion(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	webhostingID, ok := d.GetOk("webhosting_id")
	if !ok {
		hostingDomain := d.Get("domain").(string)
		res, err := api.ListHostings(&webhosting.ListHostingsRequest{
			Region:         region,
			Domain:         types.ExpandStringPtr(hostingDomain),
			ProjectID:      types.ExpandStringPtr(d.Get("project_id")),
			OrganizationID: types.ExpandStringPtr(d.Get("organization_id")),
		}, scw.WithContext(ctx))
		if err != nil {
			return diag.FromErr(err)
		}

		foundDomain, err := findExact(
			res.Hostings,
			func(s *webhosting.Hosting) bool { return s.Domain == hostingDomain },
			hostingDomain,
		)
		if err != nil {
			return diag.FromErr(err)
		}

		webhostingID = foundDomain.ID
	}

	regionalID := datasourceNewRegionalID(webhostingID, region)
	d.SetId(regionalID)
	err = d.Set("webhosting_id", regionalID)
	if err != nil {
		return diag.FromErr(err)
	}

	diags := resourceScalewayWebhostingRead(ctx, d, meta)
	if diags != nil {
		return append(diags, diag.Errorf("failed to read hosting")...)
	}

	if d.Id() == "" {
		return diag.Errorf("hosting (%s) not found", regionalID)
	}

	return nil
}
