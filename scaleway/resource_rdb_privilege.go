package scaleway

import (
	"context"
	"fmt"
	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/scaleway/errors"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/locality"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/verify"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/scaleway/scaleway-sdk-go/api/rdb/v1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func resourceScalewayRdbPrivilege() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceScalewayRdbPrivilegeCreate,
		ReadContext:   resourceScalewayRdbPrivilegeRead,
		DeleteContext: resourceScalewayRdbPrivilegeDelete,
		UpdateContext: resourceScalewayRdbPrivilegeUpdate,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Timeouts: &schema.ResourceTimeout{
			Create:  schema.DefaultTimeout(defaultRdbInstanceTimeout),
			Read:    schema.DefaultTimeout(defaultRdbInstanceTimeout),
			Update:  schema.DefaultTimeout(defaultRdbInstanceTimeout),
			Delete:  schema.DefaultTimeout(defaultRdbInstanceTimeout),
			Default: schema.DefaultTimeout(defaultRdbInstanceTimeout),
		},
		SchemaVersion: 1,
		StateUpgraders: []schema.StateUpgrader{
			{Version: 0, Type: rdbPrivilegeUpgradeV1SchemaType(), Upgrade: rdbPrivilegeV1SchemaUpgradeFunc},
		},
		Schema: map[string]*schema.Schema{
			"instance_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: verify.UUIDorUUIDWithLocality(),
				Description:  "Instance on which the database is created",
			},
			"user_name": {
				Type:        schema.TypeString,
				Description: "User name",
				Required:    true,
			},
			"database_name": {
				Type:        schema.TypeString,
				Description: "Database name",
				Required:    true,
			},
			"permission": {
				Type:        schema.TypeString,
				Description: "Privilege",
				ValidateFunc: validation.StringInSlice([]string{
					rdb.PermissionReadonly.String(),
					rdb.PermissionReadwrite.String(),
					rdb.PermissionAll.String(),
					rdb.PermissionCustom.String(),
					rdb.PermissionNone.String(),
				}, false),
				Required: true,
			},
			// Common
			"region": regionSchema(),
		},
		CustomizeDiff: customizeDiffLocalityCheck("instance_id"),
	}
}

func resourceScalewayRdbPrivilegeCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, err := rdbAPIWithRegion(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	instanceID := locality.ExpandID(d.Get("instance_id").(string))
	_, err = waitForRDBInstance(ctx, api, region, instanceID, d.Timeout(schema.TimeoutCreate))
	if err != nil {
		return diag.FromErr(err)
	}

	userName, _ := d.Get("user_name").(string)
	databaseName, _ := d.Get("database_name").(string)
	createReq := &rdb.SetPrivilegeRequest{
		Region:       region,
		InstanceID:   instanceID,
		DatabaseName: databaseName,
		UserName:     userName,
		Permission:   rdb.Permission(d.Get("permission").(string)),
	}

	//  wrapper around StateChangeConf that will just retry  write on database
	err = retry.RetryContext(ctx, d.Timeout(schema.TimeoutCreate), func() *retry.RetryError {
		_, errSetPrivilege := api.SetPrivilege(createReq, scw.WithContext(ctx))
		if errSetPrivilege != nil {
			if http_errors.Is409Error(errSetPrivilege) {
				_, errWait := waitForRDBInstance(ctx, api, region, instanceID, d.Timeout(schema.TimeoutCreate))
				if errWait != nil {
					return retry.NonRetryableError(errWait)
				}
				return retry.RetryableError(errSetPrivilege)
			}
			return retry.NonRetryableError(errSetPrivilege)
		}
		return nil
	})
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = waitForRDBInstance(ctx, api, region, instanceID, d.Timeout(schema.TimeoutCreate))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(resourceScalewayRdbUserPrivilegeID(region, locality.ExpandID(instanceID), databaseName, userName))

	return resourceScalewayRdbPrivilegeRead(ctx, d, meta)
}

func resourceScalewayRdbPrivilegeRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := newRdbAPI(meta)

	region, instanceID, databaseName, userName, err := resourceScalewayRdbUserPrivilegeParseID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = waitForRDBInstance(ctx, api, region, instanceID, d.Timeout(schema.TimeoutRead))
	if err != nil {
		if http_errors.Is404Error(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	listUsers, err := api.ListUsers(&rdb.ListUsersRequest{
		Region:     region,
		InstanceID: instanceID,
		Name:       &userName,
	}, scw.WithContext(ctx))
	if err != nil {
		if http_errors.Is404Error(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	if listUsers == nil || len(listUsers.Users) == 0 {
		d.SetId("")
		return nil
	}

	res, err := api.ListPrivileges(&rdb.ListPrivilegesRequest{
		Region:       region,
		InstanceID:   instanceID,
		DatabaseName: &databaseName,
		UserName:     &userName,
	}, scw.WithContext(ctx))
	if err != nil {
		if http_errors.Is404Error(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	if len(res.Privileges) == 0 {
		return diag.FromErr(fmt.Errorf("couldn't retrieve privileges for user[%s] on database [%s]", userName, databaseName))
	}
	privilege := res.Privileges[0]
	_ = d.Set("database_name", privilege.DatabaseName)
	_ = d.Set("user_name", privilege.UserName)
	_ = d.Set("permission", privilege.Permission)
	_ = d.Set("instance_id", newRegionalIDString(region, instanceID))
	_ = d.Set("region", region)

	return nil
}

func resourceScalewayRdbPrivilegeUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	rdbAPI := newRdbAPI(meta)
	region, instanceID, databaseName, userName, err := resourceScalewayRdbUserPrivilegeParseID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	_, err = waitForRDBInstance(ctx, rdbAPI, region, instanceID, d.Timeout(schema.TimeoutUpdate))
	if err != nil {
		return diag.FromErr(err)
	}

	listUsers, err := rdbAPI.ListUsers(&rdb.ListUsersRequest{
		Region:     region,
		InstanceID: instanceID,
		Name:       &userName,
	}, scw.WithContext(ctx))
	if err != nil {
		if http_errors.Is404Error(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	if listUsers == nil || len(listUsers.Users) == 0 {
		d.SetId("")
		return nil
	}

	updateReq := &rdb.SetPrivilegeRequest{
		Region:       region,
		InstanceID:   instanceID,
		DatabaseName: databaseName,
		UserName:     userName,
		Permission:   rdb.Permission(d.Get("permission").(string)),
	}

	//  wrapper around StateChangeConf that will just retry the database creation
	err = retry.RetryContext(ctx, d.Timeout(schema.TimeoutUpdate), func() *retry.RetryError {
		_, errSet := rdbAPI.SetPrivilege(updateReq, scw.WithContext(ctx))
		if errSet != nil {
			if http_errors.Is409Error(errSet) {
				_, errWait := waitForRDBInstance(ctx, rdbAPI, region, instanceID, d.Timeout(schema.TimeoutUpdate))
				if errWait != nil {
					return retry.NonRetryableError(errWait)
				}
				return retry.RetryableError(errSet)
			}
			return retry.NonRetryableError(errSet)
		}
		return nil
	})
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = waitForRDBInstance(ctx, rdbAPI, region, instanceID, d.Timeout(schema.TimeoutUpdate))
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

//gocyclo:ignore
func resourceScalewayRdbPrivilegeDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	rdbAPI := newRdbAPI(meta)
	region, instanceID, databaseName, userName, err := resourceScalewayRdbUserPrivilegeParseID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = waitForRDBInstance(ctx, rdbAPI, region, instanceID, d.Timeout(schema.TimeoutDelete))
	if err != nil {
		return diag.FromErr(err)
	}

	_ = d.Set("permission", rdb.PermissionNone)
	listUsers, err := rdbAPI.ListUsers(&rdb.ListUsersRequest{
		Region:     region,
		InstanceID: instanceID,
		Name:       &userName,
	}, scw.WithContext(ctx))
	if err != nil {
		if http_errors.Is404Error(err) {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	if listUsers != nil && len(listUsers.Users) == 0 {
		d.SetId("")
		return nil
	}

	updateReq := &rdb.SetPrivilegeRequest{
		Region:       region,
		InstanceID:   instanceID,
		DatabaseName: databaseName,
		UserName:     userName,
		Permission:   rdb.PermissionNone,
	}

	//  wrapper around StateChangeConf that will just retry the database creation
	err = retry.RetryContext(ctx, defaultRdbInstanceTimeout, func() *retry.RetryError {
		// check if user exist on retry
		listUsers, errUserExist := rdbAPI.ListUsers(&rdb.ListUsersRequest{
			Region:     region,
			InstanceID: instanceID,
			Name:       &userName,
		}, scw.WithContext(ctx))
		if err != nil {
			if http_errors.Is404Error(err) {
				d.SetId("")
				return nil
			}
			return retry.NonRetryableError(errUserExist)
		}

		if listUsers != nil && len(listUsers.Users) == 0 {
			d.SetId("")
			return nil
		}
		_, errSet := rdbAPI.SetPrivilege(updateReq, scw.WithContext(ctx))
		if errSet != nil {
			if http_errors.Is409Error(errSet) {
				_, errWait := waitForRDBInstance(ctx, rdbAPI, region, instanceID, d.Timeout(schema.TimeoutDelete))
				if errWait != nil {
					return retry.NonRetryableError(errWait)
				}
				return retry.RetryableError(errSet)
			}
			return retry.NonRetryableError(errSet)
		}
		return nil
	})
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = waitForRDBInstance(ctx, rdbAPI, region, instanceID, d.Timeout(schema.TimeoutDelete))
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

// Build the resource identifier
// The resource identifier format is "Region/InstanceId/database/UserName"
func resourceScalewayRdbUserPrivilegeID(region scw.Region, instanceID, database, userName string) (resourceID string) {
	return fmt.Sprintf("%s/%s/%s/%s", region, instanceID, database, userName)
}

// resourceScalewayRdbUserPrivilegeParseID: The resource identifier format is "Region/InstanceId/DatabaseName/UserName"
func resourceScalewayRdbUserPrivilegeParseID(resourceID string) (region scw.Region, instanceID, databaseName, userName string, err error) {
	idParts := strings.Split(resourceID, "/")
	if len(idParts) != 4 {
		return "", "", "", "", fmt.Errorf("can't parse user privilege resource id: %s", resourceID)
	}
	return scw.Region(idParts[0]), idParts[1], idParts[2], idParts[3], nil
}
