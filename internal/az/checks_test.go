package az_test

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	domainSDK "github.com/scaleway/scaleway-sdk-go/api/domain/v2beta1"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/errs"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/domain"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/tests"
)

func CheckDomainRecordDestroy(tt *tests.TestTools) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		for _, rs := range state.RootModule().Resources {
			if rs.Type != "scaleway_domain_record" {
				continue
			}

			// check if the zone still exists
			domainAPI := domain.NewDomainAPI(tt.GetMeta())
			listDNSZones, err := domainAPI.ListDNSZoneRecords(&domainSDK.ListDNSZoneRecordsRequest{
				DNSZone: rs.Primary.Attributes["dns_zone"],
			})
			if errs.Is403Error(err) { // forbidden: subdomain not found
				return nil
			}

			if err != nil {
				return fmt.Errorf("failed to check if domain zone exists: %w", err)
			}

			if listDNSZones.TotalCount > 0 {
				return fmt.Errorf("zone %s still exist", rs.Primary.Attributes["dns_zone"])
			}
			return nil
		}

		return nil
	}
}
