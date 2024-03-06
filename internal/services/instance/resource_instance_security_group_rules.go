package instance

import (
	"context"
	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/internal/errs"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceScalewayInstanceSecurityGroupRules() *schema.Resource {
	return &schema.Resource{
		CreateContext: ResourceScalewayInstanceSecurityGroupRulesCreate,
		ReadContext:   ResourceScalewayInstanceSecurityGroupRulesRead,
		UpdateContext: ResourceScalewayInstanceSecurityGroupRulesUpdate,
		DeleteContext: ResourceScalewayInstanceSecurityGroupRulesDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Timeouts: &schema.ResourceTimeout{
			Default: schema.DefaultTimeout(defaultInstanceSecurityGroupRuleTimeout),
		},
		Schema: map[string]*schema.Schema{
			"security_group_id": {
				Type:     schema.TypeString,
				Required: true,
				// Ensure SecurityGroupRules.ID and SecurityGroupRules.security_group_id stay in sync.
				// If security_group_id is changed, a new SecurityGroupRules is created, with a new ID.
				ForceNew:    true,
				Description: "The security group associated with this volume",
			},
			"inbound_rule": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Inbound rules for this set of security group rules",
				Elem:        securityGroupRuleSchema(),
			},
			"outbound_rule": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Outbound rules for this set of security group rules",
				Elem:        securityGroupRuleSchema(),
			},
		},
	}
}

func ResourceScalewayInstanceSecurityGroupRulesCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	d.SetId(d.Get("security_group_id").(string))

	// We call update instead of read as it will take care of creating rules.
	return ResourceScalewayInstanceSecurityGroupRulesUpdate(ctx, d, meta)
}

func ResourceScalewayInstanceSecurityGroupRulesRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	securityGroupZonedID := d.Id()

	instanceAPI, zone, securityGroupID, err := instanceAPIWithZoneAndID(meta, securityGroupZonedID)
	if err != nil {
		return diag.FromErr(err)
	}

	_ = d.Set("security_group_id", securityGroupZonedID)

	inboundRules, outboundRules, err := getSecurityGroupRules(ctx, instanceAPI, zone, securityGroupID, d)
	if err != nil {
		if http_errors.Is404Error(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	_ = d.Set("inbound_rule", inboundRules)
	_ = d.Set("outbound_rule", outboundRules)

	return nil
}

func ResourceScalewayInstanceSecurityGroupRulesUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	securityGroupZonedID := d.Id()
	instanceAPI, zone, securityGroupID, err := instanceAPIWithZoneAndID(meta, securityGroupZonedID)
	if err != nil {
		return diag.FromErr(err)
	}

	err = updateSecurityGroupeRules(ctx, d, zone, securityGroupID, instanceAPI)
	if err != nil {
		return diag.FromErr(err)
	}

	return ResourceScalewayInstanceSecurityGroupRulesRead(ctx, d, meta)
}

func ResourceScalewayInstanceSecurityGroupRulesDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	securityGroupZonedID := d.Id()
	instanceAPI, zone, securityGroupID, err := instanceAPIWithZoneAndID(meta, securityGroupZonedID)
	if err != nil {
		return diag.FromErr(err)
	}

	_ = d.Set("inbound_rule", nil)
	_ = d.Set("outbound_rule", nil)

	err = updateSecurityGroupeRules(ctx, d, zone, securityGroupID, instanceAPI)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}
