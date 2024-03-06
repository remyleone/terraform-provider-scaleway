package vpcgw_test

import (
	"fmt"
	"testing"

	"github.com/scaleway/terraform-provider-scaleway/v2/internal/tests"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccScalewayDataSourceVPCPublicGateway_Basic(t *testing.T) {
	tt := tests.NewTestTools(t)
	defer tt.Cleanup()
	pgName := "TestAccScalewayDataSourceVPCPublicGateway_Basic"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { tests.TestAccPreCheck(t) },
		ProviderFactories: tt.ProviderFactories,
		CheckDestroy:      testAccCheckScalewayVPCPublicGatewayDestroy(tt),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "scaleway_vpc_public_gateway" "main" {
						name = "%s"
						type = "VPC-GW-S"
					}

					resource "scaleway_vpc_public_gateway" "with-zone" {
						name = "public-gateway-with-not-default-zone"
						type = "VPC-GW-S"
						zone = "nl-ams-1"
					}
					`, pgName),
			},
			{
				Config: fmt.Sprintf(`
					resource "scaleway_vpc_public_gateway" "main" {
						name = "%s"
						type = "VPC-GW-S"
					}

					resource "scaleway_vpc_public_gateway" "with-zone" {
						name = "public-gateway-with-not-default-zone"
						type = "VPC-GW-S"
						zone = "nl-ams-1"
					}

					data "scaleway_vpc_public_gateway" "pg_test_by_name" {
						name = "${scaleway_vpc_public_gateway.main.name}"
					}

					data "scaleway_vpc_public_gateway" "pg_test_by_id" {
						public_gateway_id = "${scaleway_vpc_public_gateway.main.id}"
					}

					data "scaleway_vpc_public_gateway" "pg_test_by_id_with_zone" {
						public_gateway_id = "${scaleway_vpc_public_gateway.with-zone.id}"
					}

					data "scaleway_vpc_public_gateway" "pg_test_by_name_with_zone" {
						name = "${scaleway_vpc_public_gateway.with-zone.name}"
						zone = "nl-ams-1"
					}
				`, pgName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckScalewayVPCPublicGatewayExists(tt, "scaleway_vpc_public_gateway.main"),
					resource.TestCheckResourceAttrPair(
						"data.scaleway_vpc_public_gateway.pg_test_by_name", "name",
						"scaleway_vpc_public_gateway.main", "name"),
					resource.TestCheckResourceAttrPair(
						"data.scaleway_vpc_public_gateway.pg_test_by_id", "public_gateway_id",
						"scaleway_vpc_public_gateway.main", "id"),
					resource.TestCheckResourceAttrPair(
						"data.scaleway_vpc_public_gateway.pg_test_by_id_with_zone", "public_gateway_id",
						"scaleway_vpc_public_gateway.with-zone", "id"),
					resource.TestCheckResourceAttrPair(
						"data.scaleway_vpc_public_gateway.pg_test_by_name_with_zone", "public_gateway_id",
						"scaleway_vpc_public_gateway.with-zone", "id"),
				),
			},
		},
	})
}
