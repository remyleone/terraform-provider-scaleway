package scaleway

import (
	"context"
	"math"
	"sort"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	lbSDK "github.com/scaleway/scaleway-sdk-go/api/lb/v1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func resourceScalewayLbFrontend() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceScalewayLbFrontendCreate,
		ReadContext:   resourceScalewayLbFrontendRead,
		UpdateContext: resourceScalewayLbFrontendUpdate,
		DeleteContext: resourceScalewayLbFrontendDelete,
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
			"lb_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validationUUIDorUUIDWithLocality(),
				Description:  "The load-balancer ID",
			},
			"backend_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validationUUIDorUUIDWithLocality(),
				Description:  "The load-balancer backend ID",
			},
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The name of the frontend",
			},
			"inbound_port": {
				Type:         schema.TypeInt,
				Required:     true,
				ValidateFunc: validation.IntBetween(0, math.MaxUint16),
				Description:  "TCP port to listen on the front side",
			},
			"timeout_client": {
				Type:             schema.TypeString,
				Optional:         true,
				DiffSuppressFunc: diffSuppressFuncDuration,
				ValidateFunc:     validateDuration(),
				Description:      "Set the maximum inactivity time on the client side",
			},
			"certificate_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Certificate ID",
				Deprecated:  "Please use certificate_ids",
			},
			"certificate_ids": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validationUUIDorUUIDWithLocality(),
				},
				Description: "Collection of Certificate IDs related to the load balancer and domain",
			},
			"acl": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "ACL rules",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
							Description: "The ACL name",
						},
						"action": {
							Type:        schema.TypeList,
							Required:    true,
							Description: "Action to undertake when an ACL filter matches",
							MaxItems:    1,
							MinItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"type": {
										Type:     schema.TypeString,
										Required: true,
										ValidateFunc: validation.StringInSlice([]string{
											lbSDK.ACLActionTypeAllow.String(),
											lbSDK.ACLActionTypeDeny.String(),
										}, false),
										Description: "The action type",
									},
								},
							},
						},
						"match": {
							Type:        schema.TypeList,
							Required:    true,
							MaxItems:    1,
							MinItems:    1,
							Description: "The ACL match rule",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"ip_subnet": {
										Type: schema.TypeList,
										Elem: &schema.Schema{
											Type: schema.TypeString,
										},
										Optional:    true,
										Description: "A list of IPs or CIDR v4/v6 addresses of the client of the session to match",
									},
									"http_filter": {
										Type:     schema.TypeString,
										Optional: true,
										Default:  lbSDK.ACLHTTPFilterACLHTTPFilterNone.String(),
										ValidateFunc: validation.StringInSlice([]string{
											lbSDK.ACLHTTPFilterACLHTTPFilterNone.String(),
											lbSDK.ACLHTTPFilterPathBegin.String(),
											lbSDK.ACLHTTPFilterPathEnd.String(),
											lbSDK.ACLHTTPFilterRegex.String(),
											lbSDK.ACLHTTPFilterHTTPHeaderMatch.String(),
										}, false),
										Description: "The HTTP filter to match",
									},
									"http_filter_value": {
										Type:        schema.TypeList,
										Optional:    true,
										Description: "A list of possible values to match for the given HTTP filter",
										Elem: &schema.Schema{
											Type: schema.TypeString,
										},
									},
									"http_filter_option": {
										Type:        schema.TypeString,
										Optional:    true,
										Description: "You can use this field with http_header_match acl type to set the header name to filter",
									},
									"invert": {
										Type:        schema.TypeBool,
										Optional:    true,
										Description: `If set to true, the condition will be of type "unless"`,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func resourceScalewayLbFrontendCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	lbAPI, zone, err := lbAPIWithZone(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	lbID := expandID(d.Get("lb_id"))
	if lbID == "" {
		return diag.Errorf("load balancer id wrong format: %v", d.Get("lb_id").(string))
	}

	_, err = waitForLB(ctx, lbAPI, zone, lbID, d.Timeout(schema.TimeoutCreate))
	if err != nil {
		if is403Error(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	timeoutClient, err := expandDuration(d.Get("timeout_client"))
	if err != nil {
		return diag.FromErr(err)
	}

	createFrontendRequest := &lbSDK.ZonedAPICreateFrontendRequest{
		Zone:          zone,
		LBID:          lbID,
		Name:          expandOrGenerateString(d.Get("name"), "lb-frt"),
		InboundPort:   int32(d.Get("inbound_port").(int)),
		BackendID:     expandID(d.Get("backend_id")),
		TimeoutClient: timeoutClient,
	}

	certificatesRaw, certificatesExist := d.GetOk("certificate_ids")
	if certificatesExist {
		createFrontendRequest.CertificateIDs = expandSliceIDsPtr(certificatesRaw)
	}

	frontend, err := lbAPI.CreateFrontend(createFrontendRequest, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(newZonedIDString(zone, frontend.ID))

	diagnostics := resourceScalewayLbFrontendUpdateACL(ctx, d, lbAPI, zone, frontend.ID)
	if diagnostics != nil {
		return diagnostics
	}

	return resourceScalewayLbFrontendRead(ctx, d, meta)
}

func resourceScalewayLbFrontendRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	lbAPI, zone, ID, err := lbAPIWithZoneAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	frontend, err := lbAPI.GetFrontend(&lbSDK.ZonedAPIGetFrontendRequest{
		Zone:       zone,
		FrontendID: ID,
	}, scw.WithContext(ctx))
	if err != nil {
		if is404Error(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	_ = d.Set("lb_id", newZonedIDString(zone, frontend.LB.ID))
	_ = d.Set("backend_id", newZonedIDString(zone, frontend.Backend.ID))
	_ = d.Set("name", frontend.Name)
	_ = d.Set("inbound_port", int(frontend.InboundPort))
	_ = d.Set("timeout_client", flattenDuration(frontend.TimeoutClient))

	if frontend.Certificate != nil {
		_ = d.Set("certificate_id", newZonedIDString(zone, frontend.Certificate.ID))
	} else {
		_ = d.Set("certificate_id", "")
	}

	if len(frontend.CertificateIDs) > 0 {
		_ = d.Set("certificate_ids", flattenSliceIDs(frontend.CertificateIDs, zone))
	}

	//read related acls.
	resACL, err := lbAPI.ListACLs(&lbSDK.ZonedAPIListACLsRequest{
		Zone:       zone,
		FrontendID: ID,
	}, scw.WithAllPages(), scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	_ = d.Set("acl", flattenLBACLs(resACL.ACLs))

	return nil
}

func flattenLBACLs(ACLs []*lbSDK.ACL) interface{} {
	sort.Slice(ACLs, func(i, j int) bool {
		return ACLs[i].Index < ACLs[j].Index
	})
	rawACLs := make([]interface{}, 0, len(ACLs))
	for _, apiACL := range ACLs {
		rawACLs = append(rawACLs, flattenLbACL(apiACL))
	}
	return rawACLs
}

func resourceScalewayLbFrontendUpdateACL(ctx context.Context, d *schema.ResourceData, lbAPI *lbSDK.ZonedAPI, zone scw.Zone, frontendID string) diag.Diagnostics {
	//Fetch existing acl from the api. and convert it to a hashmap with index as key
	resACL, err := lbAPI.ListACLs(&lbSDK.ZonedAPIListACLsRequest{
		Zone:       zone,
		FrontendID: frontendID,
	}, scw.WithAllPages(), scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}
	apiACLs := make(map[int32]*lbSDK.ACL)
	for _, acl := range resACL.ACLs {
		apiACLs[acl.Index] = acl
	}

	//convert state acl and sanitize them a bit
	newACL := expandsLBACLs(d.Get("acl"))

	//loop
	for index, stateACL := range newACL {
		key := int32(index) + 1
		if apiACL, found := apiACLs[key]; found {
			//there is an old acl with the same key. Remove it from array to mark that we've dealt with it
			delete(apiACLs, key)

			//if the state acl doesn't specify a name, set it to the same as the existing rule
			if stateACL.Name == "" {
				stateACL.Name = apiACL.Name
			}
			//Verify if their values are the same and ignore if that's the case, update otherwise
			if aclEquals(stateACL, apiACL) {
				continue
			}
			_, err = lbAPI.UpdateACL(&lbSDK.ZonedAPIUpdateACLRequest{
				Zone:   zone,
				ACLID:  apiACL.ID,
				Name:   stateACL.Name,
				Action: stateACL.Action,
				Match:  stateACL.Match,
				Index:  key,
			})
			if err != nil {
				return diag.FromErr(err)
			}

			continue
		}
		//old acl doesn't exist, create a new one
		_, err = lbAPI.CreateACL(&lbSDK.ZonedAPICreateACLRequest{
			Zone:       zone,
			FrontendID: frontendID,
			Name:       expandOrGenerateString(stateACL.Name, "lb-acl"),
			Action:     stateACL.Action,
			Match:      stateACL.Match,
			Index:      key,
		}, scw.WithContext(ctx))
		if err != nil {
			return diag.FromErr(err)
		}
	}
	//we've finished with all new acl, delete any remaining old one which were not dealt with yet
	for _, acl := range apiACLs {
		err = lbAPI.DeleteACL(&lbSDK.ZonedAPIDeleteACLRequest{
			Zone:  zone,
			ACLID: acl.ID,
		}, scw.WithContext(ctx))
		if err != nil {
			return diag.FromErr(err)
		}
	}
	return nil
}

func expandsLBACLs(raw interface{}) []*lbSDK.ACL {
	d := raw.([]interface{})
	newACL := make([]*lbSDK.ACL, 0)
	for _, rawACL := range d {
		newACL = append(newACL, expandLbACL(rawACL))
	}
	return newACL
}

func resourceScalewayLbFrontendUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	lbAPI, zone, ID, err := lbAPIWithZoneAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	_, lbID, err := parseZonedID(d.Get("lb_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	// check err waiting process
	_, err = waitForLB(ctx, lbAPI, zone, lbID, d.Timeout(schema.TimeoutUpdate))
	if err != nil {
		if is403Error(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	timeoutClient, err := expandDuration(d.Get("timeout_client"))
	if err != nil {
		return diag.FromErr(err)
	}
	req := &lbSDK.ZonedAPIUpdateFrontendRequest{
		Zone:          zone,
		FrontendID:    ID,
		Name:          d.Get("name").(string),
		InboundPort:   int32(d.Get("inbound_port").(int)),
		BackendID:     expandID(d.Get("backend_id")),
		TimeoutClient: timeoutClient,
	}

	if d.HasChanges("certificate_ids") {
		req.CertificateIDs = expandSliceIDsPtr(d.Get("certificate_ids"))
	}

	_, err = lbAPI.UpdateFrontend(req, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	//update acl
	diagnostics := resourceScalewayLbFrontendUpdateACL(ctx, d, lbAPI, zone, ID)
	if diagnostics != nil {
		return diagnostics
	}

	return resourceScalewayLbFrontendRead(ctx, d, meta)
}

func resourceScalewayLbFrontendDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	lbAPI, zone, ID, err := lbAPIWithZoneAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	_, lbID, err := parseZonedID(d.Get("lb_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	err = lbAPI.DeleteFrontend(&lbSDK.ZonedAPIDeleteFrontendRequest{
		Zone:       zone,
		FrontendID: ID,
	}, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = waitForLB(ctx, lbAPI, zone, lbID, d.Timeout(schema.TimeoutDelete))
	if err != nil && !is404Error(err) {
		return diag.FromErr(err)
	}

	return nil
}

func aclEquals(aclA, aclB *lbSDK.ACL) bool {
	if aclA.Name != aclB.Name {
		return false
	}
	if !cmp.Equal(aclA.Match, aclB.Match) {
		return false
	}
	if !cmp.Equal(aclA.Action, aclB.Action) {
		return false
	}
	return true
}
