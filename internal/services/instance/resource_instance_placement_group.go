package instance

import (
	"context"
	"github.com/scaleway/scaleway-sdk-go/api/instance/v1"
	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/internal/errs"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/project"

	"github.com/scaleway/terraform-provider-scaleway/v2/internal/organization"

	"github.com/scaleway/terraform-provider-scaleway/v2/internal/types"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func ResourceScalewayInstancePlacementGroup() *schema.Resource {
	return &schema.Resource{
		CreateContext: ResourceScalewayInstancePlacementGroupCreate,
		ReadContext:   ResourceScalewayInstancePlacementGroupRead,
		UpdateContext: ResourceScalewayInstancePlacementGroupUpdate,
		DeleteContext: ResourceScalewayInstancePlacementGroupDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Timeouts: &schema.ResourceTimeout{
			Default: schema.DefaultTimeout(defaultInstancePlacementGroupTimeout),
		},
		SchemaVersion: 0,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The name of the placement group",
			},
			"policy_type": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     instance.PlacementGroupPolicyTypeMaxAvailability.String(),
				Description: "The operating mode is selected by a policy_type",
				ValidateFunc: validation.StringInSlice([]string{
					instance.PlacementGroupPolicyTypeLowLatency.String(),
					instance.PlacementGroupPolicyTypeMaxAvailability.String(),
				}, false),
			},
			"policy_mode": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     instance.PlacementGroupPolicyModeOptional,
				Description: "One of the two policy_mode may be selected: enforced or optional.",
				ValidateFunc: validation.StringInSlice([]string{
					instance.PlacementGroupPolicyModeOptional.String(),
					instance.PlacementGroupPolicyModeEnforced.String(),
				}, false),
			},
			"policy_respected": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Is true when the policy is respected.",
			},
			"tags": {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional:    true,
				Description: "The tags associated with the placement group",
			},
			"zone":            locality.ZonalSchema(),
			"organization_id": organization.OrganizationIDSchema(),
			"project_id":      project.ProjectIDSchema(),
		},
	}
}

func ResourceScalewayInstancePlacementGroupCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	instanceAPI, zone, err := InstanceAPIWithZone(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	res, err := instanceAPI.CreatePlacementGroup(&instance.CreatePlacementGroupRequest{
		Zone:       zone,
		Name:       types.ExpandOrGenerateString(d.Get("name"), "pg"),
		Project:    types.ExpandStringPtr(d.Get("project_id")),
		PolicyMode: instance.PlacementGroupPolicyMode(d.Get("policy_mode").(string)),
		PolicyType: instance.PlacementGroupPolicyType(d.Get("policy_type").(string)),
		Tags:       types.ExpandStrings(d.Get("tags")),
	}, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(locality.NewZonedIDString(zone, res.PlacementGroup.ID))
	return ResourceScalewayInstancePlacementGroupRead(ctx, d, meta)
}

func ResourceScalewayInstancePlacementGroupRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	instanceAPI, zone, ID, err := InstanceAPIWithZoneAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	res, err := instanceAPI.GetPlacementGroup(&instance.GetPlacementGroupRequest{
		Zone:             zone,
		PlacementGroupID: ID,
	}, scw.WithContext(ctx))
	if err != nil {
		if http_errors.Is404Error(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	_ = d.Set("name", res.PlacementGroup.Name)
	_ = d.Set("zone", string(zone))
	_ = d.Set("organization_id", res.PlacementGroup.Organization)
	_ = d.Set("project_id", res.PlacementGroup.Project)
	_ = d.Set("policy_mode", res.PlacementGroup.PolicyMode.String())
	_ = d.Set("policy_type", res.PlacementGroup.PolicyType.String())
	_ = d.Set("policy_respected", res.PlacementGroup.PolicyRespected)
	_ = d.Set("tags", res.PlacementGroup.Tags)

	return nil
}

func ResourceScalewayInstancePlacementGroupUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	instanceAPI, zone, ID, err := InstanceAPIWithZoneAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	req := &instance.UpdatePlacementGroupRequest{
		Zone:             zone,
		PlacementGroupID: ID,
		Tags:             scw.StringsPtr([]string{}),
	}

	hasChanged := false

	if d.HasChange("name") {
		req.Name = types.ExpandStringPtr(d.Get("name").(string))
		hasChanged = true
	}

	if d.HasChange("policy_mode") {
		policyMode := instance.PlacementGroupPolicyMode(d.Get("policy_mode").(string))
		req.PolicyMode = &policyMode
		hasChanged = true
	}

	if d.HasChange("policy_type") {
		policyType := instance.PlacementGroupPolicyType(d.Get("policy_type").(string))
		req.PolicyType = &policyType
		hasChanged = true
	}

	if d.HasChange("tags") {
		req.Tags = types.ExpandUpdatedStringsPtr(d.Get("tags"))
		hasChanged = true
	}

	if hasChanged {
		_, err = instanceAPI.UpdatePlacementGroup(req, scw.WithContext(ctx))
		if err != nil {
			return diag.FromErr(err)
		}
	}

	return ResourceScalewayInstancePlacementGroupRead(ctx, d, meta)
}

func ResourceScalewayInstancePlacementGroupDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	instanceAPI, zone, ID, err := InstanceAPIWithZoneAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	err = instanceAPI.DeletePlacementGroup(&instance.DeletePlacementGroupRequest{
		Zone:             zone,
		PlacementGroupID: ID,
	}, scw.WithContext(ctx))

	if err != nil && !http_errors.Is404Error(err) {
		return diag.FromErr(err)
	}

	return nil
}
