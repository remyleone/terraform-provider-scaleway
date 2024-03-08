package instance_test

import (
	"fmt"
	instanceSDK "github.com/scaleway/scaleway-sdk-go/api/instance/v1"
	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/internal/errs"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/logging"
	"regexp"
	"testing"

	"github.com/scaleway/terraform-provider-scaleway/v2/internal/tests"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func init() {
	resource.AddTestSweepers("scaleway_instance_volume", &resource.Sweeper{
		Name: "scaleway_instance_volume",
		F:    testSweepComputeInstanceVolume,
	})
}

func testSweepComputeInstanceVolume(_ string) error {
	return tests.SweepZones(scw.AllZones, func(scwClient *scw.Client, zone scw.Zone) error {
		instanceAPI := instanceSDK.NewAPI(scwClient)
		logging.L.Debugf("sweeper: destroying the volumes in (%s)", zone)

		listVolumesResponse, err := instanceAPI.ListVolumes(&instanceSDK.ListVolumesRequest{
			Zone: zone,
		}, scw.WithAllPages())
		if err != nil {
			return fmt.Errorf("error listing volumes in sweeper: %s", err)
		}

		for _, volume := range listVolumesResponse.Volumes {
			if volume.Server == nil {
				err := instanceAPI.DeleteVolume(&instanceSDK.DeleteVolumeRequest{
					Zone:     zone,
					VolumeID: volume.ID,
				})
				if err != nil {
					return fmt.Errorf("error deleting volume in sweeper: %s", err)
				}
			}
		}
		return nil
	})
}

func TestAccScalewayInstanceVolume_Basic(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayInstanceVolumeDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource "scaleway_instance_volume" "test" {
						type       = "l_ssd"
						size_in_gb = 20
						tags = ["test-terraform"]
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstanceVolumeExists(tt, "scaleway_instance_volume.test"),
					resource.TestCheckResourceAttr("scaleway_instance_volume.test", "size_in_gb", "20"),
					resource.TestCheckResourceAttr("scaleway_instance_volume.test", "tags.0", "test-terraform"),
				),
			},
			{
				Config: `
					resource "scaleway_instance_volume" "test" {
						type       = "l_ssd"
						name       = "terraform-test"
						size_in_gb = 20
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstanceVolumeExists(tt, "scaleway_instance_volume.test"),
					resource.TestCheckResourceAttr("scaleway_instance_volume.test", "name", "terraform-test"),
					resource.TestCheckResourceAttr("scaleway_instance_volume.test", "size_in_gb", "20"),
					resource.TestCheckResourceAttr("scaleway_instance_volume.test", "tags.#", "0"),
				),
			},
		},
	})
}

func TestAccScalewayInstanceVolume_DifferentNameGenerated(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayInstanceVolumeDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource "scaleway_instance_volume" "test" {
						type       = "l_ssd"
						size_in_gb = 20
					}
				`,
			},
			{
				Config: `
					resource "scaleway_instance_volume" "test" {
						type       = "l_ssd"
						size_in_gb = 20
					}
				`,
			},
		},
	})
}

func TestAccScalewayInstanceVolume_ResizeBlock(t *testing.T) {
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
					}`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstanceVolumeExists(tt, "scaleway_instance_volume.main"),
					resource.TestCheckResourceAttr("scaleway_instance_volume.main", "size_in_gb", "20"),
				),
			},
			{
				Config: `
					resource "scaleway_instance_volume" "main" {
						type       = "b_ssd"
						size_in_gb = 30
					}`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstanceVolumeExists(tt, "scaleway_instance_volume.main"),
					resource.TestCheckResourceAttr("scaleway_instance_volume.main", "size_in_gb", "30"),
				),
			},
		},
	})
}

func TestAccScalewayInstanceVolume_ResizeNotBlock(t *testing.T) {
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
						type       = "l_ssd"
						size_in_gb = 20
					}`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayInstanceVolumeExists(tt, "scaleway_instance_volume.main"),
					resource.TestCheckResourceAttr("scaleway_instance_volume.main", "size_in_gb", "20"),
				),
			},
			{
				Config: `
					resource "scaleway_instance_volume" "main" {
						type       = "l_ssd"
						size_in_gb = 30
					}`,
				ExpectError: regexp.MustCompile("only block volume can be resized"),
			},
		},
	})
}

func TestAccScalewayInstanceVolume_CannotResizeBlockDown(t *testing.T) {
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
					}`,
			},
			{
				Config: `
					resource "scaleway_instance_volume" "main" {
						type       = "b_ssd"
						size_in_gb = 10
					}`,
				ExpectError: regexp.MustCompile("block volumes cannot be resized down"),
			},
		},
	})
}

func TestAccScalewayInstanceVolume_Scratch(t *testing.T) {
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
						type       = "scratch"
						size_in_gb = 20
					}`,
			},
		},
	})
}

func testAccCheckScalewayInstanceVolumeExists(tt *tests.TestTools, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		zone, id, err := locality.ParseZonedID(rs.Primary.ID)
		if err != nil {
			return err
		}

		instanceAPI := instanceSDK.NewAPI(tt.GetMeta().GetScwClient())
		_, err = instanceAPI.GetVolume(&instanceSDK.GetVolumeRequest{
			VolumeID: id,
			Zone:     zone,
		})
		if err != nil {
			return err
		}

		return nil
	}
}

func testAccCheckScalewayInstanceVolumeDestroy(tt *tests.TestTools) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		instanceAPI := instanceSDK.NewAPI(tt.GetMeta().GetScwClient())
		for _, rs := range state.RootModule().Resources {
			if rs.Type != "scaleway_instance_volume" {
				continue
			}

			zone, id, err := locality.ParseZonedID(rs.Primary.ID)
			if err != nil {
				return err
			}

			_, err = instanceAPI.GetVolume(&instanceSDK.GetVolumeRequest{
				Zone:     zone,
				VolumeID: id,
			})

			// If no error resource still exist
			if err == nil {
				return fmt.Errorf("volume (%s) still exists", rs.Primary.ID)
			}

			// Unexpected api error we return it
			if !http_errors.Is404Error(err) {
				return err
			}
		}
		return nil
	}
}
