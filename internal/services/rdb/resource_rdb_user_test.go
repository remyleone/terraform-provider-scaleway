package rdb_test

import (
	"errors"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	rdbSDK "github.com/scaleway/scaleway-sdk-go/api/rdb/v1"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/rdb"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/tests"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/tests/checks"
	"testing"
)

func TestAccScalewayRdbUser_Basic(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()

	instanceName := "TestAccScalewayRdbUser_Basic"
	latestEngineVersion := checks.TestAccCheckScalewayRdbEngineGetLatestVersion(tt, tests.PostgreSQLEngineName)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      checks.TestAccCheckScalewayRdbInstanceDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource scaleway_rdb_instance main {
						name = "%s"
						node_type = "db-dev-s"
						engine = %q
						is_ha_cluster = false
						tags = [ "terraform-test", "scaleway_rdb_user", "minimal" ]
					}

					resource scaleway_rdb_user db_user {
						instance_id = scaleway_rdb_instance.main.id
						name = "foo"
						password = "R34lP4sSw#Rd"
						is_admin = true
					}`, instanceName, latestEngineVersion),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRdbUserExists(tt, "scaleway_rdb_instance.main", "scaleway_rdb_user.db_user"),
					resource.TestCheckResourceAttr("scaleway_rdb_user.db_user", "name", "foo"),
					resource.TestCheckResourceAttr("scaleway_rdb_user.db_user", "is_admin", "true"),
				),
			},
			{
				Config: fmt.Sprintf(`
					resource scaleway_rdb_instance main {
						name = "%s"
						node_type = "db-dev-s"
						engine = %q
						is_ha_cluster = false
						tags = [ "terraform-test", "scaleway_rdb_user", "minimal" ]
					}

					resource scaleway_rdb_user db_user {
						instance_id = scaleway_rdb_instance.main.id
						name = "bar"
						password = "R34lP4sSw#Rd"
						is_admin = false
					}`, instanceName, latestEngineVersion),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRdbUserExists(tt, "scaleway_rdb_instance.main", "scaleway_rdb_user.db_user"),
					resource.TestCheckResourceAttr("scaleway_rdb_user.db_user", "name", "bar"),
					resource.TestCheckResourceAttr("scaleway_rdb_user.db_user", "is_admin", "false"),
				),
			},
		},
	})
}

func testAccCheckRdbUserExists(tt *tests.TestTools, instance string, user string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		instanceResource, ok := state.RootModule().Resources[instance]
		if !ok {
			return fmt.Errorf("resource not found: %s", instance)
		}

		userResource, ok := state.RootModule().Resources[user]
		if !ok {
			return fmt.Errorf("resource not found: %s", user)
		}

		rdbAPI, _, _, err := rdb.RdbAPIWithRegionAndID(tt.GetMeta(), instanceResource.Primary.ID)
		if err != nil {
			return err
		}

		region, instanceID, userName, err := rdb.ResourceScalewayRdbUserParseID(userResource.Primary.ID)
		if err != nil {
			return err
		}

		users, err := rdbAPI.ListUsers(&rdbSDK.ListUsersRequest{
			InstanceID: instanceID,
			Region:     region,
			Name:       &userName,
		})
		if err != nil {
			return err
		}

		if len(users.Users) != 1 {
			return errors.New("no user found")
		}

		return nil
	}
}
