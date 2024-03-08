package vpcgw

import (
	"context"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/difffuncs"
	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/internal/errs"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/transport"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/types"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	vpcgw "github.com/scaleway/scaleway-sdk-go/api/vpcgw/v1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func ResourceScalewayVPCGatewayNetwork() *schema.Resource {
	return &schema.Resource{
		CreateContext: ResourceScalewayVPCGatewayNetworkCreate,
		ReadContext:   ResourceScalewayVPCGatewayNetworkRead,
		UpdateContext: ResourceScalewayVPCGatewayNetworkUpdate,
		DeleteContext: ResourceScalewayVPCGatewayNetworkDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Timeouts: &schema.ResourceTimeout{
			Create:  schema.DefaultTimeout(defaultVPCGatewayTimeout),
			Read:    schema.DefaultTimeout(defaultVPCGatewayTimeout),
			Update:  schema.DefaultTimeout(defaultVPCGatewayTimeout),
			Delete:  schema.DefaultTimeout(defaultVPCGatewayTimeout),
			Default: schema.DefaultTimeout(defaultVPCGatewayTimeout),
		},
		SchemaVersion: 0,
		Schema: map[string]*schema.Schema{
			"gateway_id": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: locality.UUIDorUUIDWithLocality(),
				Description:  "The ID of the public gateway where connect to",
			},
			"private_network_id": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateFunc:     locality.UUIDorUUIDWithLocality(),
				DiffSuppressFunc: difffuncs.DiffSuppressFuncLocality,
				Description:      "The ID of the private network where connect to",
			},
			"dhcp_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ValidateFunc:  locality.UUIDorUUIDWithLocality(),
				Description:   "The ID of the public gateway DHCP config",
				ConflictsWith: []string{"static_address", "ipam_config"},
			},
			"enable_masquerade": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Enable masquerade on this network",
			},
			"enable_dhcp": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Enable DHCP config on this network",
			},
			"cleanup_dhcp": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Remove DHCP config on this network on destroy",
			},
			"static_address": {
				Type:          schema.TypeString,
				Description:   "The static IP address in CIDR on this network",
				Optional:      true,
				Computed:      true,
				ValidateFunc:  validation.IsCIDR,
				ConflictsWith: []string{"dhcp_id", "ipam_config"},
			},
			"ipam_config": {
				Type:          schema.TypeList,
				Optional:      true,
				Computed:      true,
				Description:   "Auto-configure the Gateway Network using Scaleway's IPAM (IP address management service)",
				ConflictsWith: []string{"dhcp_id", "static_address"},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"push_default_route": {
							Type:        schema.TypeBool,
							Optional:    true,
							Description: "Defines whether the default route is enabled on that Gateway Network",
						},
						"ipam_ip_id": {
							Type:             schema.TypeString,
							Optional:         true,
							Computed:         true,
							Description:      "Use this IPAM-booked IP ID as the Gateway's IP in this Private Network",
							ValidateFunc:     locality.UUIDorUUIDWithLocality(),
							DiffSuppressFunc: difffuncs.DiffSuppressFuncLocality,
						},
					},
				},
			},
			// Computed elements
			"mac_address": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The mac address on this network",
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The date and time of the creation of the gateway network",
			},
			"updated_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The date and time of the last update of the gateway network",
			},
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The status of the Public Gateway's connection to the Private Network",
			},
			"zone": locality.ZonalSchema(),
		},
		CustomizeDiff: difffuncs.CustomizeDiffLocalityCheck("gateway_id", "private_network_id", "dhcp_id"),
	}
}

func ResourceScalewayVPCGatewayNetworkCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	vpcgwAPI, zone, err := VpcgwAPIWithZone(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	gatewayID := locality.ExpandZonedID(d.Get("gateway_id").(string)).ID

	gateway, err := waitForVPCPublicGateway(ctx, vpcgwAPI, zone, gatewayID, d.Timeout(schema.TimeoutCreate))
	if err != nil {
		return diag.FromErr(err)
	}

	req := &vpcgw.CreateGatewayNetworkRequest{
		Zone:             zone,
		GatewayID:        gateway.ID,
		PrivateNetworkID: locality.ExpandRegionalID(d.Get("private_network_id").(string)).ID,
		EnableMasquerade: *types.ExpandBoolPtr(d.Get("enable_masquerade")),
		EnableDHCP:       types.ExpandBoolPtr(d.Get("enable_dhcp")),
		IpamConfig:       expandIpamConfig(d.Get("ipam_config")),
	}
	staticAddress, staticAddressExist := d.GetOk("static_address")
	if staticAddressExist {
		address, err := types.ExpandIPNet(staticAddress.(string))
		if err != nil {
			return diag.FromErr(err)
		}
		req.Address = &address
	}

	dhcpID, dhcpExist := d.GetOk("dhcp_id")
	if dhcpExist {
		dhcpZoned := locality.ExpandZonedID(dhcpID.(string))
		req.DHCPID = &dhcpZoned.ID
	}

	gatewayNetwork, err := transport.RetryOnTransientStateError(func() (*vpcgw.GatewayNetwork, error) {
		return vpcgwAPI.CreateGatewayNetwork(req, scw.WithContext(ctx))
	}, func() (*vpcgw.Gateway, error) {
		tflog.Warn(ctx, "Public gateway is in transient state after waiting, retrying...")
		return waitForVPCPublicGateway(ctx, vpcgwAPI, zone, gatewayID, d.Timeout(schema.TimeoutCreate))
	})
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(locality.NewZonedIDString(zone, gatewayNetwork.ID))

	_, err = waitForVPCPublicGateway(ctx, vpcgwAPI, zone, gatewayNetwork.GatewayID, d.Timeout(schema.TimeoutCreate))
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = waitForVPCGatewayNetwork(ctx, vpcgwAPI, zone, gatewayNetwork.ID, d.Timeout(schema.TimeoutCreate))
	if err != nil {
		return diag.FromErr(err)
	}

	return ResourceScalewayVPCGatewayNetworkRead(ctx, d, meta)
}

func ResourceScalewayVPCGatewayNetworkRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	vpcgwAPI, zone, ID, err := VpcgwAPIWithZoneAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	gatewayNetwork, err := waitForVPCGatewayNetwork(ctx, vpcgwAPI, zone, ID, d.Timeout(schema.TimeoutRead))
	if err != nil {
		if http_errors.Is404Error(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}
	_, err = waitForVPCPublicGateway(ctx, vpcgwAPI, zone, gatewayNetwork.GatewayID, d.Timeout(schema.TimeoutRead))
	if err != nil {
		return diag.FromErr(err)
	}

	if dhcp := gatewayNetwork.DHCP; dhcp != nil {
		_ = d.Set("dhcp_id", locality.NewZonedID(zone, dhcp.ID).String())
	}

	if staticAddress := gatewayNetwork.Address; staticAddress != nil {
		staticAddressValue, err := types.FlattenIPNet(*staticAddress)
		if err != nil {
			return diag.FromErr(err)
		}
		_ = d.Set("static_address", staticAddressValue)
	}

	if macAddress := gatewayNetwork.MacAddress; macAddress != nil {
		_ = d.Set("mac_address", types.FlattenStringPtr(macAddress).(string))
	}

	if enableDHCP := gatewayNetwork.EnableDHCP; enableDHCP {
		_ = d.Set("enable_dhcp", enableDHCP)
	}

	if ipamConfig := gatewayNetwork.IpamConfig; ipamConfig != nil {
		_ = d.Set("ipam_config", flattenIpamConfig(ipamConfig))
	}

	var cleanUpDHCPValue bool
	cleanUpDHCP, cleanUpDHCPExist := d.GetOk("cleanup_dhcp")
	if cleanUpDHCPExist {
		cleanUpDHCPValue = *types.ExpandBoolPtr(cleanUpDHCP)
	}

	gatewayNetwork, err = waitForVPCGatewayNetwork(ctx, vpcgwAPI, zone, gatewayNetwork.ID, d.Timeout(schema.TimeoutRead))
	if err != nil {
		return diag.FromErr(err)
	}

	fetchRegion, err := zone.Region()
	if err != nil {
		return diag.FromErr(err)
	}

	_ = d.Set("gateway_id", locality.NewZonedID(zone, gatewayNetwork.GatewayID).String())
	_ = d.Set("private_network_id", locality.NewRegionalIDString(fetchRegion, gatewayNetwork.PrivateNetworkID))
	_ = d.Set("enable_masquerade", gatewayNetwork.EnableMasquerade)
	_ = d.Set("cleanup_dhcp", cleanUpDHCPValue)
	_ = d.Set("created_at", gatewayNetwork.CreatedAt.Format(time.RFC3339))
	_ = d.Set("updated_at", gatewayNetwork.UpdatedAt.Format(time.RFC3339))
	_ = d.Set("zone", zone.String())
	_ = d.Set("status", gatewayNetwork.Status.String())

	return nil
}

func ResourceScalewayVPCGatewayNetworkUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	vpcgwAPI, zone, ID, err := VpcgwAPIWithZoneAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = waitForVPCGatewayNetwork(ctx, vpcgwAPI, zone, ID, d.Timeout(schema.TimeoutUpdate))
	if err != nil {
		return diag.FromErr(err)
	}

	updateRequest := &vpcgw.UpdateGatewayNetworkRequest{
		GatewayNetworkID: ID,
		Zone:             zone,
	}

	if d.HasChange("enable_masquerade") {
		updateRequest.EnableMasquerade = types.ExpandBoolPtr(d.Get("enable_masquerade"))
	}
	if d.HasChange("enable_dhcp") {
		updateRequest.EnableDHCP = types.ExpandBoolPtr(d.Get("enable_dhcp"))
	}
	if d.HasChange("dhcp_id") {
		dhcpID := locality.ExpandZonedID(d.Get("dhcp_id").(string)).ID
		updateRequest.DHCPID = &dhcpID
	}
	if d.HasChange("ipam_config") {
		updateRequest.IpamConfig = expandUpdateIpamConfig(d.Get("ipam_config"))
	}
	if d.HasChange("static_address") {
		staticAddress, staticAddressExist := d.GetOk("static_address")
		if staticAddressExist {
			address, err := types.ExpandIPNet(staticAddress.(string))
			if err != nil {
				return diag.FromErr(err)
			}
			updateRequest.Address = &address
		}
	}

	_, err = vpcgwAPI.UpdateGatewayNetwork(updateRequest, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = waitForVPCGatewayNetwork(ctx, vpcgwAPI, zone, ID, d.Timeout(schema.TimeoutUpdate))
	if err != nil {
		return diag.FromErr(err)
	}

	return ResourceScalewayVPCGatewayNetworkRead(ctx, d, meta)
}

func ResourceScalewayVPCGatewayNetworkDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	vpcgwAPI, zone, id, err := VpcgwAPIWithZoneAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	gwNetwork, err := waitForVPCGatewayNetwork(ctx, vpcgwAPI, zone, id, d.Timeout(schema.TimeoutDelete))
	if err != nil {
		return diag.FromErr(err)
	}

	req := &vpcgw.DeleteGatewayNetworkRequest{
		GatewayNetworkID: gwNetwork.ID,
		Zone:             gwNetwork.Zone,
		CleanupDHCP:      *types.ExpandBoolPtr(d.Get("cleanup_dhcp")),
	}
	err = vpcgwAPI.DeleteGatewayNetwork(req, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = waitForVPCGatewayNetwork(ctx, vpcgwAPI, zone, id, d.Timeout(schema.TimeoutDelete))
	if err != nil && !http_errors.Is404Error(err) {
		return diag.FromErr(err)
	}

	_, err = waitForVPCPublicGateway(ctx, vpcgwAPI, zone, gwNetwork.GatewayID, d.Timeout(schema.TimeoutDelete))
	if err != nil && !http_errors.Is404Error(err) {
		return diag.FromErr(err)
	}

	return nil
}
