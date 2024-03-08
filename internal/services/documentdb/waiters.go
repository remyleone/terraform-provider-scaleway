package documentdb

import (
	"context"
	"github.com/scaleway/scaleway-sdk-go/api/documentdb/v1beta1"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/transport"
	"time"
)

func waitForDocumentDBInstance(ctx context.Context, api *documentdb.API, region scw.Region, id string, timeout time.Duration) (*documentdb.Instance, error) {
	retryInterval := defaultWaitDocumentDBRetryInterval
	if transport.DefaultWaitRetryInterval != nil {
		retryInterval = *transport.DefaultWaitRetryInterval
	}

	instance, err := api.WaitForInstance(&documentdb.WaitForInstanceRequest{
		Region:        region,
		InstanceID:    id,
		RetryInterval: &retryInterval,
		Timeout:       scw.TimeDurationPtr(timeout),
	}, scw.WithContext(ctx))

	return instance, err
}
