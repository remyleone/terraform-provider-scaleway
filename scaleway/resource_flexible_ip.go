package scaleway

import (
	"context"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/locality"

	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/scaleway/errors"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/locality/zonal"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/types"

	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/project"

	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/organization"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	flexibleip "github.com/scaleway/scaleway-sdk-go/api/flexibleip/v1alpha1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func resourceScalewayFlexibleIP() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceScalewayFlexibleIPCreate,
		ReadContext:   resourceScalewayFlexibleIPRead,
		UpdateContext: resourceScalewayFlexibleIPUpdate,
		DeleteContext: resourceScalewayFlexibleIPDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Timeouts: &schema.ResourceTimeout{
			Create:  schema.DefaultTimeout(defaultFlexibleIPTimeout),
			Read:    schema.DefaultTimeout(defaultFlexibleIPTimeout),
			Update:  schema.DefaultTimeout(defaultFlexibleIPTimeout),
			Delete:  schema.DefaultTimeout(defaultFlexibleIPTimeout),
			Default: schema.DefaultTimeout(defaultFlexibleIPTimeout),
		},
		SchemaVersion: 0,
		Schema: map[string]*schema.Schema{
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Description of the flexible IP",
			},
			"is_ipv6": {
				Type:        schema.TypeBool,
				Optional:    true,
				ForceNew:    true,
				Default:     false,
				Description: "Defines whether the flexible IP has an IPv6 address",
			},
			"reverse": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The reverse DNS for this flexible IP",
				Computed:    true,
			},
			"server_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The baremetal server associated with this flexible IP",
			},
			"tags": {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional:    true,
				Description: "The tags associated with the flexible IP",
			},
			"zone":            zonal.Schema(),
			"organization_id": organization.OrganizationIDSchema(),
			"project_id":      project.ProjectIDSchema(),
			"ip_address": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The IP address of the flexible IP",
			},
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The status of the flexible IP",
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The date and time of the creation of the Flexible IP (Format ISO 8601)",
			},
			"updated_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The date and time of the last update of the Flexible IP (Format ISO 8601)",
			},
		},
		CustomizeDiff: customizeDiffLocalityCheck("server_id"),
	}
}

func resourceScalewayFlexibleIPCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	fipAPI, zone, err := fipAPIWithZone(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	flexibleIP, err := fipAPI.CreateFlexibleIP(&flexibleip.CreateFlexibleIPRequest{
		Zone:        zone,
		ProjectID:   d.Get("project_id").(string),
		Description: d.Get("description").(string),
		Tags:        expandStrings(d.Get("tags")),
		ServerID:    types.ExpandStringPtr(locality.ExpandID(d.Get("server_id"))),
		Reverse:     types.ExpandStringPtr(d.Get("reverse")),
		IsIPv6:      d.Get("is_ipv6").(bool),
	}, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(zonal.NewZonedIDString(zone, flexibleIP.ID))

	_, err = waitFlexibleIP(ctx, fipAPI, zone, flexibleIP.ID, d.Timeout(schema.TimeoutCreate))
	if err != nil {
		return diag.FromErr(err)
	}
	return resourceScalewayFlexibleIPRead(ctx, d, meta)
}

func resourceScalewayFlexibleIPRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	fipAPI, zone, ID, err := fipAPIWithZoneAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	// verify resource is ready
	_, err = waitFlexibleIP(ctx, fipAPI, zone, ID, d.Timeout(schema.TimeoutRead))
	if err != nil {
		return diag.FromErr(err)
	}

	flexibleIP, err := fipAPI.GetFlexibleIP(&flexibleip.GetFlexibleIPRequest{
		Zone:  zone,
		FipID: ID,
	}, scw.WithContext(ctx))
	if err != nil {
		// We check for 403 because flexible API returns 403 for a deleted IP
		if http_errors.Is404Error(err) || http_errors.Is403Error(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	_ = d.Set("ip_address", flexibleIP.IPAddress.String())
	_ = d.Set("zone", flexibleIP.Zone)
	_ = d.Set("organization_id", flexibleIP.OrganizationID)
	_ = d.Set("project_id", flexibleIP.ProjectID)
	_ = d.Set("reverse", flexibleIP.Reverse)
	_ = d.Set("created_at", flattenTime(flexibleIP.CreatedAt))
	_ = d.Set("updated_at", flattenTime(flexibleIP.UpdatedAt))
	_ = d.Set("tags", flexibleIP.Tags)
	_ = d.Set("status", flexibleIP.Status.String())

	if flexibleIP.ServerID != nil {
		_ = d.Set("server_id", zonal.NewZonedIDString(zone, *flexibleIP.ServerID))
	} else {
		_ = d.Set("server_id", "")
	}

	return nil
}

func resourceScalewayFlexibleIPUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	fipAPI, zone, ID, err := fipAPIWithZoneAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	flexibleIP, err := waitFlexibleIP(ctx, fipAPI, zone, ID, d.Timeout(schema.TimeoutUpdate))
	if err != nil {
		return diag.FromErr(err)
	}
	updateRequest := &flexibleip.UpdateFlexibleIPRequest{
		Zone:  zone,
		FipID: flexibleIP.ID,
	}

	hasChanged := false

	if d.HasChanges("reverse") {
		updateRequest.Reverse = expandUpdatedStringPtr(d.Get("reverse"))
		hasChanged = true
	}

	if d.HasChange("tags") {
		updateRequest.Tags = expandUpdatedStringsPtr(d.Get("tags"))
		hasChanged = true
	}

	if d.HasChange("description") {
		updateRequest.Description = expandUpdatedStringPtr(d.Get("description"))
		hasChanged = true
	}

	if hasChanged {
		_, err = fipAPI.UpdateFlexibleIP(updateRequest, scw.WithContext(ctx))
		if err != nil {
			return diag.FromErr(err)
		}

		_, err = waitFlexibleIP(ctx, fipAPI, zone, ID, d.Timeout(schema.TimeoutUpdate))
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if d.HasChange("server_id") {
		if _, serverIDExists := d.GetOk("server_id"); !serverIDExists {
			_, err = fipAPI.DetachFlexibleIP(&flexibleip.DetachFlexibleIPRequest{
				Zone:    zone,
				FipsIDs: []string{ID},
			})
			if err != nil {
				return diag.FromErr(err)
			}
		} else {
			_, err = fipAPI.AttachFlexibleIP(&flexibleip.AttachFlexibleIPRequest{
				Zone:     zone,
				FipsIDs:  []string{ID},
				ServerID: locality.ExpandID(d.Get("server_id")),
			})
			if err != nil {
				return diag.FromErr(err)
			}
		}
	}

	_, err = waitFlexibleIP(ctx, fipAPI, zone, ID, d.Timeout(schema.TimeoutUpdate))
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceScalewayFlexibleIPRead(ctx, d, meta)
}

func resourceScalewayFlexibleIPDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	fipAPI, zone, ID, err := fipAPIWithZoneAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	flexibleIP, err := waitFlexibleIP(ctx, fipAPI, zone, ID, d.Timeout(schema.TimeoutDelete))
	if err != nil {
		return diag.FromErr(err)
	}

	err = fipAPI.DeleteFlexibleIP(&flexibleip.DeleteFlexibleIPRequest{
		FipID: flexibleIP.ID,
		Zone:  zone,
	}, scw.WithContext(ctx))

	if err != nil && !http_errors.Is404Error(err) && !http_errors.Is403Error(err) {
		return diag.FromErr(err)
	}

	_, err = waitFlexibleIP(ctx, fipAPI, zone, ID, d.Timeout(schema.TimeoutDelete))
	if err != nil && !http_errors.Is404Error(err) && !http_errors.Is403Error(err) {
		return diag.FromErr(err)
	}

	return nil
}
