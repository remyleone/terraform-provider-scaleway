package instance_test

import (
	"fmt"
	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/internal/errs"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/logging"
	"testing"

	"github.com/scaleway/terraform-provider-scaleway/v2/internal/tests"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/scaleway/scaleway-sdk-go/api/instance/v1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func init() {
	resource.AddTestSweepers("scaleway_instance_snapshot", &resource.Sweeper{
		Name: "scaleway_instance_snapshot",
		F:    testSweepInstanceSnapshot,
	})
}

func testSweepInstanceSnapshot(_ string) error {
	return tests.SweepZones(scw.AllZones, func(scwClient *scw.Client, zone scw.Zone) error {
		api := instance.NewAPI(scwClient)
		logging.L.Debugf("sweeper: destroying instance snapshots in (%+v)", zone)

		listSnapshotsResponse, err := api.ListSnapshots(&instance.ListSnapshotsRequest{
			Zone: zone,
		}, scw.WithAllPages())
		if err != nil {
			return fmt.Errorf("error listing instance snapshots in sweeper: %w", err)
		}

		for _, snapshot := range listSnapshotsResponse.Snapshots {
			err := api.DeleteSnapshot(&instance.DeleteSnapshotRequest{
				Zone:       zone,
				SnapshotID: snapshot.ID,
			})
			if err != nil {
				return fmt.Errorf("error deleting instance snapshot in sweeper: %w", err)
			}
		}

		return nil
	})
}

func TestAccScalewayInstanceSnapshot_BlockVolume(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayInstanceVolumeDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource "scaleway_instance_volume" "main" {
						type       = "b_ssd"
						size_in_gb = 20
					}

					resource "scaleway_instance_snapshot" "main" {
						volume_id = scaleway_instance_volume.main.id
					}`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstanceSnapShotExists(tt, "scaleway_instance_snapshot.main"),
				),
			},
		},
	})
}

func TestAccScalewayInstanceSnapshot_Unified(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayInstanceVolumeDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource "scaleway_instance_volume" "main" {
						type       = "l_ssd"
						size_in_gb = 10
					}

					resource "scaleway_instance_server" "main" {
						image    = "ubuntu_jammy"
						type     = "DEV1-S"
						root_volume {
							size_in_gb = 10
							volume_type = "l_ssd"
						}
						additional_volume_ids = [
							scaleway_instance_volume.main.id
						]
					}

					resource "scaleway_instance_snapshot" "main" {
						volume_id = scaleway_instance_volume.main.id
						type = "unified"
						depends_on = [scaleway_instance_server.main]
					}
					`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstanceSnapShotExists(tt, "scaleway_instance_snapshot.main"),
					resource.TestCheckResourceAttr("scaleway_instance_snapshot.main", "type", "unified"),
				),
			},
		},
	})
}

func TestAccScalewayInstanceSnapshot_Server(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayInstanceVolumeDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource "scaleway_instance_server" "main" {
						image = "ubuntu_focal"
						type = "DEV1-S"
					}

					resource "scaleway_instance_snapshot" "main" {
						volume_id = scaleway_instance_server.main.root_volume.0.volume_id
					}`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstanceSnapShotExists(tt, "scaleway_instance_snapshot.main"),
				),
			},
		},
	})
}

func TestAccScalewayInstanceSnapshot_ServerWithBlockVolume(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccCheckScalewayInstanceVolumeDestroy(tt),
			testAccCheckScalewayInstanceServerDestroy(tt),
			testAccCheckScalewayInstanceSnapshotDestroy(tt),
		),
		Steps: []resource.TestStep{
			{
				Config: `
					resource "scaleway_instance_volume" main {
						type       = "b_ssd"
						size_in_gb = 10
					}

					resource "scaleway_instance_server" main {
						image = "ubuntu_focal"
						type = "DEV1-S"
						root_volume {
							size_in_gb = 10
							volume_type = "l_ssd"
						}
						additional_volume_ids = [
							scaleway_instance_volume.main.id
						]
					}

					resource "scaleway_instance_snapshot" main {
						volume_id = scaleway_instance_volume.main.id
					}`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstanceSnapShotExists(tt, "scaleway_instance_snapshot.main"),
				),
			},
		},
	})
}

func TestAccScalewayInstanceSnapshot_RenameSnapshot(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayInstanceVolumeDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource "scaleway_instance_volume" "main" {
						type       = "b_ssd"
						size_in_gb = 20
					}

					resource "scaleway_instance_snapshot" "main" {
						volume_id = scaleway_instance_volume.main.id
						name = "first_name"
						tags = ["test-terraform"]
					}`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstanceSnapShotExists(tt, "scaleway_instance_snapshot.main"),
					resource.TestCheckResourceAttr("scaleway_instance_snapshot.main", "tags.0", "test-terraform"),
				),
			},
			{
				Config: `
					resource "scaleway_instance_volume" "main" {
						type       = "b_ssd"
						size_in_gb = 20
					}

					resource "scaleway_instance_snapshot" "main" {
						volume_id = scaleway_instance_volume.main.id
						name = "second_name"
					}`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstanceSnapShotExists(tt, "scaleway_instance_snapshot.main"),
					resource.TestCheckResourceAttr("scaleway_instance_snapshot.main", "tags.#", "0"),
				),
			},
		},
	})
}

func testAccCheckScalewayInstanceSnapShotExists(tt *tests.TestTools, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("resource not found: %s", n)
		}

		instanceAPI, zone, ID, err := instanceAPIWithZoneAndID(tt.Meta, rs.Primary.ID)
		if err != nil {
			return err
		}

		_, err = instanceAPI.GetSnapshot(&instance.GetSnapshotRequest{
			Zone:       zone,
			SnapshotID: ID,
		})
		if err != nil {
			return err
		}

		return nil
	}
}

func testAccCheckScalewayInstanceSnapshotDestroy(tt *tests.TestTools) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		for _, rs := range state.RootModule().Resources {
			if rs.Type != "scaleway_instance_snapshot" {
				continue
			}

			instanceAPI, zone, ID, err := instanceAPIWithZoneAndID(tt.Meta, rs.Primary.ID)
			if err != nil {
				return err
			}

			_, err = instanceAPI.GetSnapshot(&instance.GetSnapshotRequest{
				SnapshotID: ID,
				Zone:       zone,
			})

			// If no error resource still exist
			if err == nil {
				return fmt.Errorf("snapshot (%s) still exists", rs.Primary.ID)
			}

			// Unexpected api error we return it
			if !http_errors.Is404Error(err) {
				return err
			}
		}

		return nil
	}
}
