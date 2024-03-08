package fip

import (
	"context"
	"github.com/scaleway/scaleway-sdk-go/api/flexibleip/v1alpha1"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/transport"
	"time"
)

func waitFlexibleIP(ctx context.Context, api *flexibleip.API, zone scw.Zone, id string, timeout time.Duration) (*flexibleip.FlexibleIP, error) {
	retryInterval := retryFlexibleIPInterval
	if transport.DefaultWaitRetryInterval != nil {
		retryInterval = *transport.DefaultWaitRetryInterval
	}

	return api.WaitForFlexibleIP(&flexibleip.WaitForFlexibleIPRequest{
		FipID:         id,
		Zone:          zone,
		Timeout:       scw.TimeDurationPtr(timeout),
		RetryInterval: &retryInterval,
	}, scw.WithContext(ctx))
}
