package vpc

import (
	"context"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/datasource"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/organization"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/types"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/verify"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/scaleway/scaleway-sdk-go/api/vpc/v2"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func DataSourceScalewayVPC() *schema.Resource {
	dsSchema := datasource.DatasourceSchemaFromResourceSchema(ResourceScalewayVPC().Schema)

	datasource.AddOptionalFieldsToSchema(dsSchema, "name", "is_default", "region")

	dsSchema["name"].ConflictsWith = []string{"vpc_id"}
	dsSchema["vpc_id"] = &schema.Schema{
		Type:          schema.TypeString,
		Optional:      true,
		Description:   "The ID of the VPC",
		ValidateFunc:  locality.UUIDorUUIDWithLocality(),
		ConflictsWith: []string{"name"},
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
		ReadContext: DataSourceScalewayVPCRead,
	}
}

func DataSourceScalewayVPCRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	vpcAPI, region, err := vpcAPIWithRegion(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	var vpcID interface{}
	var ok bool

	if d.Get("is_default").(bool) {
		request := &vpc.ListVPCsRequest{
			IsDefault: types.ExpandBoolPtr(d.Get("is_default").(bool)),
			Region:    region,
			ProjectID: types.ExpandStringPtr(d.Get("project_id")),
		}

		res, err := vpcAPI.ListVPCs(request, scw.WithContext(ctx))
		if err != nil {
			return diag.FromErr(err)
		}

		vpcID = locality.NewRegionalIDString(region, res.Vpcs[0].ID)
	} else {
		vpcID, ok = d.GetOk("vpc_id")
		if !ok {
			vpcName := d.Get("name").(string)
			request := &vpc.ListVPCsRequest{
				Name:           types.ExpandStringPtr(vpcName),
				Region:         region,
				ProjectID:      types.ExpandStringPtr(d.Get("project_id")),
				OrganizationID: types.ExpandStringPtr(d.Get("organization_id")),
			}

			res, err := vpcAPI.ListVPCs(request, scw.WithContext(ctx))
			if err != nil {
				return diag.FromErr(err)
			}

			foundVPC, err := datasource.FindExact(
				res.Vpcs,
				func(s *vpc.VPC) bool { return s.Name == vpcName },
				vpcName,
			)
			if err != nil {
				return diag.FromErr(err)
			}

			vpcID = foundVPC.ID
		}
	}

	regionalID := locality.DatasourceNewRegionalID(vpcID, region)
	d.SetId(regionalID)
	err = d.Set("vpc_id", regionalID)
	if err != nil {
		return diag.FromErr(err)
	}

	diags := ResourceScalewayVPCRead(ctx, d, meta)
	if diags != nil {
		return append(diags, diag.Errorf("failed to read VPC")...)
	}

	if d.Id() == "" {
		return diag.Errorf("VPC (%s) not found", regionalID)
	}

	return nil
}
