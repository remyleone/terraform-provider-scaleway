package documentdb

import (
	"context"
	"fmt"
	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/internal/errs"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/project"

	"github.com/scaleway/terraform-provider-scaleway/v2/internal/types"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	documentdb "github.com/scaleway/scaleway-sdk-go/api/documentdb/v1beta1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func ResourceScalewayDocumentDBDatabase() *schema.Resource {
	return &schema.Resource{
		CreateContext: ResourceScalewayDocumentDBDatabaseCreate,
		ReadContext:   ResourceScalewayDocumentDBDatabaseRead,
		DeleteContext: ResourceScalewayDocumentDBDatabaseDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Timeouts: &schema.ResourceTimeout{
			Create:  schema.DefaultTimeout(defaultDocumentDBInstanceTimeout),
			Delete:  schema.DefaultTimeout(defaultDocumentDBInstanceTimeout),
			Default: schema.DefaultTimeout(defaultDocumentDBInstanceTimeout),
		},
		SchemaVersion: 0,
		Schema: map[string]*schema.Schema{
			"instance_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: locality.UUIDorUUIDWithLocality(),
				Description:  "Instance on which the database is created",
			},
			"name": {
				Type:        schema.TypeString,
				Computed:    true,
				Optional:    true,
				ForceNew:    true,
				Description: "The database name",
			},
			"managed": {
				Type:        schema.TypeBool,
				Description: "Whether or not the database is managed",
				Computed:    true,
			},
			"owner": {
				Type:        schema.TypeString,
				Description: "User that own the database",
				Computed:    true,
			},
			"size": {
				Type:        schema.TypeString,
				Description: "Size of the database",
				Computed:    true,
			},
			"region":     locality.RegionalSchema(),
			"project_id": project.ProjectIDSchema(),
		},
	}
}

func ResourceScalewayDocumentDBDatabaseCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, err := documentDBAPIWithRegion(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	instanceID := locality.ExpandID(d.Get("instance_id"))

	_, err = waitForDocumentDBInstance(ctx, api, region, instanceID, d.Timeout(schema.TimeoutCreate))
	if err != nil {
		return diag.FromErr(err)
	}

	database, err := api.CreateDatabase(&documentdb.CreateDatabaseRequest{
		Region:     region,
		InstanceID: instanceID,
		Name:       types.ExpandOrGenerateString(d.Get("name").(string), "database"),
	}, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(ResourceScalewayDocumentDBDatabaseID(region, instanceID, database.Name))

	_, err = waitForDocumentDBInstance(ctx, api, region, instanceID, d.Timeout(schema.TimeoutCreate))
	if err != nil {
		return diag.FromErr(err)
	}

	return ResourceScalewayDocumentDBDatabaseRead(ctx, d, meta)
}

func getDocumentDBDatabase(ctx context.Context, api *documentdb.API, region scw.Region, instanceID string, dbName string) (*documentdb.Database, error) {
	res, err := api.ListDatabases(&documentdb.ListDatabasesRequest{
		Region:     region,
		InstanceID: instanceID,
		Name:       &dbName,
	}, scw.WithContext(ctx))
	if err != nil {
		return nil, err
	}

	if len(res.Databases) == 0 {
		return nil, fmt.Errorf("database %q not found", dbName)
	}

	return res.Databases[0], nil
}

func ResourceScalewayDocumentDBDatabaseRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	instanceLocalizedID, databaseName, err := ResourceScalewayDocumentDBDatabaseName(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	api, region, instanceID, err := documentDBAPIWithRegionAndID(meta, instanceLocalizedID)
	if err != nil {
		return diag.FromErr(err)
	}

	instance, err := waitForDocumentDBInstance(ctx, api, region, instanceID, d.Timeout(schema.TimeoutRead))
	if err != nil {
		if http_errors.Is404Error(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	database, err := getDocumentDBDatabase(ctx, api, region, instanceID, databaseName)
	if err != nil {
		return diag.FromErr(err)
	}

	_ = d.Set("name", database.Name)
	_ = d.Set("region", instance.Region)
	_ = d.Set("owner", database.Owner)
	_ = d.Set("managed", database.Managed)
	_ = d.Set("size", database.Size.String())
	_ = d.Set("project_id", instance.ProjectID)

	return nil
}

func ResourceScalewayDocumentDBDatabaseDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	instanceLocalizedID, databaseName, err := ResourceScalewayDocumentDBDatabaseName(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	api, region, instanceID, err := documentDBAPIWithRegionAndID(meta, instanceLocalizedID)
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = waitForDocumentDBInstance(ctx, api, region, instanceID, d.Timeout(schema.TimeoutDelete))
	if err != nil {
		if http_errors.Is404Error(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	err = api.DeleteDatabase(&documentdb.DeleteDatabaseRequest{
		Region:     region,
		Name:       databaseName,
		InstanceID: instanceID,
	}, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = waitForDocumentDBInstance(ctx, api, region, instanceID, d.Timeout(schema.TimeoutDelete))
	if err != nil && !http_errors.Is404Error(err) {
		return diag.FromErr(err)
	}

	return nil
}
