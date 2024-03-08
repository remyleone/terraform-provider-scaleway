package rdb

import (
	"context"
	"fmt"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/difffuncs"
	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/internal/errs"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/scaleway/scaleway-sdk-go/api/rdb/v1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func ResourceScalewayRdbDatabase() *schema.Resource {
	return &schema.Resource{
		CreateContext: ResourceScalewayRdbDatabaseCreate,
		ReadContext:   ResourceScalewayRdbDatabaseRead,
		DeleteContext: ResourceScalewayRdbDatabaseDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Timeouts: &schema.ResourceTimeout{
			Create:  schema.DefaultTimeout(defaultRdbInstanceTimeout),
			Delete:  schema.DefaultTimeout(defaultRdbInstanceTimeout),
			Default: schema.DefaultTimeout(defaultRdbInstanceTimeout),
		},
		SchemaVersion: 0,
		Schema: map[string]*schema.Schema{
			"instance_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: locality.UUIDWithLocality(),
				Description:  "Instance on which the database is created",
			},
			"name": {
				Type:        schema.TypeString,
				Description: "Database name",
				Required:    true,
				ForceNew:    true,
				ValidateFunc: validation.All(
					validation.StringLenBetween(1, 63),
					validation.StringNotInSlice([]string{
						"information_schema",
						"mysql",
						"performance_schema",
						"postgres",
						"rdb",
						"rdb",
						"sys",
						"template0",
						"template1",
					}, false),
					validation.StringMatch(regexp.MustCompile(`^[a-zA-Z\d_$-]*$`), "database name must contain only alphanumeric characters, underscores and dashes and it must start with a letter"),
				),
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
			"region": locality.RegionalSchema(),
		},
		CustomizeDiff: difffuncs.CustomizeDiffLocalityCheck("instance_id"),
	}
}

func ResourceScalewayRdbDatabaseCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	rdbAPI := NewRdbAPI(meta)
	region, instanceID, err := locality.ParseRegionalID(d.Get("instance_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = waitForRDBInstance(ctx, rdbAPI, region, instanceID, d.Timeout(schema.TimeoutCreate))
	if err != nil {
		return diag.FromErr(err)
	}

	createReq := &rdb.CreateDatabaseRequest{
		Region:     region,
		InstanceID: instanceID,
		Name:       d.Get("name").(string),
	}

	var db *rdb.Database
	//  wrapper around StateChangeConf that will just retry the database creation
	err = retry.RetryContext(ctx, d.Timeout(schema.TimeoutCreate), func() *retry.RetryError {
		currentDB, errCreateDB := rdbAPI.CreateDatabase(createReq, scw.WithContext(ctx))
		if errCreateDB != nil {
			// WIP: Issue on creation/write database. Need a database stable status
			if http_errors.Is409Error(errCreateDB) {
				return retry.RetryableError(errCreateDB)
			}
			return retry.NonRetryableError(errCreateDB)
		}
		// set database information
		db = currentDB
		return nil
	})
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = waitForRDBInstance(ctx, rdbAPI, region, instanceID, d.Timeout(schema.TimeoutCreate))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(ResourceScalewayRdbDatabaseID(region, instanceID, db.Name))
	_ = d.Set("region", region)

	return ResourceScalewayRdbDatabaseRead(ctx, d, meta)
}

func getDatabase(ctx context.Context, api *rdb.API, r scw.Region, instanceID, dbName string) (*rdb.Database, error) {
	res, err := api.ListDatabases(&rdb.ListDatabasesRequest{
		Region:     r,
		InstanceID: instanceID,
		Name:       &dbName,
	}, scw.WithContext(ctx))
	if err != nil {
		return nil, err
	}

	if len(res.Databases) == 0 {
		return nil, fmt.Errorf("database %s not found", dbName)
	}

	return res.Databases[0], nil
}

func ResourceScalewayRdbDatabaseRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	rdbAPI := NewRdbAPI(meta)
	region, instanceID, databaseName, err := ResourceScalewayRdbDatabaseParseID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	database, err := getDatabase(ctx, rdbAPI, region, instanceID, databaseName)
	if err != nil {
		if http_errors.Is404Error(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	d.SetId(ResourceScalewayRdbDatabaseID(region, instanceID, database.Name))
	_ = d.Set("instance_id", locality.NewRegionalID(region, instanceID).String())
	_ = d.Set("name", database.Name)
	_ = d.Set("owner", database.Owner)
	_ = d.Set("managed", database.Managed)
	_ = d.Set("size", database.Size.String())
	_ = d.Set("region", region)

	return nil
}

func ResourceScalewayRdbDatabaseDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	rdbAPI := NewRdbAPI(meta)
	region, instanceID, databaseName, err := ResourceScalewayRdbDatabaseParseID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = waitForRDBInstance(ctx, rdbAPI, region, instanceID, d.Timeout(schema.TimeoutDelete))
	if err != nil {
		return diag.FromErr(err)
	}

	err = rdbAPI.DeleteDatabase(&rdb.DeleteDatabaseRequest{
		Region:     region,
		InstanceID: instanceID,
		Name:       databaseName,
	}, scw.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = waitForRDBInstance(ctx, rdbAPI, region, instanceID, d.Timeout(schema.TimeoutDelete))
	if err != nil && !http_errors.Is404Error(err) {
		return diag.FromErr(err)
	}

	return nil
}

// Build the resource identifier
// The resource identifier format is "Region/InstanceId/DatabaseName"
func ResourceScalewayRdbDatabaseID(region scw.Region, instanceID string, databaseName string) (resourceID string) {
	return fmt.Sprintf("%s/%s/%s", region, instanceID, databaseName)
}

// Extract instance ID and database from the resource identifier.
// The resource identifier format is "Region/InstanceId/DatabaseId"
func ResourceScalewayRdbDatabaseParseID(resourceID string) (region scw.Region, instanceID string, database string, err error) {
	idParts := strings.Split(resourceID, "/")
	if len(idParts) != 3 {
		return "", "", "", fmt.Errorf("can't parse user resource id: %s", resourceID)
	}
	return scw.Region(idParts[0]), idParts[1], idParts[2], nil
}
