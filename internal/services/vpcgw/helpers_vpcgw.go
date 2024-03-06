package vpcgw

import (
	"context"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/instance"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/transport"
	"time"

	meta2 "github.com/scaleway/terraform-provider-scaleway/v2/internal/meta"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	vpcgw "github.com/scaleway/scaleway-sdk-go/api/vpcgw/v1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

const (
	defaultVPCGatewayTimeout                   = 10 * time.Minute
	defaultVPCGatewayRetry                     = 5 * time.Second
	defaultVPCPublicGatewayIPReverseDNSTimeout = 10 * time.Minute
)

// vpcgwAPIWithZone returns a new VPC API and the zone for a Create request
func vpcgwAPIWithZone(d *schema.ResourceData, m interface{}) (*vpcgw.API, scw.Zone, error) {
	meta := m.(*meta2.Meta)
	vpcgwAPI := vpcgw.NewAPI(meta.GetScwClient())

	zone, err := locality.ExtractZone(d, meta)
	if err != nil {
		return nil, "", err
	}
	return vpcgwAPI, zone, nil
}

// vpcgwAPIWithZoneAndID
func vpcgwAPIWithZoneAndID(m interface{}, id string) (*vpcgw.API, scw.Zone, string, error) {
	meta := m.(*meta2.Meta)
	vpcgwAPI := vpcgw.NewAPI(meta.GetScwClient())

	zone, ID, err := locality.ParseZonedID(id)
	if err != nil {
		return nil, "", "", err
	}
	return vpcgwAPI, zone, ID, nil
}

func waitForVPCPublicGateway(ctx context.Context, api *vpcgw.API, zone scw.Zone, id string, timeout time.Duration) (*vpcgw.Gateway, error) {
	retryInterval := defaultVPCGatewayRetry
	if transport.DefaultWaitRetryInterval != nil {
		retryInterval = *transport.DefaultWaitRetryInterval
	}

	gateway, err := api.WaitForGateway(&vpcgw.WaitForGatewayRequest{
		Timeout:       scw.TimeDurationPtr(timeout),
		GatewayID:     id,
		RetryInterval: &retryInterval,
		Zone:          zone,
	}, scw.WithContext(ctx))

	return gateway, err
}

func waitForVPCGatewayNetwork(ctx context.Context, api *vpcgw.API, zone scw.Zone, id string, timeout time.Duration) (*vpcgw.GatewayNetwork, error) {
	retryIntervalGWNetwork := defaultVPCGatewayRetry
	if transport.DefaultWaitRetryInterval != nil {
		retryIntervalGWNetwork = *transport.DefaultWaitRetryInterval
	}

	gatewayNetwork, err := api.WaitForGatewayNetwork(&vpcgw.WaitForGatewayNetworkRequest{
		GatewayNetworkID: id,
		Timeout:          scw.TimeDurationPtr(timeout),
		RetryInterval:    &retryIntervalGWNetwork,
		Zone:             zone,
	}, scw.WithContext(ctx))

	return gatewayNetwork, err
}

func waitForDHCPEntries(ctx context.Context, api *vpcgw.API, zone scw.Zone, gatewayID string, macAddress string, timeout time.Duration) (*vpcgw.ListDHCPEntriesResponse, error) {
	retryIntervalDHCPEntries := defaultVPCGatewayRetry
	if transport.DefaultWaitRetryInterval != nil {
		retryIntervalDHCPEntries = *transport.DefaultWaitRetryInterval
	}

	req := &vpcgw.WaitForDHCPEntriesRequest{
		MacAddress:    macAddress,
		Zone:          zone,
		Timeout:       scw.TimeDurationPtr(timeout),
		RetryInterval: &retryIntervalDHCPEntries,
	}

	if gatewayID != "" {
		req.GatewayNetworkID = &gatewayID
	}

	dhcpEntries, err := api.WaitForDHCPEntries(req, scw.WithContext(ctx))
	return dhcpEntries, err
}

func retryUpdateGatewayReverseDNS(ctx context.Context, api *vpcgw.API, req *vpcgw.UpdateIPRequest, timeout time.Duration) error {
	timeoutChannel := time.After(timeout)

	for {
		select {
		case <-time.After(defaultVPCGatewayRetry):
			_, err := api.UpdateIP(req, scw.WithContext(ctx))
			if err != nil && instance.IsIPReverseDNSResolveError(err) {
				continue
			}
			return err
		case <-timeoutChannel:
			_, err := api.UpdateIP(req, scw.WithContext(ctx))
			return err
		}
	}
}

func expandIpamConfig(raw interface{}) *vpcgw.CreateGatewayNetworkRequestIpamConfig {
	if raw == nil || len(raw.([]interface{})) != 1 {
		return nil
	}
	rawMap := raw.([]interface{})[0].(map[string]interface{})

	ipamConfig := &vpcgw.CreateGatewayNetworkRequestIpamConfig{
		PushDefaultRoute: rawMap["push_default_route"].(bool),
	}

	if ipamIPID, ok := rawMap["ipam_ip_id"].(string); ok && ipamIPID != "" {
		ipamConfig.IpamIPID = scw.StringPtr(locality.ExpandRegionalID(ipamIPID).ID)
	}

	return ipamConfig
}

func expandUpdateIpamConfig(raw interface{}) *vpcgw.UpdateGatewayNetworkRequestIpamConfig {
	if raw == nil || len(raw.([]interface{})) != 1 {
		return nil
	}
	rawMap := raw.([]interface{})[0].(map[string]interface{})

	updateIpamConfig := &vpcgw.UpdateGatewayNetworkRequestIpamConfig{
		PushDefaultRoute: scw.BoolPtr(rawMap["push_default_route"].(bool)),
	}

	if ipamIPID, ok := rawMap["ipam_ip_id"].(string); ok && ipamIPID != "" {
		updateIpamConfig.IpamIPID = scw.StringPtr(locality.ExpandRegionalID(ipamIPID).ID)
	}

	return updateIpamConfig
}

func flattenIpamConfig(config *vpcgw.IpamConfig) interface{} {
	if config == nil {
		return nil
	}
	return []map[string]interface{}{
		{
			"push_default_route": config.PushDefaultRoute,
			"ipam_ip_id":         config.IpamIPID,
		},
	}
}
