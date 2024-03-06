package lb

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	lbSDK "github.com/scaleway/scaleway-sdk-go/api/lb/v1"
	"github.com/scaleway/scaleway-sdk-go/scw"
	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/internal/errs"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/types"
)

func ResourceScalewayLbRoute() *schema.Resource {
	return &schema.Resource{
		CreateContext: ResourceScalewayLbRouteCreate,
		ReadContext:   ResourceScalewayLbRouteRead,
		UpdateContext: ResourceScalewayLbRouteUpdate,
		DeleteContext: ResourceScalewayLbRouteDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Timeouts: &schema.ResourceTimeout{
			Default: schema.DefaultTimeout(defaultLbLbTimeout),
		},
		SchemaVersion: 1,
		StateUpgraders: []schema.StateUpgrader{
			{Version: 0, Type: lbUpgradeV1SchemaType(), Upgrade: lbUpgradeV1SchemaUpgradeFunc},
		},
		Schema: map[string]*schema.Schema{
			"frontend_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: locality.UUIDorUUIDWithLocality(),
				Description:  "The frontend ID origin of redirection",
			},
			"backend_id": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: locality.UUIDorUUIDWithLocality(),
				Description:  "The backend ID destination of redirection",
			},
			"match_sni": {
				Type:          schema.TypeString,
				Optional:      true,
				Description:   "Server Name Indication TLS extension field from an incoming connection made via an SSL/TLS transport layer",
				ConflictsWith: []string{"match_host_header"},
			},
			"match_host_header": {
				Type:          schema.TypeString,
				Optional:      true,
				Description:   "Specifies the host of the server to which the request is being sent",
				ConflictsWith: []string{"match_sni"},
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The date at which the route was created (RFC 3339 format)",
			},
			"updated_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The date at which the route was last updated (RFC 3339 format)",
			},
		},
	}
}

func ResourceScalewayLbRouteCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	lbAPI, _, err := lbAPIWithZone(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	frontZone, frontID, err := locality.ParseZonedID(d.Get("frontend_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	backZone, backID, err := locality.ParseZonedID(d.Get("backend_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	if frontZone != backZone {
		return diag.Errorf("Frontend and Backend must be in the same zone (got %s and %s)", frontZone, backZone)
	}

	createReq := &lbSDK.ZonedAPICreateRouteRequest{
		Zone:       frontZone,
		FrontendID: frontID,
		BackendID:  backID,
		Match: &lbSDK.RouteMatch{
			Sni:        types.ExpandStringPtr(d.Get("match_sni")),
			HostHeader: types.ExpandStringPtr(d.Get("match_host_header")),
		},
	}

	route, err := lbAPI.CreateRoute(createReq, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(locality.NewZonedIDString(frontZone, route.ID))

	return ResourceScalewayLbRouteRead(ctx, d, meta)
}

func ResourceScalewayLbRouteRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	lbAPI, zone, ID, err := lbAPIWithZoneAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	route, err := lbAPI.GetRoute(&lbSDK.ZonedAPIGetRouteRequest{
		Zone:    zone,
		RouteID: ID,
	}, scw.WithContext(ctx))
	if err != nil {
		if http_errors.Is404Error(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	_ = d.Set("frontend_id", locality.NewZonedIDString(zone, route.FrontendID))
	_ = d.Set("backend_id", locality.NewZonedIDString(zone, route.BackendID))
	_ = d.Set("match_sni", types.FlattenStringPtr(route.Match.Sni))
	_ = d.Set("match_host_header", types.FlattenStringPtr(route.Match.HostHeader))
	_ = d.Set("created_at", types.FlattenTime(route.CreatedAt))
	_ = d.Set("updated_at", types.FlattenTime(route.UpdatedAt))

	return nil
}

func ResourceScalewayLbRouteUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	lbAPI, zone, ID, err := lbAPIWithZoneAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	backZone, backID, err := locality.ParseZonedID(d.Get("backend_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	if zone != backZone {
		return diag.Errorf("Route and Backend must be in the same zone (got %s and %s)", zone, backZone)
	}

	req := &lbSDK.ZonedAPIUpdateRouteRequest{
		Zone:      zone,
		RouteID:   ID,
		BackendID: backID,
		Match: &lbSDK.RouteMatch{
			Sni:        types.ExpandStringPtr(d.Get("match_sni")),
			HostHeader: types.ExpandStringPtr(d.Get("match_host_header")),
		},
	}

	_, err = lbAPI.UpdateRoute(req, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	return ResourceScalewayLbRouteRead(ctx, d, meta)
}

func ResourceScalewayLbRouteDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	lbAPI, zone, ID, err := lbAPIWithZoneAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	err = lbAPI.DeleteRoute(&lbSDK.ZonedAPIDeleteRouteRequest{
		Zone:    zone,
		RouteID: ID,
	}, scw.WithContext(ctx))

	if err != nil && !http_errors.Is404Error(err) {
		return diag.FromErr(err)
	}

	return nil
}
