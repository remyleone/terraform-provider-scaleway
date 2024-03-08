package documentdb_test

import (
	"errors"
	"fmt"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/documentdb"
	"testing"

	"github.com/scaleway/terraform-provider-scaleway/v2/internal/tests"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	document_db "github.com/scaleway/scaleway-sdk-go/api/documentdb/v1beta1"
)

func TestAccScalewayDocumentDBDatabase_Basic(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayDocumentDBInstanceDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource scaleway_documentdb_instance main {
						name = "test-document_db-database-basic"
						node_type = "docdb-play2-pico"
						engine = "FerretDB-1"
						user_name = "my_initial_user"
						password = "thiZ_is_v&ry_s3cret"
						tags = [ "terraform-test", "scaleway_documentdb_database", "minimal" ]
						volume_size_in_gb = 20
					}

					resource scaleway_documentdb_database main {
						instance_id = scaleway_documentdb_instance.main.id
						name        = "test-document_db-database-basic"
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayDocumentDBDatabaseExists(tt, "scaleway_documentdb_database.main"),
					tests.TestCheckResourceAttrUUID("scaleway_documentdb_database.main", "id"),
					resource.TestCheckResourceAttr("scaleway_documentdb_database.main", "name", "test-document_db-database-basic"),
				),
			},
		},
	})
}

func testAccCheckScalewayDocumentDBDatabaseExists(tt *tests.TestTools, n string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		rs, ok := state.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("resource not found: %s", n)
		}

		localizedInstanceID, databaseName, err := documentdb.ResourceScalewayDocumentDBDatabaseName(rs.Primary.ID)
		if err != nil {
			return err
		}

		api, region, instanceID, err := documentdb.DocumentDBAPIWithRegionAndID(tt.GetMeta(), localizedInstanceID)
		if err != nil {
			return err
		}

		resp, err := api.ListDatabases(&document_db.ListDatabasesRequest{
			InstanceID: instanceID,
			Name:       &databaseName,
			Region:     region,
		})
		if err != nil {
			return err
		}

		if len(resp.Databases) != 1 {
			return errors.New("no database found")
		}

		return nil
	}
}
