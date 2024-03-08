package documentdb

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	documentdb "github.com/scaleway/scaleway-sdk-go/api/documentdb/v1beta1"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/datasource"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/difffuncs"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/project"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/types"
)

func DataSourceScalewayDocumentDBEndpointLoadBalancer() *schema.Resource {
	return &schema.Resource{
		ReadContext: DataSourceScalewayDocumentDBLoadBalancerRead,
		Schema: map[string]*schema.Schema{
			"instance_id": {
				Type:             schema.TypeString,
				Optional:         true,
				Computed:         true,
				Description:      "Instance on which the endpoint is attached",
				ConflictsWith:    []string{"instance_name"},
				ValidateFunc:     locality.UUIDorUUIDWithLocality(),
				DiffSuppressFunc: difffuncs.DiffSuppressFuncLocality,
			},
			"instance_name": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				Description:   "Instance Name on which the endpoint is attached",
				ConflictsWith: []string{"instance_id"},
			},
			"ip": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The IP of your load balancer service",
			},
			"port": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The port of your load balancer service",
			},
			"name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The name of your load balancer service",
			},
			"hostname": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The hostname of your endpoint",
			},
			"region":     locality.RegionalSchema(),
			"project_id": project.ProjectIDSchema(),
		},
	}
}

func DataSourceScalewayDocumentDBLoadBalancerRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, err := DocumentDBAPIWithRegion(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	rawInstanceID, instanceIDExists := d.GetOk("instance_id")
	if !instanceIDExists {
		rawInstanceName := d.Get("instance_name").(string)
		res, err := api.ListInstances(&documentdb.ListInstancesRequest{
			Region:    region,
			Name:      types.ExpandStringPtr(rawInstanceName),
			ProjectID: types.ExpandStringPtr(d.Get("project_id")),
		})
		if err != nil {
			return diag.FromErr(err)
		}

		foundRawInstance, err := datasource.FindExact(
			res.Instances,
			func(s *documentdb.Instance) bool { return s.Name == rawInstanceName },
			rawInstanceName,
		)
		if err != nil {
			return diag.FromErr(err)
		}

		rawInstanceID = foundRawInstance.ID
	}

	instanceID := locality.ExpandID(rawInstanceID)
	instance, err := waitForDocumentDBInstance(ctx, api, region, instanceID, d.Timeout(schema.TimeoutRead))
	if err != nil {
		return diag.FromErr(err)
	}

	lb := getEndPointDocumentDBLoadBalancer(instance.Endpoints)
	_ = d.Set("instance_id", instanceID)
	_ = d.Set("instance_name", instance.Name)
	_ = d.Set("hostname", types.FlattenStringPtr(lb.Hostname))
	_ = d.Set("port", int(lb.Port))
	_ = d.Set("ip", types.FlattenIPPtr(lb.IP))
	_ = d.Set("name", lb.Name)

	d.SetId(locality.DatasourceNewRegionalID(lb.ID, region))

	return nil
}

func getEndPointDocumentDBLoadBalancer(endpoints []*documentdb.Endpoint) *documentdb.Endpoint {
	for _, endpoint := range endpoints {
		if endpoint.LoadBalancer != nil {
			return endpoint
		}
	}

	return nil
}
