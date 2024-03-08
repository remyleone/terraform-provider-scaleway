package applesilicon

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	applesilicon "github.com/scaleway/scaleway-sdk-go/api/applesilicon/v1alpha1"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/errs"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/organization"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/project"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/types"
	"time"
)

func ResourceScalewayAppleSiliconServer() *schema.Resource {
	return &schema.Resource{
		CreateContext: ResourceScalewayAppleSiliconServerCreate,
		ReadContext:   ResourceScalewayAppleSiliconServerRead,
		UpdateContext: ResourceScalewayAppleSiliconServerUpdate,
		DeleteContext: ResourceScalewayAppleSiliconServerDelete,
		Timeouts: &schema.ResourceTimeout{
			Create:  schema.DefaultTimeout(defaultAppleSiliconServerTimeout),
			Default: schema.DefaultTimeout(defaultAppleSiliconServerTimeout),
		},
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		SchemaVersion: 0,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Description: "Name of the server",
				Computed:    true,
				Optional:    true,
			},
			"type": {
				Type:        schema.TypeString,
				Description: "Type of the server",
				Required:    true,
				ForceNew:    true,
			},
			// Computed
			"ip": {
				Type:        schema.TypeString,
				Description: "IPv4 address of the server",
				Computed:    true,
			},
			"vnc_url": {
				Type:        schema.TypeString,
				Description: "VNC url use to connect remotely to the desktop GUI",
				Computed:    true,
			},
			"state": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The state of the server",
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The date and time of the creation of the server",
			},
			"updated_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The date and time of the last update of the server",
			},
			"deletable_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The minimal date and time on which you can delete this server due to Apple licence",
			},

			// Common
			"zone":            locality.ZonalSchema(),
			"organization_id": organization.OrganizationIDSchema(),
			"project_id":      project.ProjectIDSchema(),
		},
	}
}

func ResourceScalewayAppleSiliconServerCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	asAPI, zone, err := NewAPIWithZone(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	createReq := &applesilicon.CreateServerRequest{
		Name:      types.ExpandOrGenerateString(d.Get("name"), "m1"),
		Type:      d.Get("type").(string),
		ProjectID: d.Get("project_id").(string),
	}

	res, err := asAPI.CreateServer(createReq, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(locality.NewZonedIDString(zone, res.ID))

	_, err = waitForAppleSiliconServer(ctx, asAPI, zone, res.ID, d.Timeout(schema.TimeoutCreate))
	if err != nil {
		return diag.FromErr(err)
	}

	return ResourceScalewayAppleSiliconServerRead(ctx, d, meta)
}

func ResourceScalewayAppleSiliconServerRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	asAPI, zone, ID, err := NewAPIWithZoneAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	res, err := asAPI.GetServer(&applesilicon.GetServerRequest{
		Zone:     zone,
		ServerID: ID,
	}, scw.WithContext(ctx))
	if err != nil {
		if errs.Is404Error(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	_ = d.Set("name", res.Name)
	_ = d.Set("type", res.Type)
	_ = d.Set("state", res.Status.String())
	_ = d.Set("created_at", res.CreatedAt.Format(time.RFC3339))
	_ = d.Set("updated_at", res.UpdatedAt.Format(time.RFC3339))
	_ = d.Set("deletable_at", res.DeletableAt.Format(time.RFC3339))
	_ = d.Set("ip", res.IP.String())
	_ = d.Set("vnc_url", res.VncURL)

	_ = d.Set("zone", res.Zone.String())
	_ = d.Set("organization_id", res.OrganizationID)
	_ = d.Set("project_id", res.ProjectID)

	return nil
}

func ResourceScalewayAppleSiliconServerUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	asAPI, zone, ID, err := NewAPIWithZoneAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	req := &applesilicon.UpdateServerRequest{
		Zone:     zone,
		ServerID: ID,
	}

	if d.HasChange("name") {
		req.Name = types.ExpandStringPtr(d.Get("name"))
	}

	_, err = asAPI.UpdateServer(req, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	return ResourceScalewayAppleSiliconServerRead(ctx, d, meta)
}

func ResourceScalewayAppleSiliconServerDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	asAPI, zone, ID, err := NewAPIWithZoneAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	err = asAPI.DeleteServer(&applesilicon.DeleteServerRequest{
		Zone:     zone,
		ServerID: ID,
	}, scw.WithContext(ctx))

	if err != nil && !errs.Is404Error(err) {
		return diag.FromErr(err)
	}

	return nil
}
