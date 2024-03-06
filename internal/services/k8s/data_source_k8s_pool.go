package k8s

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/scaleway/scaleway-sdk-go/api/k8s/v1"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/datasource"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/types"
)

func DataSourceScalewayK8SPool() *schema.Resource {
	// Generate datasource schema from resource
	dsSchema := datasource.DatasourceSchemaFromResourceSchema(ResourceScalewayK8SPool().Schema)

	// Set 'Optional' schema elements
	datasource.AddOptionalFieldsToSchema(dsSchema, "name", "region", "cluster_id", "size")

	dsSchema["name"].ConflictsWith = []string{"pool_id"}
	dsSchema["cluster_id"].ConflictsWith = []string{"pool_id"}
	dsSchema["cluster_id"].RequiredWith = []string{"name"}
	dsSchema["pool_id"] = &schema.Schema{
		Type:          schema.TypeString,
		Optional:      true,
		Description:   "The ID of the pool",
		ValidateFunc:  locality.UUIDorUUIDWithLocality(),
		ConflictsWith: []string{"name", "cluster_id"},
	}

	return &schema.Resource{
		ReadContext: DataSourceScalewayK8SPoolRead,

		Schema: dsSchema,
	}
}

func DataSourceScalewayK8SPoolRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	k8sAPI, region, err := k8sAPIWithRegion(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	poolID, ok := d.GetOk("pool_id")
	if !ok {
		poolName := d.Get("name").(string)
		clusterID := locality.ExpandRegionalID(d.Get("cluster_id"))
		res, err := k8sAPI.ListPools(&k8s.ListPoolsRequest{
			Region:    region,
			Name:      types.ExpandStringPtr(poolName),
			ClusterID: clusterID.ID,
		}, scw.WithContext(ctx))
		if err != nil {
			return diag.FromErr(err)
		}

		foundPool, err := datasource.FindExact(
			res.Pools,
			func(s *k8s.Pool) bool { return s.Name == poolName },
			poolName,
		)
		if err != nil {
			return diag.FromErr(err)
		}

		poolID = foundPool.ID
	}

	regionalizedID := locality.DatasourceNewRegionalID(poolID, region)
	d.SetId(regionalizedID)
	_ = d.Set("pool_id", regionalizedID)
	return ResourceScalewayK8SPoolRead(ctx, d, meta)
}
