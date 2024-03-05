package scaleway

import (
	"context"

	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/types"

	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/locality/zonal"

	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/project"

	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/organization"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/scaleway/scaleway-sdk-go/api/lb/v1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func dataSourceScalewayLbBackends() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceScalewayLbBackendsRead,
		Schema: map[string]*schema.Schema{
			"lb_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "backends with a lb id like it are listed.",
			},
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Backends with a name like it are listed.",
			},
			"backends": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Computed: true,
							Type:     schema.TypeString,
						},
						"name": {
							Computed: true,
							Type:     schema.TypeString,
						},
						"lb_id": {
							Computed: true,
							Type:     schema.TypeString,
						},
						"forward_protocol": {
							Computed: true,
							Type:     schema.TypeString,
						},
						"forward_port": {
							Computed: true,
							Type:     schema.TypeInt,
						},
						"forward_port_algorithm": {
							Computed: true,
							Type:     schema.TypeString,
						},
						"sticky_sessions": {
							Computed: true,
							Type:     schema.TypeString,
						},
						"sticky_sessions_cookie_name": {
							Computed: true,
							Type:     schema.TypeString,
						},
						"server_ips": {
							Computed: true,
							Type:     schema.TypeList,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"timeout_server": {
							Computed: true,
							Type:     schema.TypeString,
						},
						"timeout_connect": {
							Computed: true,
							Type:     schema.TypeString,
						},
						"timeout_tunnel": {
							Computed: true,
							Type:     schema.TypeString,
						},
						"on_marked_down_action": {
							Computed: true,
							Type:     schema.TypeString,
						},
						"proxy_protocol": {
							Computed: true,
							Type:     schema.TypeString,
						},
						"failover_host": {
							Computed: true,
							Type:     schema.TypeString,
						},
						"ssl_bridging": {
							Computed: true,
							Type:     schema.TypeBool,
						},
						"ignore_ssl_server_verify": {
							Computed: true,
							Type:     schema.TypeBool,
						},
						"health_check_port": {
							Computed: true,
							Type:     schema.TypeInt,
						},
						"health_check_max_retries": {
							Computed: true,
							Type:     schema.TypeInt,
						},
						"health_check_timeout": {
							Computed: true,
							Type:     schema.TypeString,
						},
						"health_check_delay": {
							Computed: true,
							Type:     schema.TypeString,
						},
						"health_check_tcp": {
							Type:     schema.TypeList,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{},
							},
						},
						"health_check_http": {
							Type:     schema.TypeList,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"uri": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"method": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"code": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"host_header": {
										Computed: true,
										Type:     schema.TypeString,
									},
								},
							},
						},
						"health_check_https": {
							Type:     schema.TypeList,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"uri": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"method": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"code": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"host_header": {
										Computed: true,
										Type:     schema.TypeString,
									},
									"sni": {
										Computed: true,
										Type:     schema.TypeString,
									},
								},
							},
						},
						"created_at": {
							Computed: true,
							Type:     schema.TypeString,
						},
						"update_at": {
							Computed: true,
							Type:     schema.TypeString,
						},
					},
				},
			},
			"zone":            zonal.Schema(),
			"organization_id": organization.OrganizationIDSchema(),
			"project_id":      project.ProjectIDSchema(),
		},
	}
}

func dataSourceScalewayLbBackendsRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	lbAPI, zone, err := lbAPIWithZone(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	_, lbID, err := zonal.ParseZonedID(d.Get("lb_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	res, err := lbAPI.ListBackends(&lb.ZonedAPIListBackendsRequest{
		Zone: zone,
		LBID: lbID,
		Name: types.ExpandStringPtr(d.Get("name")),
	}, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	backends := []interface{}(nil)
	for _, backend := range res.Backends {
		rawBackend := make(map[string]interface{})
		rawBackend["id"] = newZonedID(zone, backend.ID).String()
		rawBackend["name"] = backend.Name
		rawBackend["lb_id"] = zonal.NewZonedIDString(zone, backend.LB.ID)
		rawBackend["created_at"] = flattenTime(backend.CreatedAt)
		rawBackend["update_at"] = flattenTime(backend.UpdatedAt)
		rawBackend["forward_protocol"] = backend.ForwardProtocol
		rawBackend["forward_port"] = backend.ForwardPort
		rawBackend["forward_port_algorithm"] = flattenLbForwardPortAlgorithm(backend.ForwardPortAlgorithm)
		rawBackend["sticky_sessions"] = flattenLbStickySessionsType(backend.StickySessions)
		rawBackend["sticky_sessions_cookie_name"] = backend.StickySessionsCookieName
		rawBackend["server_ips"] = backend.Pool
		rawBackend["timeout_server"] = flattenDuration(backend.TimeoutServer)
		rawBackend["timeout_connect"] = flattenDuration(backend.TimeoutConnect)
		rawBackend["timeout_tunnel"] = flattenDuration(backend.TimeoutTunnel)
		rawBackend["on_marked_down_action"] = flattenLbBackendMarkdownAction(backend.OnMarkedDownAction)
		rawBackend["proxy_protocol"] = flattenLbProxyProtocol(backend.ProxyProtocol)
		rawBackend["failover_host"] = flattenStringPtr(backend.FailoverHost)
		rawBackend["ssl_bridging"] = flattenBoolPtr(backend.SslBridging)
		rawBackend["ignore_ssl_server_verify"] = flattenBoolPtr(backend.IgnoreSslServerVerify)
		rawBackend["health_check_port"] = backend.HealthCheck.Port
		rawBackend["health_check_max_retries"] = backend.HealthCheck.CheckMaxRetries
		rawBackend["health_check_timeout"] = flattenDuration(backend.HealthCheck.CheckTimeout)
		rawBackend["health_check_delay"] = flattenDuration(backend.HealthCheck.CheckDelay)
		rawBackend["health_check_tcp"] = flattenLbHCTCP(backend.HealthCheck.TCPConfig)
		rawBackend["health_check_http"] = flattenLbHCHTTP(backend.HealthCheck.HTTPConfig)
		rawBackend["health_check_https"] = flattenLbHCHTTPS(backend.HealthCheck.HTTPSConfig)

		backends = append(backends, rawBackend)
	}

	d.SetId(zone.String())
	_ = d.Set("backends", backends)

	return nil
}
