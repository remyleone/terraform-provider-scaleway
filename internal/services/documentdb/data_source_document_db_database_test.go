package documentdb

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/tests"
)

func TestAccScalewayDataSourceDocumentDBDatabase_Basic(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccCheckScalewayDocumentDBInstanceDestroy(tt),
		),
		Steps: []resource.TestStep{
			{
				Config: `
					resource scaleway_documentdb_instance main {
						name = "test-ds-document_db-database-basic"
						node_type = "docdb-play2-pico"
						engine = "FerretDB-1"
						user_name = "my_initial_user"
						password = "thiZ_is_v&ry_s3cret"
						volume_size_in_gb = 20
					}

					resource scaleway_documentdb_database main {
						instance_id = scaleway_documentdb_instance.main.id
						name        = "test-ds-document_db-database-basic"
					}

					data scaleway_documentdb_database main {
						instance_id = scaleway_documentdb_instance.main.id
						name        = scaleway_documentdb_database.main.name
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayDocumentDBDatabaseExists(tt, "scaleway_documentdb_database.main"),

					resource.TestCheckResourceAttrPair("scaleway_documentdb_database.main", "id", "data.scaleway_documentdb_database.main", "id"),
				),
			},
		},
	})
}
