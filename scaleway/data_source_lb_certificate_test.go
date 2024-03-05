package scaleway

import (
	"testing"

	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/tests"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccScalewayDataSourceLbCertificate_Basic(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayLbIPDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: `
					resource scaleway_lb_ip main {
					}

					resource scaleway_lb main {
					    ip_id = scaleway_lb_ip.main.id
						name = "data-test-lb-cert"
						type = "LB-S"
					}

					resource scaleway_lb_certificate main {
						lb_id = scaleway_lb.main.id
						name = "data-test-lb-cert"
						letsencrypt {
							common_name = "${replace(scaleway_lb.main.ip_address, ".", "-")}.lb.${scaleway_lb.main.region}.scw.cloud"
						}
					}
					
					data "scaleway_lb_certificate" "byID" {
						certificate_id = "${scaleway_lb_certificate.main.id}"
					}
					
					data "scaleway_lb_certificate" "byName" {
						name = "${scaleway_lb_certificate.main.name}"
						lb_id = "${scaleway_lb.main.id}"
					}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.scaleway_lb_certificate.byID", "name",
						"scaleway_lb_certificate.main", "name"),
					resource.TestCheckResourceAttrPair(
						"data.scaleway_lb_certificate.byName", "id",
						"scaleway_lb_certificate.main", "id"),
				),
			},
		},
	})
}
