package rdb

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/scaleway/scaleway-sdk-go/api/rdb/v1"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/types"
)

func flattenInstanceSettings(settings []*rdb.InstanceSetting) interface{} {
	res := make(map[string]string)
	for _, value := range settings {
		res[value.Name] = value.Value
	}

	return res
}

func flattenPrivateNetwork(endpoints []*rdb.Endpoint, enableIpam bool) (interface{}, bool) {
	pnI := []map[string]interface{}(nil)
	for _, endpoint := range endpoints {
		if endpoint.PrivateNetwork != nil {
			pn := endpoint.PrivateNetwork
			fetchRegion, err := pn.Zone.Region()
			if err != nil {
				return diag.FromErr(err), false
			}
			pnRegionalID := locality.NewRegionalIDString(fetchRegion, pn.PrivateNetworkID)
			serviceIP, err := types.FlattenIPNet(pn.ServiceIP)
			if err != nil {
				return pnI, false
			}
			pnI = append(pnI, map[string]interface{}{
				"endpoint_id": endpoint.ID,
				"ip":          types.FlattenIPPtr(endpoint.IP),
				"port":        int(endpoint.Port),
				"name":        endpoint.Name,
				"ip_net":      serviceIP,
				"pn_id":       pnRegionalID,
				"hostname":    types.FlattenStringPtr(endpoint.Hostname),
				"enable_ipam": enableIpam,
			})
			return pnI, true
		}
	}

	return pnI, false
}

func expandLoadBalancer() *rdb.EndpointSpec {
	return &rdb.EndpointSpec{
		LoadBalancer: &rdb.EndpointSpecLoadBalancer{},
	}
}
