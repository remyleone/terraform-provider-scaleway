package fip_test

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	baremetalSDK "github.com/scaleway/scaleway-sdk-go/api/baremetal/v1"
	fipSDK "github.com/scaleway/scaleway-sdk-go/api/flexibleip/v1alpha1"
	"github.com/scaleway/scaleway-sdk-go/scw"
	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/internal/errs"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/logging"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/baremetal"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/fip"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/tests"
	"net"
	"testing"
)

const SSHKeyFlexibleIP = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIM7HUxRyQtB2rnlhQUcbDGCZcTJg7OvoznOiyC9W6IxH opensource@scaleway.com"

func init() {
	resource.AddTestSweepers("scaleway_flexible_ip", &resource.Sweeper{
		Name: "scaleway_flexible_ip",
		F:    testSweepFlexibleIP,
	})
}

func testSweepFlexibleIP(_ string) error {
	return tests.SweepZones(scw.AllZones, func(scwClient *scw.Client, zone scw.Zone) error {
		fipAPI := fipSDK.NewAPI(scwClient)

		listIPs, err := fipAPI.ListFlexibleIPs(&fipSDK.ListFlexibleIPsRequest{Zone: zone}, scw.WithAllPages())
		if err != nil {
			logging.L.Warningf("error listing ips in (%s) in sweeper: %s", zone, err)
			return nil
		}

		for _, ip := range listIPs.FlexibleIPs {
			err := fipAPI.DeleteFlexibleIP(&fipSDK.DeleteFlexibleIPRequest{
				FipID: ip.ID,
				Zone:  zone,
			})
			if err != nil {
				return fmt.Errorf("error deleting ip in sweeper: %s", err)
			}
		}

		return nil
	})
}

func TestAccScalewayFlexibleIP_Basic(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayFlexibleIPDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
						resource "scaleway_flexible_ip" "main" {}
					`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayFlexibleIPExists(tt, "scaleway_flexible_ip.main"),
				),
			},
			{
				ResourceName:            "scaleway_flexible_ip.main",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"is_ipv6"},
			},
		},
	})
}

func TestAccScalewayFlexibleIP_WithZone(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayFlexibleIPDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
						resource "scaleway_flexible_ip" "base" {}
					`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayFlexibleIPExists(tt, "scaleway_flexible_ip.base"),
					resource.TestCheckResourceAttr("scaleway_flexible_ip.base", "zone", "fr-par-1"),
				),
			},
			{
				Config: `
						resource "scaleway_flexible_ip" "base" {
							zone = "nl-ams-1"
						}
					`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayFlexibleIPExists(tt, "scaleway_flexible_ip.base"),
					resource.TestCheckResourceAttr("scaleway_flexible_ip.base", "zone", "nl-ams-1"),
				),
			},
		},
	})
}

func TestAccScalewayFlexibleIP_IPv6(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayFlexibleIPDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
						resource "scaleway_flexible_ip" "main" {
							is_ipv6 = true
						}
					`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayFlexibleIPExists(tt, "scaleway_flexible_ip.main"),
					resource.TestCheckResourceAttr("scaleway_flexible_ip.main", "is_ipv6", "true"),
					testAccCheckScalewayFlexibleIPIsIPv6(tt, "scaleway_flexible_ip.main"),
				),
			},
		},
	})
}

func TestAccScalewayFlexibleIP_CreateAndAttachToBaremetalServer(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()

	SSHKeyName := "TestAccScalewayFlexibleIP_CreateAndAttachToBaremetalServer"
	name := "TestAccScalewayFlexibleIP_CreateAndAttachToBaremetalServer"

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccCheckScalewayFlexibleIPDestroy(tt),
			CheckServerDestroy(tt),
		),
		Steps: []resource.TestStep{
			{
				Config: `
						resource "scaleway_flexible_ip" "base" {
							zone = "fr-par-2"
						}
					`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayFlexibleIPExists(tt, "scaleway_flexible_ip.base"),
					resource.TestCheckResourceAttr("scaleway_flexible_ip.base", "zone", "fr-par-2"),
				),
			},
			{
				Config: fmt.Sprintf(`
						data "scaleway_baremetal_os" "by_id" {
							zone = "fr-par-2"
							name = "Ubuntu"
							version = "22.04 LTS (Jammy Jellyfish)"						
						}

						data "scaleway_baremetal_offer" "my_offer" {
							zone = "fr-par-2"
							name = "EM-B112X-SSD"
						}				

						resource "scaleway_iam_ssh_key" "main" {
							name 	   = "%s"
							public_key = "%s"
						}

						resource "scaleway_baremetal_server" "base" {
							name        = "%s"
							zone        = "fr-par-2"
							offer       = data.scaleway_baremetal_offer.my_offer.offer_id
							os          = data.scaleway_baremetal_os.by_id.os_id

							ssh_key_ids = [ scaleway_iam_ssh_key.main.id ]
						}
					`, SSHKeyName, SSHKeyFlexibleIP, name),
			},
			{
				Config: fmt.Sprintf(`
						data "scaleway_baremetal_os" "by_id" {
							zone = "fr-par-2"
							name = "Ubuntu"
							version = "22.04 LTS (Jammy Jellyfish)"						
						}

						data "scaleway_baremetal_offer" "my_offer" {
							zone = "fr-par-2"
							name = "EM-B112X-SSD"
						}				

						resource "scaleway_iam_ssh_key" "main" {
							name 	   = "%s"
							public_key = "%s"
						}

						resource "scaleway_baremetal_server" "base" {
							name        = "%s"
							zone        = "fr-par-2"
							offer       = data.scaleway_baremetal_offer.my_offer.offer_id
							os          = data.scaleway_baremetal_os.by_id.os_id

							ssh_key_ids = [ scaleway_iam_ssh_key.main.id ]
						}

						resource "scaleway_flexible_ip" "base" {
							server_id = scaleway_baremetal_server.base.id
							zone = "fr-par-2"
						}
					`, SSHKeyName, SSHKeyFlexibleIP, name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayFlexibleIPExists(tt, "scaleway_flexible_ip.base"),
					testAccCheckScalewayFlexibleIPAttachedToBaremetalServer(tt, "scaleway_flexible_ip.base", "scaleway_baremetal_server.base"),
					resource.TestCheckResourceAttr("scaleway_flexible_ip.base", "zone", "fr-par-2"),
				),
			},
		},
	})
}

func TestAccScalewayFlexibleIP_AttachAndDetachFromBaremetalServer(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()

	SSHKeyName := "TestAccScalewayFlexibleIP_AttachAndDetachFromBaremetalServer"
	name := "TestAccScalewayFlexibleIP_AttachAndDetachFromBaremetalServer"
	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccCheckScalewayFlexibleIPDestroy(tt),
			CheckServerDestroy(tt),
		),
		Steps: []resource.TestStep{
			{
				Config: `
						resource "scaleway_flexible_ip" "base" {
							zone = "fr-par-2"
						}
					`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayFlexibleIPExists(tt, "scaleway_flexible_ip.base"),
					resource.TestCheckResourceAttr("scaleway_flexible_ip.base", "zone", "fr-par-2"),
				),
			},
			{
				Config: fmt.Sprintf(`
						data "scaleway_baremetal_os" "by_id" {
							zone = "fr-par-2"
							name = "Ubuntu"
							version = "22.04 LTS (Jammy Jellyfish)"						
						}

						data "scaleway_baremetal_offer" "my_offer" {
							zone = "fr-par-2"
							name = "EM-B112X-SSD"
						}		

						resource "scaleway_iam_ssh_key" "main" {
							name 	   = "%s"
							public_key = "%s"
						}

						resource "scaleway_baremetal_server" "base" {
							name        = "%s"
							zone        = "fr-par-2"
							offer       = data.scaleway_baremetal_offer.my_offer.offer_id
							os          = data.scaleway_baremetal_os.by_id.os_id

							ssh_key_ids = [ scaleway_iam_ssh_key.main.id ]
						}
					`, SSHKeyName, SSHKeyFlexibleIP, name),
			},
			{
				Config: fmt.Sprintf(`
						data "scaleway_baremetal_os" "by_id" {
							zone = "fr-par-2"
							name = "Ubuntu"
							version = "22.04 LTS (Jammy Jellyfish)"						
						}

						data "scaleway_baremetal_offer" "my_offer" {
							zone = "fr-par-2"
							name = "EM-B112X-SSD"
						}		

						resource "scaleway_iam_ssh_key" "main" {
							name 	   = "%s"
							public_key = "%s"
						}

						resource "scaleway_baremetal_server" "base" {
							name        = "%s"
							zone        = "fr-par-2"
							offer       = data.scaleway_baremetal_offer.my_offer.offer_id
							os          = data.scaleway_baremetal_os.by_id.os_id

							ssh_key_ids = [ scaleway_iam_ssh_key.main.id ]
						}

						resource "scaleway_flexible_ip" "base" {
							server_id = scaleway_baremetal_server.base.id
							zone = "fr-par-2"
						}
					`, SSHKeyName, SSHKeyFlexibleIP, name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayFlexibleIPExists(tt, "scaleway_flexible_ip.base"),
					testAccCheckScalewayFlexibleIPAttachedToBaremetalServer(tt, "scaleway_flexible_ip.base", "scaleway_baremetal_server.base"),
					resource.TestCheckResourceAttr("scaleway_flexible_ip.base", "zone", "fr-par-2"),
				),
			},
			{
				Config: `
						resource "scaleway_flexible_ip" "base" {
							zone = "fr-par-2"
						}
					`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayFlexibleIPExists(tt, "scaleway_flexible_ip.base"),
					resource.TestCheckResourceAttr("scaleway_flexible_ip.base", "zone", "fr-par-2"),
				),
			},
		},
	})
}

func testAccCheckScalewayFlexibleIPExists(tt *tests.TestTools, name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("resource not found: %s", name)
		}

		fipAPI, zone, ID, err := fip.FipAPIWithZoneAndID(tt.GetMeta(), rs.Primary.ID)
		if err != nil {
			return err
		}

		_, err = fipAPI.GetFlexibleIP(&fipSDK.GetFlexibleIPRequest{
			FipID: ID,
			Zone:  zone,
		})
		if err != nil {
			return err
		}

		return nil
	}
}

func testAccCheckScalewayFlexibleIPDestroy(tt *tests.TestTools) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "scaleway_flexible_ip" {
				continue
			}

			fipAPI, zone, id, err := fip.FipAPIWithZoneAndID(tt.GetMeta(), rs.Primary.ID)
			if err != nil {
				return err
			}

			_, err = fipAPI.GetFlexibleIP(&fipSDK.GetFlexibleIPRequest{
				FipID: id,
				Zone:  zone,
			})

			// If no error resource still exist
			if err == nil {
				return fmt.Errorf("resource %s(%s) still exist", rs.Type, rs.Primary.ID)
			}

			// Unexpected api error we return it
			// We check for 403 because instance API return 403 for deleted IP
			if !http_errors.Is404Error(err) && !http_errors.Is403Error(err) {
				return err
			}
		}

		return nil
	}
}

func testAccCheckScalewayFlexibleIPAttachedToBaremetalServer(tt *tests.TestTools, ipResource, serverResource string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		ipState, ok := s.RootModule().Resources[ipResource]
		if !ok {
			return fmt.Errorf("resource not found: %s", ipResource)
		}
		serverState, ok := s.RootModule().Resources[serverResource]
		if !ok {
			return fmt.Errorf("resource not found: %s", serverResource)
		}

		baremetalAPI, zoneID, err := baremetal.BaremetalAPIWithZoneAndID(tt.GetMeta(), ipState.Primary.ID)
		if err != nil {
			return err
		}

		server, err := baremetalAPI.GetServer(&baremetalSDK.GetServerRequest{
			Zone:     zoneID.Zone,
			ServerID: locality.ExpandID(serverState.Primary.ID),
		})
		if err != nil {
			return err
		}

		fipAPI, zone, ID, err := fip.FipAPIWithZoneAndID(tt.GetMeta(), ipState.Primary.ID)
		if err != nil {
			return err
		}
		ip, err := fipAPI.GetFlexibleIP(&fipSDK.GetFlexibleIPRequest{
			FipID: ID,
			Zone:  zone,
		})
		if err != nil {
			return err
		}

		if ip.ServerID == nil || server.ID != *ip.ServerID {
			return fmt.Errorf("IDs should be the same in %s and %s: %v is different than %v", ipResource, serverResource, server.ID, ip.ServerID)
		}

		return nil
	}
}

func testAccCheckScalewayFlexibleIPIsIPv6(tt *tests.TestTools, resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		fipAPI, zone, ID, err := fip.FipAPIWithZoneAndID(tt.GetMeta(), rs.Primary.ID)
		if err != nil {
			return err
		}

		flexibleIP, err := fipAPI.GetFlexibleIP(&fipSDK.GetFlexibleIPRequest{
			Zone:  zone,
			FipID: ID,
		})
		if err != nil {
			return err
		}

		if len(flexibleIP.IPAddress.IP.To16()) != net.IPv6len {
			return fmt.Errorf("expected an IPv6 address but got: %s", flexibleIP.IPAddress.String())
		}

		return nil
	}
}
