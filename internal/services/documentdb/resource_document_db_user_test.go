package documentdb_test

import (
	"errors"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	documentdbSDK "github.com/scaleway/scaleway-sdk-go/api/documentdb/v1beta1"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/documentdb"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/tests"
	"testing"
)

func TestAccScalewayDocumentDBUser_Basic(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayDocumentDBInstanceDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
				resource "scaleway_documentdb_instance" "main" {
				  name              = "test-documentdb-instance-endpoint-migration"
				  node_type         = "docdb-play2-pico"
				  engine            = "FerretDB-1"
				  user_name         = "my_initial_user"
				  password          = "thiZ_is_v&ry_s3cret"
				  volume_size_in_gb = 20
				}
				
				resource "scaleway_documentdb_user" "db_user" {
				  instance_id = scaleway_documentdb_instance.main.id
				  name        = "foo"
				  password    = "R34lP4sSw#Rd"
				  is_admin    = true
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDocumentDBUserExists(tt, "scaleway_documentdb_instance.main", "scaleway_documentdb_user.db_user"),
					resource.TestCheckResourceAttr("scaleway_documentdb_user.db_user", "name", "foo"),
					resource.TestCheckResourceAttr("scaleway_documentdb_user.db_user", "is_admin", "true"),
				),
			},
			{
				Config: `
				resource "scaleway_documentdb_instance" "main" {
				  name              = "test-documentdb-instance-endpoint-migration"
				  node_type         = "docdb-play2-pico"
				  engine            = "FerretDB-1"
				  user_name         = "my_initial_user"
				  password          = "thiZ_is_v&ry_s3cret"
				  volume_size_in_gb = 20
				}
				
				resource "scaleway_documentdb_user" "db_user" {
				  instance_id = scaleway_documentdb_instance.main.id
				  name        = "bar"
				  password    = "R34lP4sSw#Rd"
				  is_admin    = false
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDocumentDBUserExists(tt, "scaleway_documentdb_instance.main", "scaleway_documentdb_user.db_user"),
					resource.TestCheckResourceAttr("scaleway_documentdb_user.db_user", "name", "bar"),
					resource.TestCheckResourceAttr("scaleway_documentdb_user.db_user", "is_admin", "false"),
				),
			},
		},
	})
}

func testAccCheckDocumentDBUserExists(tt *tests.TestTools, instance string, user string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		rs, ok := state.RootModule().Resources[instance]
		if !ok {
			return fmt.Errorf("resource not found: %s", instance)
		}

		userResource, okUser := state.RootModule().Resources[user]
		if !okUser {
			return fmt.Errorf("resource not found: %s", user)
		}

		api, _, _, err := documentdb.DocumentDBAPIWithRegionAndID(tt.GetMeta(), rs.Primary.ID)
		if err != nil {
			return err
		}

		region, instanceID, userName, err := documentdb.ResourceScalewayDocumentDBUserParseID(userResource.Primary.ID)
		if err != nil {
			return err
		}

		users, err := api.ListUsers(&documentdbSDK.ListUsersRequest{
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
