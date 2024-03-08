package block_test

import (
	"fmt"
	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/internal/errs"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/logging"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/block"
	"testing"

	"github.com/scaleway/terraform-provider-scaleway/v2/internal/tests"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	blockSDK "github.com/scaleway/scaleway-sdk-go/api/block/v1alpha1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func init() {
	resource.AddTestSweepers("scaleway_block_snapshot", &resource.Sweeper{
		Name: "scaleway_block_snapshot",
		F:    testSweepBlockSnapshot,
	})
}

func testSweepBlockSnapshot(_ string) error {
	return tests.SweepZones((&blockSDK.API{}).Zones(), func(scwClient *scw.Client, zone scw.Zone) error {
		blockAPI := blockSDK.NewAPI(scwClient)
		logging.L.Debugf("sweeper: destroying the block snapshots in (%s)", zone)
		listSnapshots, err := blockAPI.ListSnapshots(
			&blockSDK.ListSnapshotsRequest{
				Zone: zone,
			}, scw.WithAllPages())
		if err != nil {
			return fmt.Errorf("error listing snapshot in (%s) in sweeper: %s", zone, err)
		}

		for _, snapshot := range listSnapshots.Snapshots {
			err := blockAPI.DeleteSnapshot(&blockSDK.DeleteSnapshotRequest{
				SnapshotID: snapshot.ID,
				Zone:       zone,
			})
			if err != nil {
				logging.L.Debugf("sweeper: error (%s)", err)

				return fmt.Errorf("error deleting snapshot in sweeper: %s", err)
			}
		}

		return nil
	})
}

func TestAccScalewayBlockSnapshot_Basic(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayBlockSnapshotDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource scaleway_block_volume main {
						iops = 5000
						size_in_gb = 10
					}

					resource scaleway_block_snapshot main {
						name = "test-block-snapshot-basic"
						volume_id = scaleway_block_volume.main.id
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayBlockSnapshotExists(tt, "scaleway_block_snapshot.main"),
					tests.TestCheckResourceAttrUUID("scaleway_block_snapshot.main", "id"),
					resource.TestCheckResourceAttr("scaleway_block_snapshot.main", "name", "test-block-snapshot-basic"),
				),
			},
		},
	})
}

func testAccCheckScalewayBlockSnapshotExists(tt *tests.TestTools, n string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		rs, ok := state.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("resource not found: %s", n)
		}

		api, zone, id, err := block.BlockAPIWithZoneAndID(tt.GetMeta(), rs.Primary.ID)
		if err != nil {
			return err
		}

		_, err = api.GetSnapshot(&blockSDK.GetSnapshotRequest{
			SnapshotID: id,
			Zone:       zone,
		})
		if err != nil {
			return err
		}

		return nil
	}
}

func testAccCheckScalewayBlockSnapshotDestroy(tt *tests.TestTools) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		for _, rs := range state.RootModule().Resources {
			if rs.Type != "scaleway_block_snapshot" {
				continue
			}

			api, zone, id, err := block.BlockAPIWithZoneAndID(tt.GetMeta(), rs.Primary.ID)
			if err != nil {
				return err
			}

			err = api.DeleteSnapshot(&blockSDK.DeleteSnapshotRequest{
				SnapshotID: id,
				Zone:       zone,
			})

			if err == nil {
				return fmt.Errorf("block snapshot (%s) still exists", rs.Primary.ID)
			}

			if !http_errors.Is404Error(err) {
				return err
			}
		}

		return nil
	}
}
