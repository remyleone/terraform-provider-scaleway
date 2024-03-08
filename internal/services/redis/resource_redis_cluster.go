package redis

import (
	"context"
	"errors"
	"fmt"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/difffuncs"
	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/internal/errs"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
	"io"
	"strings"
	"time"

	"github.com/scaleway/terraform-provider-scaleway/v2/internal/project"

	"github.com/scaleway/terraform-provider-scaleway/v2/internal/types"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/customdiff"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/scaleway/scaleway-sdk-go/api/redis/v1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func ResourceScalewayRedisCluster() *schema.Resource {
	return &schema.Resource{
		CreateContext: ResourceScalewayRedisClusterCreate,
		ReadContext:   ResourceScalewayRedisClusterRead,
		UpdateContext: ResourceScalewayRedisClusterUpdate,
		DeleteContext: ResourceScalewayRedisClusterDelete,
		Timeouts: &schema.ResourceTimeout{
			Create:  schema.DefaultTimeout(defaultRedisClusterTimeout),
			Update:  schema.DefaultTimeout(defaultRedisClusterTimeout),
			Delete:  schema.DefaultTimeout(defaultRedisClusterTimeout),
			Default: schema.DefaultTimeout(defaultRedisClusterTimeout),
		},
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		SchemaVersion: 0,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Name of the redis cluster",
			},
			"version": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Redis version of the cluster",
			},
			"node_type": {
				Type:             schema.TypeString,
				Required:         true,
				Description:      "Type of node to use for the cluster",
				DiffSuppressFunc: difffuncs.DiffSuppressFuncIgnoreCase,
			},
			"user_name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the user created when the cluster is created",
			},
			"password": {
				Type:        schema.TypeString,
				Sensitive:   true,
				Required:    true,
				Description: "Password of the user",
			},
			"tags": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "List of tags [\"tag1\", \"tag2\", ...] attached to a redis cluster",
			},
			"cluster_size": {
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
				Description: "Number of nodes for the cluster.",
			},
			"tls_enabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Whether or not TLS is enabled.",
				ForceNew:    true,
			},
			"acl": {
				Type:          schema.TypeSet,
				Description:   "List of acl rules.",
				Optional:      true,
				ConflictsWith: []string{"private_network"},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Description: "ID of the rule (UUID format).",
							Computed:    true,
						},
						"ip": {
							Type:         schema.TypeString,
							Description:  "IPv4 network address of the rule (IP network in a CIDR format).",
							Required:     true,
							ValidateFunc: validation.IsCIDR,
						},
						"description": {
							Type:        schema.TypeString,
							Description: "Description of the rule.",
							Optional:    true,
							Computed:    true,
						},
					},
				},
			},
			"settings": {
				Type:        schema.TypeMap,
				Description: "Map of settings to define for the cluster.",
				Optional:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"private_network": {
				Type:          schema.TypeSet,
				Optional:      true,
				Description:   "Private network specs details",
				ConflictsWith: []string{"acl"},
				Set:           redisPrivateNetworkSetHash,
				DiffSuppressFunc: func(k, oldValue, newValue string, _ *schema.ResourceData) bool {
					// Check if the key is for the 'id' attribute
					if strings.HasSuffix(k, "id") {
						return locality.ExpandID(oldValue) == locality.ExpandID(newValue)
					}
					// For all other attributes, don't suppress the diff
					return false
				},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: locality.UUIDorUUIDWithLocality(),
							Description:  "UUID of the private network to be connected to the cluster",
						},
						"service_ips": {
							Type:     schema.TypeList,
							Optional: true,
							Computed: true,
							Elem: &schema.Schema{
								Type:         schema.TypeString,
								ValidateFunc: validation.IsCIDR,
							},
							Description: "List of IPv4 addresses of the private network with a CIDR notation",
						},
						// computed
						"endpoint_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "UUID of the endpoint to be connected to the cluster",
						},
						"zone": locality.ZoneComputedSchema(),
					},
				},
			},
			// Computed
			"public_network": {
				Type:        schema.TypeList,
				Optional:    true,
				Computed:    true,
				MaxItems:    1,
				Description: "Public network specs details",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"port": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "TCP port of the endpoint",
						},
						"ips": {
							Type: schema.TypeList,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
							Computed: true,
						},
					},
				},
			},
			"certificate": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "public TLS certificate used by redis cluster, empty if tls is disabled",
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The date and time of the creation of the Redis cluster",
			},
			"updated_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The date and time of the last update of the Redis cluster",
			},
			// Common
			"zone":       locality.ZonalSchema(),
			"project_id": project.ProjectIDSchema(),
		},
		CustomizeDiff: customdiff.All(
			difffuncs.CustomizeDiffLocalityCheck("private_network.#.id"),
			customizeDiffMigrateClusterSize(),
		),
	}
}

func customizeDiffMigrateClusterSize() schema.CustomizeDiffFunc {
	return func(_ context.Context, diff *schema.ResourceDiff, _ interface{}) error {
		oldSize, newSize := diff.GetChange("cluster_size")
		if newSize == 2 {
			return errors.New("cluster_size can be either 1 (standalone) ou >3 (cluster mode), not 2")
		}
		if oldSize == 1 && newSize != 1 || newSize.(int) < oldSize.(int) {
			return diff.ForceNew("cluster_size")
		}
		return nil
	}
}

func ResourceScalewayRedisClusterCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	redisAPI, zone, err := RedisAPIWithZone(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	createReq := &redis.CreateClusterRequest{
		Zone:      zone,
		ProjectID: d.Get("project_id").(string),
		Name:      types.ExpandOrGenerateString(d.Get("name"), "redis"),
		Version:   d.Get("version").(string),
		NodeType:  d.Get("node_type").(string),
		UserName:  d.Get("user_name").(string),
		Password:  d.Get("password").(string),
	}

	tags, tagsExist := d.GetOk("tags")
	if tagsExist {
		createReq.Tags = types.ExpandStrings(tags)
	}
	clusterSize, clusterSizeExist := d.GetOk("cluster_size")
	if clusterSizeExist {
		createReq.ClusterSize = scw.Int32Ptr(int32(clusterSize.(int)))
	}
	tlsEnabled, tlsEnabledExist := d.GetOk("tls_enabled")
	if tlsEnabledExist {
		createReq.TLSEnabled = tlsEnabled.(bool)
	}
	aclRules, aclRulesExist := d.GetOk("acl")
	if aclRulesExist {
		rules, err := expandRedisACLSpecs(aclRules)
		if err != nil {
			return diag.FromErr(err)
		}
		createReq.ACLRules = rules
	}
	settings, settingsExist := d.GetOk("settings")
	if settingsExist {
		createReq.ClusterSettings = expandRedisSettings(settings)
	}

	pn, pnExists := d.GetOk("private_network")
	if pnExists {
		pnSpecs, err := expandRedisPrivateNetwork(pn.(*schema.Set).List())
		if err != nil {
			return diag.FromErr(err)
		}
		createReq.Endpoints = pnSpecs
	}

	res, err := redisAPI.CreateCluster(createReq, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(locality.NewZonedIDString(zone, res.ID))

	_, err = waitForRedisCluster(ctx, redisAPI, zone, res.ID, d.Timeout(schema.TimeoutCreate))
	if err != nil {
		return diag.FromErr(err)
	}

	return ResourceScalewayRedisClusterRead(ctx, d, meta)
}

func ResourceScalewayRedisClusterRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	redisAPI, zone, ID, err := RedisAPIWithZoneAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	getReq := &redis.GetClusterRequest{
		Zone:      zone,
		ClusterID: ID,
	}
	cluster, err := redisAPI.GetCluster(getReq, scw.WithContext(ctx))
	if err != nil {
		if http_errors.Is404Error(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	_ = d.Set("name", cluster.Name)
	_ = d.Set("node_type", cluster.NodeType)
	_ = d.Set("user_name", d.Get("user_name").(string))
	_ = d.Set("password", d.Get("password").(string))
	_ = d.Set("zone", cluster.Zone.String())
	_ = d.Set("project_id", cluster.ProjectID)
	_ = d.Set("version", cluster.Version)
	_ = d.Set("cluster_size", int(cluster.ClusterSize))
	_ = d.Set("created_at", cluster.CreatedAt.Format(time.RFC3339))
	_ = d.Set("updated_at", cluster.UpdatedAt.Format(time.RFC3339))
	_ = d.Set("acl", flattenRedisACLs(cluster.ACLRules))
	_ = d.Set("settings", flattenRedisSettings(cluster.ClusterSettings))

	if len(cluster.Tags) > 0 {
		_ = d.Set("tags", cluster.Tags)
	}

	// set endpoints
	pnI, pnExists := flattenRedisPrivateNetwork(cluster.Endpoints)
	if pnExists {
		_ = d.Set("private_network", pnI)
	}
	_ = d.Set("public_network", flattenRedisPublicNetwork(cluster.Endpoints))

	if cluster.TLSEnabled {
		certificate, err := redisAPI.GetClusterCertificate(&redis.GetClusterCertificateRequest{
			Zone:      zone,
			ClusterID: cluster.ID,
		})
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to fetch cluster certificate: %w", err))
		}

		certificateContent, err := io.ReadAll(certificate.Content)
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to read cluster certificate: %w", err))
		}

		_ = d.Set("certificate", string(certificateContent))
	} else {
		_ = d.Set("certificate", "")
	}

	return nil
}

func ResourceScalewayRedisClusterUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	redisAPI, zone, ID, err := RedisAPIWithZoneAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	req := &redis.UpdateClusterRequest{
		Zone:      zone,
		ClusterID: ID,
	}

	if d.HasChange("name") {
		req.Name = types.ExpandStringPtr(d.Get("name"))
	}
	if d.HasChange("user_name") {
		req.UserName = types.ExpandStringPtr(d.Get("user_name"))
	}
	if d.HasChange("password") {
		req.Password = types.ExpandStringPtr(d.Get("password"))
	}
	if d.HasChange("tags") {
		req.Tags = types.ExpandUpdatedStringsPtr(d.Get("tags"))
	}
	if d.HasChange("acl") {
		diagnostics := ResourceScalewayRedisClusterUpdateACL(ctx, d, redisAPI, zone, ID)
		if diagnostics != nil {
			return diagnostics
		}
	}
	if d.HasChange("settings") {
		diagnostics := ResourceScalewayRedisClusterUpdateSettings(ctx, d, redisAPI, zone, ID)
		if diagnostics != nil {
			return diagnostics
		}
	}

	_, err = waitForRedisCluster(ctx, redisAPI, zone, ID, d.Timeout(schema.TimeoutUpdate))
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = redisAPI.UpdateCluster(req, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	migrateClusterRequests := []redis.MigrateClusterRequest(nil)
	if d.HasChange("cluster_size") {
		migrateClusterRequests = append(migrateClusterRequests, redis.MigrateClusterRequest{
			Zone:        zone,
			ClusterID:   ID,
			ClusterSize: scw.Uint32Ptr(uint32(d.Get("cluster_size").(int))),
		})
	}
	if d.HasChange("version") {
		migrateClusterRequests = append(migrateClusterRequests, redis.MigrateClusterRequest{
			Zone:      zone,
			ClusterID: ID,
			Version:   types.ExpandStringPtr(d.Get("version")),
		})
	}
	if d.HasChange("node_type") {
		migrateClusterRequests = append(migrateClusterRequests, redis.MigrateClusterRequest{
			Zone:      zone,
			ClusterID: ID,
			NodeType:  types.ExpandStringPtr(d.Get("node_type")),
		})
	}
	for i := range migrateClusterRequests {
		_, err = waitForRedisCluster(ctx, redisAPI, zone, ID, d.Timeout(schema.TimeoutUpdate))
		if err != nil && !http_errors.Is404Error(err) {
			return diag.FromErr(err)
		}
		_, err = redisAPI.MigrateCluster(&migrateClusterRequests[i], scw.WithContext(ctx))
		if err != nil {
			return diag.FromErr(err)
		}

		_, err = waitForRedisCluster(ctx, redisAPI, zone, ID, d.Timeout(schema.TimeoutUpdate))
		if err != nil && !http_errors.Is404Error(err) {
			return diag.FromErr(err)
		}
	}

	if d.HasChanges("private_network") {
		diagnostics := ResourceScalewayRedisClusterUpdateEndpoints(ctx, d, redisAPI, zone, ID)
		if diagnostics != nil {
			return diagnostics
		}
	}

	_, err = waitForRedisCluster(ctx, redisAPI, zone, ID, d.Timeout(schema.TimeoutUpdate))
	if err != nil {
		return diag.FromErr(err)
	}

	return ResourceScalewayRedisClusterRead(ctx, d, meta)
}

func ResourceScalewayRedisClusterUpdateACL(ctx context.Context, d *schema.ResourceData, redisAPI *redis.API, zone scw.Zone, clusterID string) diag.Diagnostics {
	rules, err := expandRedisACLSpecs(d.Get("acl"))
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = redisAPI.SetACLRules(&redis.SetACLRulesRequest{
		Zone:      zone,
		ClusterID: clusterID,
		ACLRules:  rules,
	}, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func ResourceScalewayRedisClusterUpdateSettings(ctx context.Context, d *schema.ResourceData, redisAPI *redis.API, zone scw.Zone, clusterID string) diag.Diagnostics {
	settings := expandRedisSettings(d.Get("settings"))

	_, err := redisAPI.SetClusterSettings(&redis.SetClusterSettingsRequest{
		Zone:      zone,
		ClusterID: clusterID,
		Settings:  settings,
	}, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func ResourceScalewayRedisClusterUpdateEndpoints(ctx context.Context, d *schema.ResourceData, redisAPI *redis.API, zone scw.Zone, clusterID string) diag.Diagnostics {
	// retrieve state
	cluster, err := waitForRedisCluster(ctx, redisAPI, zone, clusterID, d.Timeout(schema.TimeoutUpdate))
	if err != nil {
		return diag.FromErr(err)
	}

	// get new desired state of endpoints
	rawNewEndpoints := d.Get("private_network")
	newEndpoints, err := expandRedisPrivateNetwork(rawNewEndpoints.(*schema.Set).List())
	if err != nil {
		return diag.FromErr(err)
	}
	if len(newEndpoints) == 0 {
		newEndpoints = append(newEndpoints, &redis.EndpointSpec{
			PublicNetwork: &redis.EndpointSpecPublicNetworkSpec{},
		})
	}
	// send request
	_, err = redisAPI.SetEndpoints(&redis.SetEndpointsRequest{
		Zone:      cluster.Zone,
		ClusterID: cluster.ID,
		Endpoints: newEndpoints,
	}, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = waitForRedisCluster(ctx, redisAPI, zone, clusterID, d.Timeout(schema.TimeoutUpdate))
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func ResourceScalewayRedisClusterDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	redisAPI, zone, ID, err := RedisAPIWithZoneAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = waitForRedisCluster(ctx, redisAPI, zone, ID, d.Timeout(schema.TimeoutDelete))
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = redisAPI.DeleteCluster(&redis.DeleteClusterRequest{
		Zone:      zone,
		ClusterID: ID,
	}, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = waitForRedisCluster(ctx, redisAPI, zone, ID, d.Timeout(schema.TimeoutDelete))
	if err != nil && !http_errors.Is404Error(err) {
		return diag.FromErr(err)
	}

	return nil
}
