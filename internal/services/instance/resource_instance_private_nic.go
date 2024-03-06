package instance

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/scaleway/scaleway-sdk-go/api/instance/v1"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/difffuncs"
	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/internal/errs"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/types"
)

func ResourceScalewayInstancePrivateNIC() *schema.Resource {
	return &schema.Resource{
		CreateContext: ResourceScalewayInstancePrivateNICCreate,
		ReadContext:   ResourceScalewayInstancePrivateNICRead,
		UpdateContext: ResourceScalewayInstancePrivateNICUpdate,
		DeleteContext: ResourceScalewayInstancePrivateNICDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Timeouts: &schema.ResourceTimeout{
			Create:  schema.DefaultTimeout(defaultInstancePrivateNICWaitTimeout),
			Read:    schema.DefaultTimeout(defaultInstancePrivateNICWaitTimeout),
			Update:  schema.DefaultTimeout(defaultInstancePrivateNICWaitTimeout),
			Delete:  schema.DefaultTimeout(defaultInstancePrivateNICWaitTimeout),
			Default: schema.DefaultTimeout(defaultInstancePrivateNICWaitTimeout),
		},
		Schema: map[string]*schema.Schema{
			"server_id": {
				Type:        schema.TypeString,
				Description: "The server ID",
				Required:    true,
				ForceNew:    true,
			},
			"private_network_id": {
				Type:             schema.TypeString,
				Description:      "The private network ID",
				Required:         true,
				ForceNew:         true,
				DiffSuppressFunc: difffuncs.DiffSuppressFuncLocality,
			},
			"mac_address": {
				Type:        schema.TypeString,
				Description: "MAC address of the NIC",
				Computed:    true,
			},
			"tags": {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional:    true,
				Description: "The tags associated with the private-nic",
			},
			"ip_ids": {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional:    true,
				Description: "IPAM ip list, should be for internal use only",
				ForceNew:    true,
			},
			"zone": locality.ZonalSchema(),
		},
		CustomizeDiff: difffuncs.CustomizeDiffLocalityCheck("server_id", "private_network_id"),
	}
}

func ResourceScalewayInstancePrivateNICCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	instanceAPI, zone, err := instanceAPIWithZone(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = waitForInstanceServer(ctx, instanceAPI, zone, locality.ExpandID(d.Get("server_id")), d.Timeout(schema.TimeoutCreate))
	if err != nil {
		return diag.FromErr(err)
	}

	createPrivateNICRequest := &instance.CreatePrivateNICRequest{
		Zone:             zone,
		ServerID:         locality.ExpandZonedID(d.Get("server_id").(string)).ID,
		PrivateNetworkID: locality.ExpandRegionalID(d.Get("private_network_id").(string)).ID,
		Tags:             types.ExpandStrings(d.Get("tags")),
		IPIDs:            types.ExpandStrings(d.Get("ip_ids")),
	}

	privateNIC, err := instanceAPI.CreatePrivateNIC(
		createPrivateNICRequest,
		scw.WithContext(ctx),
	)
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = waitForPrivateNIC(ctx, instanceAPI, zone, privateNIC.PrivateNic.ServerID, privateNIC.PrivateNic.ID, d.Timeout(schema.TimeoutCreate))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(
		locality.NewZonedNestedIDString(
			zone,
			privateNIC.PrivateNic.ServerID,
			privateNIC.PrivateNic.ID,
		),
	)

	return ResourceScalewayInstancePrivateNICRead(ctx, d, meta)
}

func ResourceScalewayInstancePrivateNICRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	instanceAPI, _, err := instanceAPIWithZone(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}
	zone, privateNICID, serverID, err := locality.ParseZonedNestedID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	privateNIC, err := waitForPrivateNIC(ctx, instanceAPI, zone, serverID, privateNICID, d.Timeout(schema.TimeoutRead))
	if err != nil {
		if http_errors.Is404Error(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	fetchRegion, err := zone.Region()
	if err != nil {
		return diag.FromErr(err)
	}

	_ = d.Set("zone", zone)
	_ = d.Set("server_id", locality.NewZonedID(zone, privateNIC.ServerID).String())
	_ = d.Set("private_network_id", locality.NewRegionalIDString(fetchRegion, privateNIC.PrivateNetworkID))
	_ = d.Set("mac_address", privateNIC.MacAddress)

	if len(privateNIC.Tags) > 0 {
		_ = d.Set("tags", privateNIC.Tags)
	}

	return nil
}

func ResourceScalewayInstancePrivateNICUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	instanceAPI, _, err := instanceAPIWithZone(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	zone, privateNICID, serverID, err := locality.ParseZonedNestedID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if d.HasChange("tags") {
		_, err := instanceAPI.UpdatePrivateNIC(
			&instance.UpdatePrivateNICRequest{
				Zone:         zone,
				ServerID:     serverID,
				PrivateNicID: privateNICID,
				Tags:         types.ExpandUpdatedStringsPtr(d.Get("tags")),
			},
			scw.WithContext(ctx),
		)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	return ResourceScalewayInstancePrivateNICRead(ctx, d, meta)
}

func ResourceScalewayInstancePrivateNICDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	instanceAPI, _, err := instanceAPIWithZone(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}
	zone, privateNICID, serverID, err := locality.ParseZonedNestedID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = waitForPrivateNIC(ctx, instanceAPI, zone, serverID, privateNICID, d.Timeout(schema.TimeoutDelete))
	if err != nil {
		if http_errors.Is404Error(err) {
			return nil
		}
		return diag.FromErr(err)
	}

	err = instanceAPI.DeletePrivateNIC(&instance.DeletePrivateNICRequest{
		ServerID:     serverID,
		PrivateNicID: privateNICID,
		Zone:         zone,
	}, scw.WithContext(ctx))
	if err != nil {
		if http_errors.Is404Error(err) {
			return nil
		}
		return diag.FromErr(err)
	}

	_, err = waitForPrivateNIC(ctx, instanceAPI, zone, serverID, privateNICID, d.Timeout(schema.TimeoutDelete))
	if err != nil {
		if http_errors.Is404Error(err) {
			return nil
		}
		return diag.FromErr(err)
	}

	return nil
}
