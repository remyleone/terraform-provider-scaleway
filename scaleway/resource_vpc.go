package scaleway

import (
	"context"
	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/scaleway/errors"

	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/project"

	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/organization"

	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/types"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/scaleway/scaleway-sdk-go/api/vpc/v2"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func resourceScalewayVPC() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceScalewayVPCCreate,
		ReadContext:   resourceScalewayVPCRead,
		UpdateContext: resourceScalewayVPCUpdate,
		DeleteContext: resourceScalewayVPCDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		SchemaVersion: 0,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The name of the VPC",
				Computed:    true,
			},
			"tags": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "The tags associated with the VPC",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"project_id": project.ProjectIDSchema(),
			"region":     regionSchema(),
			// Computed elements
			"organization_id": organization.OrganizationIDSchema(),
			"is_default": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Defines whether the VPC is the default one for its Project",
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The date and time of the creation of the private network",
			},
			"updated_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The date and time of the last update of the private network",
			},
		},
	}
}

func resourceScalewayVPCCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	vpcAPI, region, err := vpcAPIWithRegion(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	res, err := vpcAPI.CreateVPC(&vpc.CreateVPCRequest{
		Name:      types.ExpandOrGenerateString(d.Get("name"), "vpc"),
		Tags:      expandStrings(d.Get("tags")),
		ProjectID: d.Get("project_id").(string),
		Region:    region,
	}, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(newRegionalIDString(region, res.ID))

	return resourceScalewayVPCRead(ctx, d, meta)
}

func resourceScalewayVPCRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	vpcAPI, region, ID, err := vpcAPIWithRegionAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	res, err := vpcAPI.GetVPC(&vpc.GetVPCRequest{
		Region: region,
		VpcID:  ID,
	}, scw.WithContext(ctx))
	if err != nil {
		if http_errors.Is404Error(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	_ = d.Set("name", res.Name)
	_ = d.Set("organization_id", res.OrganizationID)
	_ = d.Set("project_id", res.ProjectID)
	_ = d.Set("created_at", flattenTime(res.CreatedAt))
	_ = d.Set("updated_at", flattenTime(res.UpdatedAt))
	_ = d.Set("is_default", res.IsDefault)
	_ = d.Set("region", region)

	if len(res.Tags) > 0 {
		_ = d.Set("tags", res.Tags)
	}

	return nil
}

func resourceScalewayVPCUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	vpcAPI, region, ID, err := vpcAPIWithRegionAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = vpcAPI.UpdateVPC(&vpc.UpdateVPCRequest{
		VpcID:  ID,
		Region: region,
		Name:   scw.StringPtr(d.Get("name").(string)),
		Tags:   expandUpdatedStringsPtr(d.Get("tags")),
	}, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceScalewayVPCRead(ctx, d, meta)
}

func resourceScalewayVPCDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	vpcAPI, region, ID, err := vpcAPIWithRegionAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	err = vpcAPI.DeleteVPC(&vpc.DeleteVPCRequest{
		Region: region,
		VpcID:  ID,
	}, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}
