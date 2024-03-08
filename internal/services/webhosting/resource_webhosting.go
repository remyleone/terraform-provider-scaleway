package webhosting

import (
	"context"
	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/internal/errs"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/types"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/verify"

	"github.com/scaleway/terraform-provider-scaleway/v2/internal/project"

	"github.com/scaleway/terraform-provider-scaleway/v2/internal/organization"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	webhosting "github.com/scaleway/scaleway-sdk-go/api/webhosting/v1alpha1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func ResourceScalewayWebhosting() *schema.Resource {
	return &schema.Resource{
		CreateContext: ResourceScalewayWebhostingCreate,
		ReadContext:   ResourceScalewayWebhostingRead,
		UpdateContext: ResourceScalewayWebhostingUpdate,
		DeleteContext: ResourceScalewayWebhostingDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Timeouts: &schema.ResourceTimeout{
			Create:  schema.DefaultTimeout(defaultWebhostingTimeout),
			Read:    schema.DefaultTimeout(defaultWebhostingTimeout),
			Delete:  schema.DefaultTimeout(defaultWebhostingTimeout),
			Default: schema.DefaultTimeout(defaultWebhostingTimeout),
		},
		SchemaVersion: 0,
		Schema: map[string]*schema.Schema{
			"offer_id": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: locality.UUIDorUUIDWithLocality(),
				Description:  "The ID of the selected offer for the hosting",
			},
			"email": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: verify.Email(),
				Description:  "Contact email of the client for the hosting",
			},
			"domain": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The domain name of the hosting",
			},
			"tags": {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional:    true,
				Computed:    true,
				Description: "The tags of the hosting",
			},
			"option_ids": {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional:    true,
				Description: "IDs of the selected options for the hosting",
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Date and time of hosting's creation (RFC 3339 format)",
			},
			"updated_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Date and time of hosting's last update (RFC 3339 format)",
			},
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The hosting status",
			},
			"platform_hostname": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Hostname of the host platform",
			},
			"platform_number": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Number of the host platform",
			},
			"offer_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Name of the active offer",
			},
			"options": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Active options of the hosting",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"dns_status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "DNS status of the hosting",
			},
			"cpanel_urls": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "URL to connect to cPanel Dashboard and to Webmail interface",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"dashboard": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"webmail": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"username": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Main hosting cPanel username",
			},
			"region":          locality.RegionalSchema(),
			"project_id":      project.ProjectIDSchema(),
			"organization_id": organization.OrganizationIDSchema(),
		},
		CustomizeDiff: func(_ context.Context, diff *schema.ResourceDiff, _ interface{}) error {
			if diff.HasChange("tags") {
				oldTagsInterface, newTagsInterface := diff.GetChange("tags")
				oldTags := types.ExpandStrings(oldTagsInterface)
				newTags := types.ExpandStrings(newTagsInterface)
				// If the 'internal' tag has been added, remove it from the diff
				if types.SliceContainsString(oldTags, "internal") && !types.SliceContainsString(newTags, "internal") {
					err := diff.SetNew("tags", oldTags)
					if err != nil {
						return err
					}
				}
			}
			return nil
		},
	}
}

func ResourceScalewayWebhostingCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, err := WebhostingAPIWithRegion(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	_, offerID, err := locality.ParseRegionalID(d.Get("offer_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	hostingCreateRequest := &webhosting.CreateHostingRequest{
		Region:    region,
		OfferID:   offerID,
		ProjectID: d.Get("project_id").(string),
		Domain:    d.Get("domain").(string),
		OptionIDs: types.ExpandStrings(d.Get("option_ids")),
	}

	rawTags, tagsExist := d.GetOk("tags")
	if tagsExist {
		hostingCreateRequest.Tags = types.ExpandStrings(rawTags)
	}

	rawOptionIDs, rawOptionIDsExist := d.GetOk("option_ids")
	if rawOptionIDsExist {
		hostingCreateRequest.OptionIDs = types.ExpandStrings(rawOptionIDs)
	}

	rawEmail, emailExist := d.GetOk("email")
	if emailExist {
		hostingCreateRequest.Email = types.ExpandStringPtr(rawEmail)
	}

	hostingResponse, err := api.CreateHosting(hostingCreateRequest, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(locality.NewRegionalIDString(region, hostingResponse.ID))

	_, err = waitForHosting(ctx, api, region, hostingResponse.ID, d.Timeout(schema.TimeoutCreate))
	if err != nil {
		return diag.FromErr(err)
	}

	return ResourceScalewayWebhostingRead(ctx, d, meta)
}

func ResourceScalewayWebhostingRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, id, err := WebhostingAPIWithRegionAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	webhostingResponse, err := waitForHosting(ctx, api, region, id, d.Timeout(schema.TimeoutRead))
	if err != nil {
		if http_errors.Is404Error(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	if len(webhostingResponse.Tags) > 0 {
		_ = d.Set("tags", webhostingResponse.Tags)
	}

	_ = d.Set("offer_id", locality.NewRegionalIDString(region, webhostingResponse.OfferID))
	_ = d.Set("domain", webhostingResponse.Domain)
	_ = d.Set("created_at", types.FlattenTime(webhostingResponse.CreatedAt))
	_ = d.Set("updated_at", types.FlattenTime(webhostingResponse.UpdatedAt))
	_ = d.Set("status", webhostingResponse.Status.String())
	_ = d.Set("platform_hostname", webhostingResponse.PlatformHostname)
	_ = d.Set("platform_number", types.FlattenInt32Ptr(webhostingResponse.PlatformNumber))
	_ = d.Set("options", flattenHostingOptions(webhostingResponse.Options))
	_ = d.Set("offer_name", webhostingResponse.OfferName)
	_ = d.Set("dns_status", webhostingResponse.DNSStatus.String())
	_ = d.Set("cpanel_urls", flattenHostingCpanelUrls(webhostingResponse.CpanelURLs))
	_ = d.Set("username", webhostingResponse.Username)
	_ = d.Set("region", string(region))
	_ = d.Set("organization_id", webhostingResponse.OrganizationID)
	_ = d.Set("project_id", webhostingResponse.ProjectID)

	return nil
}

func ResourceScalewayWebhostingUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, id, err := WebhostingAPIWithRegionAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	res, err := waitForHosting(ctx, api, region, id, d.Timeout(schema.TimeoutRead))
	if err != nil {
		return diag.FromErr(err)
	}

	updateRequest := &webhosting.UpdateHostingRequest{
		Region:    region,
		HostingID: res.ID,
	}

	hasChanged := false

	if d.HasChange("option_ids") {
		updateRequest.OptionIDs = types.ExpandUpdatedStringsPtr(d.Get("option_ids"))
		hasChanged = true
	}

	if d.HasChange("offer_id") {
		_, offerID, err := locality.ParseRegionalID(d.Get("offer_id").(string))
		if err != nil {
			return diag.FromErr(err)
		}
		updateRequest.OfferID = types.ExpandUpdatedStringPtr(offerID)
		hasChanged = true
	}

	if d.HasChange("email") {
		updateRequest.Email = types.ExpandUpdatedStringPtr(d.Get("email"))
		hasChanged = true
	}

	if d.HasChange("tags") {
		updateRequest.Tags = types.ExpandUpdatedStringsPtr(d.Get("tags"))
		hasChanged = true
	}

	if hasChanged {
		_, err := api.UpdateHosting(updateRequest, scw.WithContext(ctx))
		if err != nil {
			return diag.FromErr(err)
		}
	}

	return ResourceScalewayWebhostingRead(ctx, d, meta)
}

func ResourceScalewayWebhostingDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, id, err := WebhostingAPIWithRegionAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = waitForHosting(ctx, api, region, id, d.Timeout(schema.TimeoutDelete))
	if err != nil {
		return nil
	}

	_, err = api.DeleteHosting(&webhosting.DeleteHostingRequest{
		Region:    region,
		HostingID: id,
	}, scw.WithContext(ctx))
	if err != nil && !http_errors.Is404Error(err) {
		return diag.FromErr(err)
	}

	_, err = waitForHosting(ctx, api, region, id, d.Timeout(schema.TimeoutDelete))
	if err != nil {
		return nil
	}

	return nil
}
