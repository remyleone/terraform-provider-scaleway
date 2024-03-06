package domain

import (
	"fmt"
	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/internal/errs"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/tests"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	domain "github.com/scaleway/scaleway-sdk-go/api/domain/v2beta1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func TestAccScalewayDomainZone_Basic(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()

	testDNSZone := "test-zone"
	logging.L.Debugf("TestAccScalewayDomainZone_Basic: test dns zone: %s, with domain: %s", testDNSZone, testDomain)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayDomainZoneDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "scaleway_domain_zone" "test" {
						domain    = "%s"
						subdomain = "%s"
					}
				`, testDomain, testDNSZone),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayDomainZoneExists(tt, "scaleway_domain_zone.test"),
					resource.TestCheckResourceAttr("scaleway_domain_zone.test", "subdomain", testDNSZone),
					resource.TestCheckResourceAttr("scaleway_domain_zone.test", "domain", testDomain),
					resource.TestCheckResourceAttr("scaleway_domain_zone.test", "status", "active"),
				),
			},
		},
	})
}

func testAccCheckScalewayDomainZoneExists(tt *tests.TestTools, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("resource not found: %s", n)
		}

		domainAPI := newDomainAPI(tt.Meta)
		listDNSZones, err := domainAPI.ListDNSZones(&domain.ListDNSZonesRequest{
			DNSZone: scw.StringPtr(fmt.Sprintf("%s.%s", rs.Primary.Attributes["subdomain"], rs.Primary.Attributes["domain"])),
		})
		if err != nil {
			return err
		}

		if len(listDNSZones.DNSZones) == 0 {
			return fmt.Errorf("zone (%s) not found in: %s",
				rs.Primary.Attributes["subdomain"],
				rs.Primary.Attributes["domain"],
			)
		}

		return nil
	}
}

func testAccCheckScalewayDomainZoneDestroy(tt *tests.TestTools) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		for _, rs := range state.RootModule().Resources {
			if rs.Type != "scaleway_domain_zone" {
				continue
			}

			// check if the zone still exists
			domainAPI := newDomainAPI(tt.Meta)
			listDNSZones, err := domainAPI.ListDNSZones(&domain.ListDNSZonesRequest{
				DNSZone: scw.StringPtr(fmt.Sprintf("%s.%s", rs.Primary.Attributes["subdomain"], rs.Primary.Attributes["domain"])),
			})

			if http_errors.Is403Error(err) { // forbidden: subdomain not found
				return nil
			}

			if err != nil {
				return err
			}

			if listDNSZones.TotalCount > 0 {
				return fmt.Errorf("zone %s still exist for domain: %s",
					rs.Primary.Attributes["subdomain"],
					rs.Primary.Attributes["domain"])
			}
			return nil
		}

		return nil
	}
}
