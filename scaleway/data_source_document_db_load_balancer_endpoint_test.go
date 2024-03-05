package scaleway

import (
	"testing"

	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/tests"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccScalewayDataSourceDocumentDBInstanceLoadBalancer_Basic(t *testing.T) {
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
				resource "scaleway_documentdb_instance" "main" {
				  name              = "test-ds-document_db-instance-basic-load-balancer"
				  node_type         = "docdb-play2-pico"
				  engine            = "FerretDB-1"
				  user_name         = "my_initial_user"
				  password          = "thiZ_is_v&ry_s3cret"
				  volume_size_in_gb = 20
				}
				
				data "scaleway_documentdb_load_balancer_endpoint" "find_by_name" {
				  instance_name = scaleway_documentdb_instance.main.name
				}
				
				data "scaleway_documentdb_load_balancer_endpoint" "find_by_id" {
				  instance_id = scaleway_documentdb_instance.main.id
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayDocumentDBInstanceEndpointExists(tt, "data.scaleway_documentdb_load_balancer_endpoint.find_by_name"),
					testAccCheckScalewayDocumentDBInstanceEndpointExists(tt, "data.scaleway_documentdb_load_balancer_endpoint.find_by_id"),
					resource.TestCheckResourceAttrPair("scaleway_documentdb_instance.main", "name", "data.scaleway_documentdb_load_balancer_endpoint.find_by_name", "instance_name"),
					resource.TestCheckResourceAttrPair("scaleway_documentdb_instance.main", "name", "data.scaleway_documentdb_load_balancer_endpoint.find_by_id", "instance_name"),
				),
			},
		},
	})
}
