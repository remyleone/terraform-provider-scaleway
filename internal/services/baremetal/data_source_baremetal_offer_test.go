package baremetal_test

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	baremetalSDK "github.com/scaleway/scaleway-sdk-go/api/baremetal/v1"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/baremetal"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/tests"
	"testing"
)

func TestAccScalewayDataSourceBaremetalOffer_Basic(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
					data "scaleway_baremetal_offer" "test1" {
						zone = "fr-par-2"
						name = "EM-A210R-HDD"
					}
					
					data "scaleway_baremetal_offer" "test2" {
						offer_id = data.scaleway_baremetal_offer.test1.offer_id
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayBaremetalOfferExists(tt, "data.scaleway_baremetal_offer.test1"),
					resource.TestCheckResourceAttr("data.scaleway_baremetal_offer.test1", "name", "EM-A210R-HDD"),
					testAccCheckScalewayBaremetalOfferExists(tt, "data.scaleway_baremetal_offer.test2"),
					resource.TestCheckResourceAttrPair("data.scaleway_baremetal_offer.test2", "offer_id", "data.scaleway_baremetal_offer.test1", "offer_id"),
					resource.TestCheckResourceAttr("data.scaleway_baremetal_offer.test2", "name", "EM-A210R-HDD"),
					resource.TestCheckResourceAttr("data.scaleway_baremetal_offer.test2", "commercial_range", "aluminium"),
					resource.TestCheckResourceAttr("data.scaleway_baremetal_offer.test2", "include_disabled", "false"),
					resource.TestCheckResourceAttr("data.scaleway_baremetal_offer.test2", "bandwidth", "1000000000"),
					resource.TestCheckResourceAttr("data.scaleway_baremetal_offer.test2", "commercial_range", "aluminium"),
					// resource.TestCheckResourceAttr("data.scaleway_baremetal_offer.test2", "stock", "available"), // skipping this as stocks vary too much
					resource.TestCheckResourceAttr("data.scaleway_baremetal_offer.test2", "cpu.0.name", "AMD Ryzen PRO 3600"),
					resource.TestCheckResourceAttr("data.scaleway_baremetal_offer.test2", "cpu.0.core_count", "6"),
					resource.TestCheckResourceAttr("data.scaleway_baremetal_offer.test2", "cpu.0.frequency", "3600"),
					resource.TestCheckResourceAttr("data.scaleway_baremetal_offer.test2", "cpu.0.thread_count", "12"),
					resource.TestCheckResourceAttr("data.scaleway_baremetal_offer.test2", "disk.0.type", "HDD"),
					resource.TestCheckResourceAttr("data.scaleway_baremetal_offer.test2", "disk.0.capacity", "1000000000000"),
					resource.TestCheckResourceAttr("data.scaleway_baremetal_offer.test2", "disk.1.type", "HDD"),
					resource.TestCheckResourceAttr("data.scaleway_baremetal_offer.test2", "disk.1.capacity", "1000000000000"),
					resource.TestCheckResourceAttr("data.scaleway_baremetal_offer.test2", "memory.0.type", "DDR4"),
					resource.TestCheckResourceAttr("data.scaleway_baremetal_offer.test2", "memory.0.capacity", "16000000000"),
					resource.TestCheckResourceAttr("data.scaleway_baremetal_offer.test2", "memory.0.frequency", "3200"),
					resource.TestCheckResourceAttr("data.scaleway_baremetal_offer.test2", "memory.0.is_ecc", "true"),
				),
			},
		},
	})
}

func TestAccScalewayDataSourceBaremetalOffer_SubscriptionPeriodHourly(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
					data "scaleway_baremetal_offer" "test1" {
						zone = "fr-par-2"
						name = "EM-A210R-HDD"

						subscription_period = "hourly"
					}
					
					data "scaleway_baremetal_offer" "test2" {
						offer_id = data.scaleway_baremetal_offer.test1.offer_id
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayBaremetalOfferExists(tt, "data.scaleway_baremetal_offer.test1"),
					resource.TestCheckResourceAttr("data.scaleway_baremetal_offer.test1", "name", "EM-A210R-HDD"),
					resource.TestCheckResourceAttr("data.scaleway_baremetal_offer.test1", "subscription_period", "hourly"),
					testAccCheckScalewayBaremetalOfferExists(tt, "data.scaleway_baremetal_offer.test2"),
					resource.TestCheckResourceAttrPair("data.scaleway_baremetal_offer.test2", "offer_id", "data.scaleway_baremetal_offer.test1", "offer_id"),
					resource.TestCheckResourceAttr("data.scaleway_baremetal_offer.test2", "name", "EM-A210R-HDD"),
					resource.TestCheckResourceAttr("data.scaleway_baremetal_offer.test2", "subscription_period", "hourly"),
					resource.TestCheckResourceAttr("data.scaleway_baremetal_offer.test2", "commercial_range", "aluminium"),
					resource.TestCheckResourceAttr("data.scaleway_baremetal_offer.test2", "include_disabled", "false"),
					resource.TestCheckResourceAttr("data.scaleway_baremetal_offer.test2", "bandwidth", "1000000000"),
					resource.TestCheckResourceAttr("data.scaleway_baremetal_offer.test2", "commercial_range", "aluminium"),
					// resource.TestCheckResourceAttr("data.scaleway_baremetal_offer.test2", "stock", "available"), // skipping this as stocks vary too much
					resource.TestCheckResourceAttr("data.scaleway_baremetal_offer.test2", "cpu.0.name", "AMD Ryzen PRO 3600"),
					resource.TestCheckResourceAttr("data.scaleway_baremetal_offer.test2", "cpu.0.core_count", "6"),
					resource.TestCheckResourceAttr("data.scaleway_baremetal_offer.test2", "cpu.0.frequency", "3600"),
					resource.TestCheckResourceAttr("data.scaleway_baremetal_offer.test2", "cpu.0.thread_count", "12"),
					resource.TestCheckResourceAttr("data.scaleway_baremetal_offer.test2", "disk.0.type", "HDD"),
					resource.TestCheckResourceAttr("data.scaleway_baremetal_offer.test2", "disk.0.capacity", "1000000000000"),
					resource.TestCheckResourceAttr("data.scaleway_baremetal_offer.test2", "disk.1.type", "HDD"),
					resource.TestCheckResourceAttr("data.scaleway_baremetal_offer.test2", "disk.1.capacity", "1000000000000"),
					resource.TestCheckResourceAttr("data.scaleway_baremetal_offer.test2", "memory.0.type", "DDR4"),
					resource.TestCheckResourceAttr("data.scaleway_baremetal_offer.test2", "memory.0.capacity", "16000000000"),
					resource.TestCheckResourceAttr("data.scaleway_baremetal_offer.test2", "memory.0.frequency", "3200"),
					resource.TestCheckResourceAttr("data.scaleway_baremetal_offer.test2", "memory.0.is_ecc", "true"),
				),
			},
		},
	})
}

func TestAccScalewayDataSourceBaremetalOffer_SubscriptionPeriodMonthly(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
					data "scaleway_baremetal_offer" "test1" {
						zone = "fr-par-2"
						name = "EM-A210R-HDD"

						subscription_period = "monthly"
					}
					
					data "scaleway_baremetal_offer" "test2" {
						offer_id = data.scaleway_baremetal_offer.test1.offer_id
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayBaremetalOfferExists(tt, "data.scaleway_baremetal_offer.test1"),
					resource.TestCheckResourceAttr("data.scaleway_baremetal_offer.test1", "name", "EM-A210R-HDD"),
					resource.TestCheckResourceAttr("data.scaleway_baremetal_offer.test1", "subscription_period", "monthly"),
					testAccCheckScalewayBaremetalOfferExists(tt, "data.scaleway_baremetal_offer.test2"),
					resource.TestCheckResourceAttrPair("data.scaleway_baremetal_offer.test2", "offer_id", "data.scaleway_baremetal_offer.test1", "offer_id"),
					resource.TestCheckResourceAttr("data.scaleway_baremetal_offer.test2", "name", "EM-A210R-HDD"),
					resource.TestCheckResourceAttr("data.scaleway_baremetal_offer.test2", "subscription_period", "monthly"),
					resource.TestCheckResourceAttr("data.scaleway_baremetal_offer.test2", "commercial_range", "aluminium"),
					resource.TestCheckResourceAttr("data.scaleway_baremetal_offer.test2", "include_disabled", "false"),
					resource.TestCheckResourceAttr("data.scaleway_baremetal_offer.test2", "bandwidth", "1000000000"),
					resource.TestCheckResourceAttr("data.scaleway_baremetal_offer.test2", "commercial_range", "aluminium"),
					// resource.TestCheckResourceAttr("data.scaleway_baremetal_offer.test2", "stock", "available"), // skipping this as stocks vary too much
					resource.TestCheckResourceAttr("data.scaleway_baremetal_offer.test2", "cpu.0.name", "AMD Ryzen PRO 3600"),
					resource.TestCheckResourceAttr("data.scaleway_baremetal_offer.test2", "cpu.0.core_count", "6"),
					resource.TestCheckResourceAttr("data.scaleway_baremetal_offer.test2", "cpu.0.frequency", "3600"),
					resource.TestCheckResourceAttr("data.scaleway_baremetal_offer.test2", "cpu.0.thread_count", "12"),
					resource.TestCheckResourceAttr("data.scaleway_baremetal_offer.test2", "disk.0.type", "HDD"),
					resource.TestCheckResourceAttr("data.scaleway_baremetal_offer.test2", "disk.0.capacity", "1000000000000"),
					resource.TestCheckResourceAttr("data.scaleway_baremetal_offer.test2", "disk.1.type", "HDD"),
					resource.TestCheckResourceAttr("data.scaleway_baremetal_offer.test2", "disk.1.capacity", "1000000000000"),
					resource.TestCheckResourceAttr("data.scaleway_baremetal_offer.test2", "memory.0.type", "DDR4"),
					resource.TestCheckResourceAttr("data.scaleway_baremetal_offer.test2", "memory.0.capacity", "16000000000"),
					resource.TestCheckResourceAttr("data.scaleway_baremetal_offer.test2", "memory.0.frequency", "3200"),
					resource.TestCheckResourceAttr("data.scaleway_baremetal_offer.test2", "memory.0.is_ecc", "true"),
				),
			},
		},
	})
}

func testAccCheckScalewayBaremetalOfferExists(tt *tests.TestTools, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		zone, id, err := locality.ParseZonedID(rs.Primary.ID)
		if err != nil {
			return err
		}

		baremetalAPI := baremetalSDK.NewAPI(tt.GetMeta().GetScwClient())
		_, err = baremetal.BaremetalFindOfferByID(context.Background(), baremetalAPI, zone, id)
		if err != nil {
			return err
		}

		return nil
	}
}
