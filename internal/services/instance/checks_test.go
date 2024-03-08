package instance_test

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/scaleway/scaleway-sdk-go/api/instance/v1"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/tests"
	"net"
)

func CheckImageExists(tt *tests.TestTools, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}
		zone, ID, err := locality.ParseZonedID(rs.Primary.ID)
		if err != nil {
			return err
		}
		instanceAPI := instance.NewAPI(tt.GetMeta().GetScwClient())
		_, err = instanceAPI.GetImage(&instance.GetImageRequest{
			ImageID: ID,
			Zone:    zone,
		})
		if err != nil {
			return err
		}
		return nil
	}
}

func testCheckResourceAttrIP(name string, key string) resource.TestCheckFunc {
	return tests.TestCheckResourceAttrFunc(name, key, func(value string) error {
		ip := net.ParseIP(value)
		if ip == nil {
			return fmt.Errorf("%s is not a valid IP", value)
		}
		return nil
	})
}

func testCheckResourceAttrIPv6(name string, key string) resource.TestCheckFunc {
	return tests.TestCheckResourceAttrFunc(name, key, func(value string) error {
		ip := net.ParseIP(value)
		if ip.To16() == nil {
			return fmt.Errorf("%s is not a valid IPv6", value)
		}
		return nil
	})
}

func testCheckResourceAttrIPv4(name string, key string) resource.TestCheckFunc {
	return tests.TestCheckResourceAttrFunc(name, key, func(value string) error {
		ip := net.ParseIP(value)
		if ip.To4() == nil {
			return fmt.Errorf("%s is not a valid IPv4", value)
		}
		return nil
	})
}
