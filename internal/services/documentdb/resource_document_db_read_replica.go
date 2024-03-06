package documentdb

import (
	"context"
	"fmt"
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
	documentdb "github.com/scaleway/scaleway-sdk-go/api/documentdb/v1beta1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func ResourceScalewayDocumentDBReadReplica() *schema.Resource {
	return &schema.Resource{
		CreateContext: ResourceScalewayDocumentDBReadReplicaCreate,
		ReadContext:   ResourceScalewayDocumentDBReadReplicaRead,
		UpdateContext: ResourceScalewayDocumentDBReadReplicaUpdate,
		DeleteContext: ResourceScalewayDocumentDBReadReplicaDelete,
		Timeouts: &schema.ResourceTimeout{
			Create:  schema.DefaultTimeout(defaultDocumentDBInstanceTimeout),
			Read:    schema.DefaultTimeout(defaultDocumentDBInstanceTimeout),
			Update:  schema.DefaultTimeout(defaultDocumentDBInstanceTimeout),
			Delete:  schema.DefaultTimeout(defaultDocumentDBInstanceTimeout),
			Default: schema.DefaultTimeout(defaultDocumentDBInstanceTimeout),
		},
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		SchemaVersion: 0,
		Schema: map[string]*schema.Schema{
			"instance_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Id of the rdb instance to replicate",
			},
			"direct_access": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Direct access endpoint, it gives you an IP and a port to access your read-replica",
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						// Endpoints common
						"endpoint_id": {
							Type:        schema.TypeString,
							Description: "UUID of the endpoint (UUID format).",
							Computed:    true,
						},
						"ip": {
							Type:        schema.TypeString,
							Description: "IPv4 address of the endpoint (IP address). Only one of ip and hostname may be set.",
							Computed:    true,
						},
						"port": {
							Type:        schema.TypeInt,
							Description: "TCP port of the endpoint.",
							Computed:    true,
						},
						"name": {
							Type:        schema.TypeString,
							Description: "Name of the endpoint.",
							Computed:    true,
						},
						"hostname": {
							Type:        schema.TypeString,
							Description: "Hostname of the endpoint. Only one of ip and hostname may be set.",
							Computed:    true,
						},
					},
				},
			},
			"private_network": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Private network endpoints",
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						// Private network specific
						"private_network_id": {
							Type:             schema.TypeString,
							Description:      "UUID of the private network to be connected to the read replica (UUID format)",
							ValidateFunc:     locality.UUIDorUUIDWithLocality(),
							DiffSuppressFunc: difffuncs.DiffSuppressFuncLocality,
							Required:         true,
						},
						"service_ip": {
							Type:         schema.TypeString,
							Description:  "The IP network address within the private subnet",
							Optional:     true,
							Computed:     true,
							ValidateFunc: validation.IsCIDR,
						},
						"zone": {
							Type:        schema.TypeString,
							Description: "Private network zone",
							Computed:    true,
						},
						// Endpoints common
						"endpoint_id": {
							Type:        schema.TypeString,
							Description: "UUID of the endpoint (UUID format).",
							Computed:    true,
						},
						"ip": {
							Type:        schema.TypeString,
							Description: "IPv4 address of the endpoint (IP address). Only one of ip and hostname may be set",
							Computed:    true,
						},
						"port": {
							Type:        schema.TypeInt,
							Description: "TCP port of the endpoint",
							Computed:    true,
						},
						"name": {
							Type:        schema.TypeString,
							Description: "Name of the endpoints",
							Computed:    true,
						},
						"hostname": {
							Type:        schema.TypeString,
							Description: "Hostname of the endpoint. Only one of ip and hostname may be set",
							Computed:    true,
						},
					},
				},
			},
			// Common
			"region": locality.RegionalSchema(),
		},
		CustomizeDiff: difffuncs.CustomizeDiffLocalityCheck("instance_id", "private_network.#.private_network_id"),
	}
}

func expandDocumentDBReadReplicaEndpointsSpecDirectAccess(data interface{}) *documentdb.ReadReplicaEndpointSpec {
	if data == nil || len(data.([]interface{})) == 0 {
		return nil
	}

	return &documentdb.ReadReplicaEndpointSpec{
		DirectAccess: new(documentdb.ReadReplicaEndpointSpecDirectAccess),
	}
}

// expandDocumentDBReadReplicaEndpointsSpecPrivateNetwork expand read-replica private network endpoints from schema to specs
func expandDocumentDBReadReplicaEndpointsSpecPrivateNetwork(data interface{}) (*documentdb.ReadReplicaEndpointSpec, error) {
	if data == nil || len(data.([]interface{})) == 0 {
		return nil, nil
	}
	// private_network is a list of size 1
	data = data.([]interface{})[0]

	rawEndpoint := data.(map[string]interface{})

	endpoint := new(documentdb.ReadReplicaEndpointSpec)

	serviceIP := rawEndpoint["service_ip"].(string)
	endpoint.PrivateNetwork = &documentdb.ReadReplicaEndpointSpecPrivateNetwork{
		PrivateNetworkID: locality.ExpandID(rawEndpoint["private_network_id"]),
	}
	if len(serviceIP) > 0 {
		ipNet, err := types.ExpandIPNet(serviceIP)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private_network service_ip (%s): %w", rawEndpoint["service_ip"], err)
		}
		endpoint.PrivateNetwork.ServiceIP = &ipNet
	} else {
		endpoint.PrivateNetwork.IpamConfig = &documentdb.ReadReplicaEndpointSpecPrivateNetworkIpamConfig{}
	}

	return endpoint, nil
}

func waitForDocumentDBReadReplica(ctx context.Context, api *documentdb.API, region scw.Region, id string, timeout time.Duration) (*documentdb.ReadReplica, error) {
	retryInterval := defaultWaitDocumentDBRetryInterval
	if transport.DefaultWaitRetryInterval != nil {
		retryInterval = *transport.DefaultWaitRetryInterval
	}

	return api.WaitForReadReplica(&documentdb.WaitForReadReplicaRequest{
		Region:        region,
		Timeout:       scw.TimeDurationPtr(timeout),
		ReadReplicaID: id,
		RetryInterval: &retryInterval,
	}, scw.WithContext(ctx))
}

// flattenDocumentDBReadReplicaEndpoints flatten read-replica endpoints to directAccess and privateNetwork
func flattenDocumentDBReadReplicaEndpoints(endpoints []*documentdb.Endpoint) (directAccess, privateNetwork interface{}) {
	for _, endpoint := range endpoints {
		rawEndpoint := map[string]interface{}{
			"endpoint_id": endpoint.ID,
			"ip":          types.FlattenIPPtr(endpoint.IP),
			"port":        int(endpoint.Port),
			"name":        endpoint.Name,
			"hostname":    types.FlattenStringPtr(endpoint.Hostname),
		}
		if endpoint.DirectAccess != nil {
			directAccess = rawEndpoint
		}
		if endpoint.PrivateNetwork != nil {
			fetchRegion, err := endpoint.PrivateNetwork.Zone.Region()
			if err != nil {
				return diag.FromErr(err), false
			}
			pnRegionalID := locality.NewRegionalIDString(fetchRegion, endpoint.PrivateNetwork.PrivateNetworkID)
			rawEndpoint["private_network_id"] = pnRegionalID
			rawEndpoint["service_ip"] = endpoint.PrivateNetwork.ServiceIP.String()
			rawEndpoint["zone"] = endpoint.PrivateNetwork.Zone
			privateNetwork = rawEndpoint
		}
	}

	// direct_access and private_network are lists

	if directAccess != nil {
		directAccess = []interface{}{directAccess}
	}
	if privateNetwork != nil {
		privateNetwork = []interface{}{privateNetwork}
	}

	return directAccess, privateNetwork
}

func ResourceScalewayDocumentDBReadReplicaCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, err := documentDBAPIWithRegion(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	endpointSpecs := []*documentdb.ReadReplicaEndpointSpec(nil)
	if directAccess := expandDocumentDBReadReplicaEndpointsSpecDirectAccess(d.Get("direct_access")); directAccess != nil {
		endpointSpecs = append(endpointSpecs, directAccess)
	}
	if pn, err := expandDocumentDBReadReplicaEndpointsSpecPrivateNetwork(d.Get("private_network")); err != nil || pn != nil {
		if err != nil {
			return diag.FromErr(err)
		}
		endpointSpecs = append(endpointSpecs, pn)
	}

	rr, err := api.CreateReadReplica(&documentdb.CreateReadReplicaRequest{
		Region:       region,
		InstanceID:   locality.ExpandID(d.Get("instance_id")),
		EndpointSpec: endpointSpecs,
	}, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to create read-replica: %w", err))
	}

	d.SetId(locality.NewRegionalIDString(region, rr.ID))

	_, err = waitForDocumentDBReadReplica(ctx, api, region, rr.ID, d.Timeout(schema.TimeoutCreate))
	if err != nil {
		return diag.FromErr(err)
	}

	return ResourceScalewayDocumentDBReadReplicaRead(ctx, d, meta)
}

func ResourceScalewayDocumentDBReadReplicaRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, id, err := documentDBAPIWithRegionAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	rr, err := waitForDocumentDBReadReplica(ctx, api, region, id, d.Timeout(schema.TimeoutRead))
	if err != nil {
		if http_errors.Is404Error(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	directAccess, privateNetwork := flattenDocumentDBReadReplicaEndpoints(rr.Endpoints)
	_ = d.Set("direct_access", directAccess)
	_ = d.Set("private_network", privateNetwork)

	_ = d.Set("region", string(region))

	return nil
}

//gocyclo:ignore
func ResourceScalewayDocumentDBReadReplicaUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, id, err := documentDBAPIWithRegionAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = waitForDocumentDBReadReplica(ctx, api, region, id, d.Timeout(schema.TimeoutUpdate))
	if err != nil {
		if http_errors.Is404Error(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	newEndpoints := []*documentdb.ReadReplicaEndpointSpec(nil)

	if d.HasChange("direct_access") {
		_, directAccessExists := d.GetOk("direct_access")
		tflog.Debug(ctx, "direct_access", map[string]interface{}{
			"exists": directAccessExists,
		})
		if !directAccessExists {
			err := api.DeleteEndpoint(&documentdb.DeleteEndpointRequest{
				Region:     region,
				EndpointID: locality.ExpandID(d.Get("direct_access.0.endpoint_id")),
			}, scw.WithContext(ctx))
			if err != nil {
				return diag.FromErr(err)
			}
		} else {
			newEndpoints = append(newEndpoints, expandDocumentDBReadReplicaEndpointsSpecDirectAccess(d.Get("direct_access")))
		}
	}

	if d.HasChange("private_network") {
		_, privateNetworkExists := d.GetOk("private_network")
		if !privateNetworkExists {
			err := api.DeleteEndpoint(&documentdb.DeleteEndpointRequest{
				Region:     region,
				EndpointID: locality.ExpandID(d.Get("private_network.0.endpoint_id")),
			}, scw.WithContext(ctx))
			if err != nil {
				return diag.FromErr(err)
			}
		} else {
			pnEndpoint, err := expandDocumentDBReadReplicaEndpointsSpecPrivateNetwork(d.Get("private_network"))
			if err != nil {
				return diag.FromErr(err)
			}
			newEndpoints = append(newEndpoints, pnEndpoint)
		}
	}

	if len(newEndpoints) > 0 {
		_, err = waitForDocumentDBReadReplica(ctx, api, region, id, d.Timeout(schema.TimeoutUpdate))
		if err != nil {
			return diag.FromErr(err)
		}

		_, err = api.CreateReadReplicaEndpoint(&documentdb.CreateReadReplicaEndpointRequest{
			Region:        region,
			ReadReplicaID: id,
			EndpointSpec:  newEndpoints,
		})
		if err != nil {
			return diag.FromErr(err)
		}
	}

	_, err = waitForDocumentDBReadReplica(ctx, api, region, id, d.Timeout(schema.TimeoutUpdate))
	if err != nil {
		return diag.FromErr(err)
	}

	return ResourceScalewayDocumentDBReadReplicaRead(ctx, d, meta)
}

func ResourceScalewayDocumentDBReadReplicaDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, id, err := documentDBAPIWithRegionAndID(meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = waitForDocumentDBReadReplica(ctx, api, region, id, d.Timeout(schema.TimeoutDelete))
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = api.DeleteReadReplica(&documentdb.DeleteReadReplicaRequest{
		Region:        region,
		ReadReplicaID: id,
	}, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = waitForDocumentDBReadReplica(ctx, api, region, id, d.Timeout(schema.TimeoutDelete))
	if err != nil && !http_errors.Is404Error(err) {
		return diag.FromErr(err)
	}

	return nil
}
