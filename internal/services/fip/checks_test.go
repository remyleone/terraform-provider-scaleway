package fip_test

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	baremetalSDK "github.com/scaleway/scaleway-sdk-go/api/baremetal/v1"
	lbSDK "github.com/scaleway/scaleway-sdk-go/api/lb/v1"
	"github.com/scaleway/scaleway-sdk-go/scw"
	http_errors "github.com/scaleway/terraform-provider-scaleway/v2/internal/errs"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/baremetal"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/instance"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/services/lb"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/tests"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/transport"
)

func CheckLbIPDestroy(tt *tests.TestTools) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		for _, rs := range state.RootModule().Resources {
			if rs.Type != "scaleway_lb_ip" {
				continue
			}

			lbAPI, zone, ID, err := lb.LbAPIWithZoneAndID(tt.GetMeta(), rs.Primary.ID)
			if err != nil {
				return err
			}

			lbID, lbExist := rs.Primary.Attributes["lb_id"]
			if lbExist && len(lbID) > 0 {
				retryInterval := lb.DefaultWaitLBRetryInterval

				if transport.DefaultWaitRetryInterval != nil {
					retryInterval = *transport.DefaultWaitRetryInterval
				}

				_, err := lbAPI.WaitForLbInstances(&lbSDK.ZonedAPIWaitForLBInstancesRequest{
					Zone:          zone,
					LBID:          lbID,
					Timeout:       scw.TimeDurationPtr(instance.DefaultInstanceServerWaitTimeout),
					RetryInterval: &retryInterval,
				}, scw.WithContext(context.Background()))

				// Unexpected api error we return it
				if !http_errors.Is404Error(err) {
					return err
				}
			}

			err = resource.RetryContext(context.Background(), lb.RetryLbIPInterval, func() *resource.RetryError {
				_, errGet := lbAPI.GetIP(&lbSDK.ZonedAPIGetIPRequest{
					Zone: zone,
					IPID: ID,
				})
				if http_errors.Is403Error(errGet) {
					return resource.RetryableError(errGet)
				}

				return resource.NonRetryableError(errGet)
			})

			// If no error resource still exist
			if err == nil {
				return fmt.Errorf("IP (%s) still exists", rs.Primary.ID)
			}

			// Unexpected api error we return it
			if !http_errors.Is404Error(err) {
				return err
			}
		}

		return nil
	}
}

func CheckServerDestroy(tt *tests.TestTools) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		for _, rs := range state.RootModule().Resources {
			if rs.Type != "scaleway_baremetal_server" {
				continue
			}

			baremetalAPI, zonedID, err := baremetal.BaremetalAPIWithZoneAndID(tt.GetMeta(), rs.Primary.ID)
			if err != nil {
				return err
			}

			_, err = baremetalAPI.GetServer(&baremetalSDK.GetServerRequest{
				ServerID: zonedID.ID,
				Zone:     zonedID.Zone,
			})

			// If no error resource still exist
			if err == nil {
				return fmt.Errorf("server (%s) still exists", rs.Primary.ID)
			}

			// Unexpected api error we return it
			if !http_errors.Is404Error(err) {
				return err
			}
		}
		return nil
	}
}
