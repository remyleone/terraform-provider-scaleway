package scaleway

import (
	"fmt"
	"testing"

	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/tests"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	documentdb "github.com/scaleway/scaleway-sdk-go/api/documentdb/v1beta1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func init() {
	resource.AddTestSweepers("scaleway_documentdb_instance", &resource.Sweeper{
		Name: "scaleway_documentdb_instance",
		F:    testSweepDocumentDBInstance,
	})
}

func testSweepDocumentDBInstance(_ string) error {
	return SweepRegions((&documentdb.API{}).Regions(), func(scwClient *scw.Client, region scw.Region) error {
		api := documentdb.NewAPI(scwClient)
		L.Debugf("sweeper: destroying the documentdb instances in (%s)", region)
		listInstances, err := api.ListInstances(
			&documentdb.ListInstancesRequest{
				Region: region,
			}, scw.WithAllPages())
		if err != nil {
			return fmt.Errorf("error listing instance in (%s) in sweeper: %s", region, err)
		}

		for _, instance := range listInstances.Instances {
			_, err := api.DeleteInstance(&documentdb.DeleteInstanceRequest{
				InstanceID: instance.ID,
				Region:     region,
			})
			if err != nil {
				L.Debugf("sweeper: error (%s)", err)

				return fmt.Errorf("error deleting instance in sweeper: %s", err)
			}
		}

		return nil
	})
}

func TestAccScalewayDocumentDBInstance_Basic(t *testing.T) {
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
				  name              = "test-documentdb-instance-basic"
				  node_type         = "docdb-play2-pico"
				  engine            = "FerretDB-1"
				  user_name         = "my_initial_user"
				  password          = "thiZ_is_v&ry_s3cret"
				  tags              = ["terraform-test", "scaleway_documentdb_instance", "minimal"]
				  volume_size_in_gb = 20
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayDocumentDBInstanceExists(tt, "scaleway_documentdb_instance.main"),
					testCheckResourceAttrUUID("scaleway_documentdb_instance.main", "id"),
					resource.TestCheckResourceAttr("scaleway_documentdb_instance.main", "name", "test-documentdb-instance-basic"),
				),
			},
		},
	})
}

func testAccCheckScalewayDocumentDBInstanceExists(tt *tests.TestTools, n string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		rs, ok := state.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("resource not found: %s", n)
		}

		api, region, id, err := documentDBAPIWithRegionAndID(tt.Meta, rs.Primary.ID)
		if err != nil {
			return err
		}

		_, err = api.GetInstance(&documentdb.GetInstanceRequest{
			InstanceID: id,
			Region:     region,
		})
		if err != nil {
			return err
		}

		return nil
	}
}

func testAccCheckScalewayDocumentDBInstanceDestroy(tt *tests.TestTools) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		for _, rs := range state.RootModule().Resources {
			if rs.Type != "scaleway_documentdb_instance" {
				continue
			}

			api, region, id, err := documentDBAPIWithRegionAndID(tt.Meta, rs.Primary.ID)
			if err != nil {
				return err
			}

			_, err = api.DeleteInstance(&documentdb.DeleteInstanceRequest{
				InstanceID: id,
				Region:     region,
			})

			if err == nil {
				return fmt.Errorf("documentdb instance (%s) still exists", rs.Primary.ID)
			}

			if !http_errors.Is404Error(err) {
				return err
			}
		}

		return nil
	}
}
