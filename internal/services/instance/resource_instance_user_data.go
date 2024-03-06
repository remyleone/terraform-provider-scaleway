package instance

import (
	"bytes"
	"context"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/difffuncs"
	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/internal/errs"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
	"io"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/scaleway/scaleway-sdk-go/api/instance/v1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func ResourceScalewayInstanceUserData() *schema.Resource {
	return &schema.Resource{
		CreateContext: ResourceScalewayInstanceUserDataCreate,
		ReadContext:   ResourceScalewayInstanceUserDataRead,
		UpdateContext: ResourceScalewayInstanceUserDataUpdate,
		DeleteContext: ResourceScalewayInstanceUserDataDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Timeouts: &schema.ResourceTimeout{
			Create:  schema.DefaultTimeout(defaultInstanceServerWaitTimeout),
			Read:    schema.DefaultTimeout(defaultInstanceServerWaitTimeout),
			Update:  schema.DefaultTimeout(defaultInstanceServerWaitTimeout),
			Delete:  schema.DefaultTimeout(defaultInstanceServerWaitTimeout),
			Default: schema.DefaultTimeout(defaultInstanceServerWaitTimeout),
		},
		SchemaVersion: 0,
		Schema: map[string]*schema.Schema{
			"server_id": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "The ID of the server",
				ValidateFunc: locality.UUIDWithLocality(),
			},
			"key": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The key of the user data to set.",
			},
			"value": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The value of the user data to set.",
			},
			"zone": locality.ZonalSchema(),
		},
		CustomizeDiff: difffuncs.CustomizeDiffLocalityCheck("server_id"),
	}
}

func ResourceScalewayInstanceUserDataCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	instanceAPI, zone, err := instanceAPIWithZone(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	serverID := locality.ExpandID(d.Get("server_id").(string))
	server, err := waitForInstanceServer(ctx, instanceAPI, zone, serverID, d.Timeout(schema.TimeoutCreate))
	if err != nil {
		return diag.FromErr(err)
	}

	key := d.Get("key").(string)
	value := bytes.NewBufferString(d.Get("value").(string))

	userDataRequest := &instance.SetServerUserDataRequest{
		Zone:     zone,
		ServerID: server.ID,
		Key:      key,
		Content:  value,
	}

	if v, ok := d.GetOk("zone"); ok {
		userDataRequest.Zone = scw.Zone(v.(string))
	}

	err = instanceAPI.SetServerUserData(userDataRequest, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(locality.NewZonedNestedIDString(zone, key, server.ID))

	return ResourceScalewayInstanceUserDataRead(ctx, d, meta)
}

func ResourceScalewayInstanceUserDataRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	instanceAPI, zone, id, key, err := instanceAPIWithZoneAndNestedID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	server, err := waitForInstanceServer(ctx, instanceAPI, zone, id, d.Timeout(schema.TimeoutRead))
	if err != nil {
		return diag.FromErr(err)
	}

	requestGetUserData := &instance.GetServerUserDataRequest{
		Zone:     zone,
		ServerID: server.ID,
		Key:      key,
	}

	if v, ok := d.GetOk("zone"); ok {
		requestGetUserData.Zone = scw.Zone(v.(string))
		zone = requestGetUserData.Zone
	}

	serverUserDataRawValue, err := instanceAPI.GetServerUserData(requestGetUserData, scw.WithContext(ctx))
	if err != nil {
		if http_errors.Is404Error(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	userDataValue, err := io.ReadAll(serverUserDataRawValue)
	if err != nil {
		return diag.FromErr(err)
	}
	_ = d.Set("server_id", locality.NewZonedID(zone, server.ID).String())
	_ = d.Set("key", key)
	_ = d.Set("value", string(userDataValue))
	_ = d.Set("zone", zone.String())

	return nil
}

func ResourceScalewayInstanceUserDataUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	instanceAPI, zone, id, key, err := instanceAPIWithZoneAndNestedID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	server, err := waitForInstanceServer(ctx, instanceAPI, zone, id, d.Timeout(schema.TimeoutUpdate))
	if err != nil {
		return diag.FromErr(err)
	}

	userDataRequest := &instance.SetServerUserDataRequest{
		Zone:     zone,
		ServerID: server.ID,
		Key:      key,
	}

	if v, ok := d.GetOk("zone"); ok {
		userDataRequest.Zone = scw.Zone(v.(string))
	}

	if d.HasChanges("value") {
		value := d.Get("value")
		userDataRequest.Content = bytes.NewBufferString(value.(string))
	}

	err = instanceAPI.SetServerUserData(userDataRequest, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = waitForInstanceServer(ctx, instanceAPI, zone, id, d.Timeout(schema.TimeoutUpdate))
	if err != nil {
		return diag.FromErr(err)
	}

	return ResourceScalewayInstanceUserDataRead(ctx, d, meta)
}

func ResourceScalewayInstanceUserDataDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	instanceAPI, zone, id, key, err := instanceAPIWithZoneAndNestedID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	deleteUserData := &instance.DeleteServerUserDataRequest{
		ServerID: locality.ExpandID(id),
		Key:      key,
		Zone:     zone,
	}

	if v, ok := d.GetOk("zone"); ok {
		deleteUserData.Zone = scw.Zone(v.(string))
	}

	err = instanceAPI.DeleteServerUserData(deleteUserData, scw.WithContext(ctx))

	if err != nil && !http_errors.Is404Error(err) {
		return diag.FromErr(err)
	}

	return nil
}
