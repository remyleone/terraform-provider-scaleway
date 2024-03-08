package block

import (
	"context"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/transport"
	"time"

	meta2 "github.com/scaleway/terraform-provider-scaleway/v2/internal/meta"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/customdiff"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	block "github.com/scaleway/scaleway-sdk-go/api/block/v1alpha1"
	"github.com/scaleway/scaleway-sdk-go/api/instance/v1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

const (
	defaultBlockTimeout       = 5 * time.Minute
	defaultBlockRetryInterval = 5 * time.Second
	BlockVolumeType           = instance.VolumeServerVolumeType("sbs_volume")
)

// BlockAPIWithZone returns a new block API and the zone for a Create request
func BlockAPIWithZone(d *schema.ResourceData, m interface{}) (*block.API, scw.Zone, error) {
	meta := m.(*meta2.Meta)
	blockAPI := block.NewAPI(meta.GetScwClient())

	zone, err := locality.ExtractZone(d, meta)
	if err != nil {
		return nil, "", err
	}

	return blockAPI, zone, nil
}

// BlockAPIWithZoneAndID returns a new block API with zone and ID extracted from the state
func BlockAPIWithZoneAndID(m interface{}, zonedID string) (*block.API, scw.Zone, string, error) {
	meta := m.(*meta2.Meta)
	blockAPI := block.NewAPI(meta.GetScwClient())

	zone, ID, err := locality.ParseZonedID(zonedID)
	if err != nil {
		return nil, "", "", err
	}

	return blockAPI, zone, ID, nil
}

func waitForBlockVolume(ctx context.Context, blockAPI *block.API, zone scw.Zone, id string, timeout time.Duration) (*block.Volume, error) {
	retryInterval := defaultBlockRetryInterval
	if transport.DefaultWaitRetryInterval != nil {
		retryInterval = *transport.DefaultWaitRetryInterval
	}

	volume, err := blockAPI.WaitForVolumeAndReferences(&block.WaitForVolumeAndReferencesRequest{
		Zone:          zone,
		VolumeID:      id,
		RetryInterval: &retryInterval,
		Timeout:       scw.TimeDurationPtr(timeout),
	}, scw.WithContext(ctx))

	return volume, err
}

// customDiffCannotShrink set key to forceNew if value shrink
func customDiffCannotShrink(key string) schema.CustomizeDiffFunc {
	return customdiff.ForceNewIf(key, func(_ context.Context, d *schema.ResourceDiff, _ interface{}) bool {
		oldValueI, newValueI := d.GetChange(key)
		oldValue := oldValueI.(int)
		newValue := newValueI.(int)

		return oldValue < newValue
	})
}

func waitForBlockSnapshot(ctx context.Context, blockAPI *block.API, zone scw.Zone, id string, timeout time.Duration) (*block.Snapshot, error) {
	retryInterval := defaultBlockRetryInterval
	if transport.DefaultWaitRetryInterval != nil {
		retryInterval = *transport.DefaultWaitRetryInterval
	}

	snapshot, err := blockAPI.WaitForSnapshot(&block.WaitForSnapshotRequest{
		Zone:          zone,
		SnapshotID:    id,
		RetryInterval: &retryInterval,
		Timeout:       scw.TimeDurationPtr(timeout),
	}, scw.WithContext(ctx))

	return snapshot, err
}
