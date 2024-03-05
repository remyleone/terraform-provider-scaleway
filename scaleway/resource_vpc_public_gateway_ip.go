package scaleway

import (
	"context"
	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/scaleway/errors"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/locality/zonal"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/types"
	"time"

	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/project"

	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/organization"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	vpcgw "github.com/scaleway/scaleway-sdk-go/api/vpcgw/v1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func resourceScalewayVPCPublicGatewayIP() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceScalewayVPCPublicGatewayIPCreate,
		ReadContext:   resourceScalewayVPCPublicGatewayIPRead,
		UpdateContext: resourceScalewayVPCPublicGatewayIPUpdate,
		DeleteContext: resourceScalewayVPCPublicGatewayIPDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		SchemaVersion: 0,
		Schema: map[string]*schema.Schema{
			"address": {
				Type:        schema.TypeString,
				Description: "the IP itself",
				Computed:    true,
			},
			"tags": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "The tags associated with public gateway IP",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"project_id": project.ProjectIDSchema(),
			"zone":       zonal.Schema(),
			// Computed elements
			"organization_id": organization.OrganizationIDSchema(),
			"reverse": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "reverse domain name for the IP address",
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The date and time of the creation of the public gateway IP",
			},
			"updated_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The date and time of the last update of the public gateway IP",
			},
		},
	}
}

func resourceScalewayVPCPublicGatewayIPCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	vpcgwAPI, zone, err := vpcgwAPIWithZone(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	req := &vpcgw.CreateIPRequest{
		Tags:      expandStrings(d.Get("tags")),
		ProjectID: d.Get("project_id").(string),
		Zone:      zone,
	}

	res, err := vpcgwAPI.CreateIP(req, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(zonal.NewZonedIDString(zone, res.ID))

	reverse := d.Get("reverse")
	if len(reverse.(string)) > 0 {
		updateRequest := &vpcgw.UpdateIPRequest{
			IPID:    res.ID,
			Zone:    zone,
			Tags:    scw.StringsPtr(expandStrings(d.Get("tags"))),
			Reverse: types.ExpandStringPtr(reverse.(string)),
		}
		_, err = vpcgwAPI.UpdateIP(updateRequest, scw.WithContext(ctx))
		if err != nil {
			return diag.FromErr(err)
		}
	}

	return resourceScalewayVPCPublicGatewayIPRead(ctx, d, meta)
}

func resourceScalewayVPCPublicGatewayIPRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	vpcgwAPI, zone, ID, err := vpcgwAPIWithZoneAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	ip, err := vpcgwAPI.GetIP(&vpcgw.GetIPRequest{
		IPID: ID,
		Zone: zone,
	}, scw.WithContext(ctx))
	if err != nil {
		if http_errors.Is404Error(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	_ = d.Set("organization_id", ip.OrganizationID)
	_ = d.Set("address", ip.Address.String())
	_ = d.Set("project_id", ip.ProjectID)
	_ = d.Set("created_at", ip.CreatedAt.Format(time.RFC3339))
	_ = d.Set("updated_at", ip.UpdatedAt.Format(time.RFC3339))
	_ = d.Set("zone", zone)
	_ = d.Set("tags", ip.Tags)
	_ = d.Set("reverse", ip.Reverse)

	return nil
}

func resourceScalewayVPCPublicGatewayIPUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	vpcgwAPI, zone, ID, err := vpcgwAPIWithZoneAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	updateRequest := &vpcgw.UpdateIPRequest{
		IPID: ID,
		Zone: zone,
	}

	hasChanged := false

	if d.HasChange("tags") {
		updateRequest.Tags = expandUpdatedStringsPtr(d.Get("tags"))
		hasChanged = true
	}

	if d.HasChange("reverse") {
		updateRequest.Reverse = types.ExpandStringPtr(d.Get("reverse").(string))
		hasChanged = true
	}

	if hasChanged {
		_, err = vpcgwAPI.UpdateIP(updateRequest, scw.WithContext(ctx))
		if err != nil {
			return diag.FromErr(err)
		}
	}

	return resourceScalewayVPCPublicGatewayIPRead(ctx, d, meta)
}

func resourceScalewayVPCPublicGatewayIPDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	vpcgwAPI, zone, ID, err := vpcgwAPIWithZoneAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	var warnings diag.Diagnostics
	err = vpcgwAPI.DeleteIP(&vpcgw.DeleteIPRequest{
		IPID: ID,
		Zone: zone,
	}, scw.WithContext(ctx))
	if err != nil {
		if http_errors.Is409Error(err) || http_errors.Is412Error(err) || http_errors.Is404Error(err) {
			return append(warnings, diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  err.Error(),
			})
		}
		return diag.FromErr(err)
	}

	return nil
}
