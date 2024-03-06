package lb

import (
	"context"
	"fmt"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/difffuncs"
	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/internal/errs"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/verify"
	"strings"
	"time"

	"github.com/scaleway/terraform-provider-scaleway/v2/internal/project"

	"github.com/scaleway/terraform-provider-scaleway/v2/internal/organization"

	"github.com/scaleway/terraform-provider-scaleway/v2/internal/types"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	lbSDK "github.com/scaleway/scaleway-sdk-go/api/lb/v1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

const (
	defaultWaitLBRetryInterval = 30 * time.Second
)

func ResourceScalewayLb() *schema.Resource {
	return &schema.Resource{
		CreateContext: ResourceScalewayLbCreate,
		ReadContext:   ResourceScalewayLbRead,
		UpdateContext: ResourceScalewayLbUpdate,
		DeleteContext: ResourceScalewayLbDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Timeouts: &schema.ResourceTimeout{
			Create:  schema.DefaultTimeout(defaultLbLbTimeout),
			Read:    schema.DefaultTimeout(defaultLbLbTimeout),
			Update:  schema.DefaultTimeout(defaultLbLbTimeout),
			Delete:  schema.DefaultTimeout(defaultLbLbTimeout),
			Default: schema.DefaultTimeout(defaultLbLbTimeout),
		},
		SchemaVersion: 1,
		StateUpgraders: []schema.StateUpgrader{
			{Version: 0, Type: lbUpgradeV1SchemaType(), Upgrade: lbUpgradeV1SchemaUpgradeFunc},
		},
		CustomizeDiff: difffuncs.CustomizeDiffLocalityCheck("ip_id", "private_network.#.private_network_id"),
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Name of the lb",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The description of the lb",
			},
			"type": {
				Type:             schema.TypeString,
				Required:         true,
				DiffSuppressFunc: difffuncs.DiffSuppressFuncIgnoreCase,
				Description:      "The type of load-balancer you want to create",
			},
			"tags": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "Array of tags to associate with the load-balancer",
			},
			"ip_id": {
				Type:             schema.TypeString,
				Optional:         true,
				ForceNew:         true,
				Description:      "The load-balance public IP ID",
				DiffSuppressFunc: difffuncs.DiffSuppressFuncLocality,
				ValidateFunc:     locality.UUIDorUUIDWithLocality(),
			},
			"ip_address": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The load-balance public IP address",
			},
			"release_ip": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Release the IPs related to this load-balancer",
				Deprecated:  "The resource ip will be destroyed by it's own resource. Please set this to `false`",
			},
			"private_network": {
				Type:        schema.TypeSet,
				Optional:    true,
				MaxItems:    8,
				Set:         lbPrivateNetworkSetHash,
				Description: "List of private network to connect with your load balancer",
				DiffSuppressFunc: func(k, oldValue, newValue string, _ *schema.ResourceData) bool {
					// Check if the key is for the 'private_network_id' attribute
					if strings.HasSuffix(k, "private_network_id") {
						return locality.ExpandID(oldValue) == locality.ExpandID(newValue)
					}
					// For all other attributes, don't suppress the diff
					return false
				},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"private_network_id": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: locality.UUIDorUUIDWithLocality(),
							Description:  "The Private Network ID",
						},
						"static_config": {
							Description: "Define an IP address in the subnet of your private network that will be assigned to your load balancer instance",
							Type:        schema.TypeList,
							Optional:    true,
							Elem: &schema.Schema{
								Type:         schema.TypeString,
								ValidateFunc: verify.StandaloneIPorCIDR(),
							},
							MaxItems: 1,
						},
						"dhcp_config": {
							Description: "Set to true if you want to let DHCP assign IP addresses",
							Type:        schema.TypeBool,
							Optional:    true,
							Computed:    true,
						},
						// Readonly attributes
						"status": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The status of private network connection",
						},
						"zone": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"ssl_compatibility_level": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Enforces minimal SSL version (in SSL/TLS offloading context)",
				Default:     lbSDK.SSLCompatibilityLevelSslCompatibilityLevelIntermediate.String(),
				ValidateFunc: validation.StringInSlice([]string{
					lbSDK.SSLCompatibilityLevelSslCompatibilityLevelUnknown.String(),
					lbSDK.SSLCompatibilityLevelSslCompatibilityLevelIntermediate.String(),
					lbSDK.SSLCompatibilityLevelSslCompatibilityLevelModern.String(),
					lbSDK.SSLCompatibilityLevelSslCompatibilityLevelOld.String(),
				}, false),
			},
			"assign_flexible_ip": {
				Type:        schema.TypeBool,
				Optional:    true,
				ForceNew:    true,
				Description: "Defines whether to automatically assign a flexible public IP to the load balancer",
			},
			"region":          locality.RegionComputedSchema(),
			"zone":            locality.ZonalSchema(),
			"organization_id": organization.OrganizationIDSchema(),
			"project_id":      project.ProjectIDSchema(),
		},
	}
}

func ResourceScalewayLbCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	lbAPI, zone, err := lbAPIWithZone(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	createReq := &lbSDK.ZonedAPICreateLBRequest{
		Zone:                  zone,
		IPID:                  types.ExpandStringPtr(locality.ExpandID(d.Get("ip_id"))),
		ProjectID:             types.ExpandStringPtr(d.Get("project_id")),
		Name:                  types.ExpandOrGenerateString(d.Get("name"), "lb"),
		Description:           d.Get("description").(string),
		Type:                  d.Get("type").(string),
		SslCompatibilityLevel: lbSDK.SSLCompatibilityLevel(*types.ExpandStringPtr(d.Get("ssl_compatibility_level"))),
		AssignFlexibleIP:      types.ExpandBoolPtr(types.GetBool(d, "assign_flexible_ip")),
	}

	if tags, ok := d.GetOk("tags"); ok {
		for _, tag := range tags.([]interface{}) {
			createReq.Tags = append(createReq.Tags, tag.(string))
		}
	}

	lb, err := lbAPI.CreateLB(createReq, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(locality.NewZonedIDString(zone, lb.ID))

	// check err waiting process
	_, err = waitForLB(ctx, lbAPI, zone, lb.ID, d.Timeout(schema.TimeoutCreate))
	if err != nil {
		return diag.FromErr(err)
	}

	// attach private network
	pnConfigs, pnExist := d.GetOk("private_network")
	if pnExist {
		pnConfigs, err := expandPrivateNetworks(pnConfigs)
		if err != nil {
			return diag.FromErr(err)
		}

		_, err = attachLBPrivateNetworks(ctx, lbAPI, zone, pnConfigs, lb.ID, d.Timeout(schema.TimeoutCreate))
		if err != nil {
			return diag.FromErr(err)
		}

		_, err = waitForLB(ctx, lbAPI, zone, lb.ID, d.Timeout(schema.TimeoutCreate))
		if err != nil {
			return diag.FromErr(err)
		}
	}

	return ResourceScalewayLbRead(ctx, d, meta)
}

func ResourceScalewayLbRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	lbAPI, zone, ID, err := lbAPIWithZoneAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	lb, err := waitForLbInstances(ctx, lbAPI, zone, ID, d.Timeout(schema.TimeoutRead))
	if err != nil {
		if http_errors.Is404Error(err) || http_errors.Is403Error(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}
	// set the region from zone
	region, err := zone.Region()
	if err != nil {
		return diag.FromErr(err)
	}

	_ = d.Set("release_ip", false)
	_ = d.Set("name", lb.Name)
	_ = d.Set("description", lb.Description)
	_ = d.Set("zone", lb.Zone.String())
	_ = d.Set("region", region.String())
	_ = d.Set("organization_id", lb.OrganizationID)
	_ = d.Set("project_id", lb.ProjectID)
	_ = d.Set("tags", lb.Tags)
	// For now API return lowercase lb type. This should be fixed in a near future on the API side
	_ = d.Set("type", strings.ToUpper(lb.Type))
	_ = d.Set("ssl_compatibility_level", lb.SslCompatibilityLevel.String())
	if len(lb.IP) > 0 {
		_ = d.Set("ip_id", locality.NewZonedIDString(zone, lb.IP[0].ID))
		_ = d.Set("ip_address", lb.IP[0].IPAddress)
	}

	// retrieve attached private networks
	privateNetworks, err := waitForLBPN(ctx, lbAPI, zone, ID, d.Timeout(schema.TimeoutRead))
	if err != nil {
		if http_errors.Is404Error(err) {
			return nil
		}
		return diag.FromErr(err)
	}
	_ = d.Set("private_network", flattenPrivateNetworkConfigs(privateNetworks))
	return nil
}

//gocyclo:ignore
func ResourceScalewayLbUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	lbAPI, zone, ID, err := lbAPIWithZoneAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	req := &lbSDK.ZonedAPIUpdateLBRequest{
		Zone:                  zone,
		LBID:                  ID,
		Name:                  d.Get("name").(string),
		Tags:                  types.ExpandStrings(d.Get("tags")),
		Description:           d.Get("description").(string),
		SslCompatibilityLevel: lbSDK.SSLCompatibilityLevel(*types.ExpandStringPtr(d.Get("ssl_compatibility_level"))),
	}

	_, err = lbAPI.UpdateLB(req, scw.WithContext(ctx))
	if err != nil && !http_errors.Is404Error(err) {
		return diag.FromErr(err)
	}

	_, err = waitForLB(ctx, lbAPI, zone, ID, d.Timeout(schema.TimeoutUpdate))
	if err != nil && !http_errors.Is404Error(err) {
		return diag.FromErr(err)
	}

	if d.HasChange("type") {
		lbType := d.Get("type").(string)
		migrateReq := &lbSDK.ZonedAPIMigrateLBRequest{
			Zone: zone,
			LBID: ID,
			Type: lbType,
		}

		lb, err := lbAPI.MigrateLB(migrateReq, scw.WithContext(ctx))
		if err != nil {
			diag.FromErr(fmt.Errorf("couldn't migrate load balancer on type %s: %w", migrateReq.Type, err))
		}

		_, err = waitForLB(ctx, lbAPI, zone, lb.ID, d.Timeout(schema.TimeoutUpdate))
		if err != nil {
			return diag.FromErr(err)
		}
	}

	////
	// Attach / Detach Private Networks
	////
	if d.HasChange("private_network") {
		// check current lb stability state
		_, err = waitForLB(ctx, lbAPI, zone, ID, d.Timeout(schema.TimeoutUpdate))
		if err != nil {
			return diag.FromErr(err)
		}

		// check that pns are in a stable state
		pns, err := waitForLBPN(ctx, lbAPI, zone, ID, d.Timeout(schema.TimeoutUpdate))
		if err != nil && !http_errors.Is404Error(err) {
			return diag.FromErr(err)
		}

		pnConfigs, err := expandPrivateNetworks(d.Get("private_network"))
		if err != nil {
			return diag.FromErr(err)
		}
		// select only private networks that have changed
		pnToDetach := privateNetworksCompare(pnConfigs, pns)

		// detach private networks
		for i := range pnToDetach {
			err = lbAPI.DetachPrivateNetwork(&lbSDK.ZonedAPIDetachPrivateNetworkRequest{
				Zone:             zone,
				LBID:             ID,
				PrivateNetworkID: pnToDetach[i].PrivateNetworkID,
			}, scw.WithContext(ctx))
			if err != nil && !http_errors.Is404Error(err) {
				return diag.FromErr(err)
			}
		}

		// check load balancer state
		_, err = waitForLB(ctx, lbAPI, zone, ID, d.Timeout(schema.TimeoutUpdate))
		if err != nil && !http_errors.Is404Error(err) {
			return diag.FromErr(err)
		}

		// check that pns are in a stable state
		pns, err = waitForLBPN(ctx, lbAPI, zone, ID, d.Timeout(schema.TimeoutUpdate))
		if err != nil && !http_errors.Is404Error(err) {
			return diag.FromErr(err)
		}

		pnToAttach := privateNetworksCompare(pns, pnConfigs)
		// attach new/updated private networks
		_, err = attachLBPrivateNetworks(ctx, lbAPI, zone, pnToAttach, ID, d.Timeout(schema.TimeoutUpdate))
		if err != nil {
			return diag.FromErr(err)
		}

		privateNetworks, err := waitForLBPN(ctx, lbAPI, zone, ID, d.Timeout(schema.TimeoutUpdate))
		if err != nil && !http_errors.Is404Error(err) {
			return diag.FromErr(err)
		}

		for _, pn := range privateNetworks {
			tflog.Debug(ctx, fmt.Sprintf("PrivateNetwork ID %s state: %v", pn.PrivateNetworkID, pn.Status))
			if pn.Status == lbSDK.PrivateNetworkStatusError {
				err = lbAPI.DetachPrivateNetwork(&lbSDK.ZonedAPIDetachPrivateNetworkRequest{
					Zone:             zone,
					LBID:             ID,
					PrivateNetworkID: pn.PrivateNetworkID,
				}, scw.WithContext(ctx))
				if err != nil && !http_errors.Is404Error(err) {
					return diag.FromErr(err)
				}
				return diag.Errorf("attaching private network with id: %s on error state. please check your config", pn.PrivateNetworkID)
			}
		}
	}

	return ResourceScalewayLbRead(ctx, d, meta)
}

func ResourceScalewayLbDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	lbAPI, zone, ID, err := lbAPIWithZoneAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	// check if current lb is on stable state
	currentLB, err := waitForLbInstances(ctx, lbAPI, zone, ID, d.Timeout(schema.TimeoutDelete))
	if err != nil {
		return diag.FromErr(err)
	}

	if currentLB.PrivateNetworkCount != 0 {
		lbPNs, err := lbAPI.ListLBPrivateNetworks(&lbSDK.ZonedAPIListLBPrivateNetworksRequest{
			Zone: zone,
			LBID: ID,
		}, scw.WithContext(ctx))
		if err != nil && !http_errors.Is404Error(err) {
			return diag.FromErr(err)
		}

		// detach private networks
		for _, pn := range lbPNs.PrivateNetwork {
			err = lbAPI.DetachPrivateNetwork(&lbSDK.ZonedAPIDetachPrivateNetworkRequest{
				Zone:             zone,
				LBID:             ID,
				PrivateNetworkID: pn.PrivateNetworkID,
			}, scw.WithContext(ctx))
			if err != nil && !http_errors.Is404Error(err) {
				return diag.FromErr(err)
			}
		}

		_, err = waitForLbInstances(ctx, lbAPI, zone, ID, d.Timeout(schema.TimeoutDelete))
		if err != nil && !http_errors.Is404Error(err) {
			return diag.FromErr(err)
		}
	}

	err = lbAPI.DeleteLB(&lbSDK.ZonedAPIDeleteLBRequest{
		Zone:      zone,
		LBID:      ID,
		ReleaseIP: false,
	}, scw.WithContext(ctx))
	if err != nil && !http_errors.Is404Error(err) {
		return diag.FromErr(err)
	}

	_, err = waitForLB(ctx, lbAPI, zone, ID, d.Timeout(schema.TimeoutDelete))
	if err != nil && !http_errors.Is404Error(err) {
		return diag.FromErr(err)
	}

	_, err = waitForLbInstances(ctx, lbAPI, zone, ID, d.Timeout(schema.TimeoutDelete))
	if err != nil && !http_errors.Is404Error(err) {
		return diag.FromErr(err)
	}

	return nil
}
