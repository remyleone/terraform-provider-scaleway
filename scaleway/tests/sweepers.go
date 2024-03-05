package tests

import (
	"context"

	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway"
)

func Sweep(f func(scwClient *scw.Client) error) error {
	ctx := context.Background()
	meta, err := scaleway.buildMeta(ctx, &scaleway.metaConfig{
		terraformVersion: "terraform-tests",
	})
	if err != nil {
		return err
	}
	return f(meta.GetScwClient())
}

func SweepZones(zones []scw.Zone, f func(scwClient *scw.Client, zone scw.Zone) error) error {
	for _, zone := range zones {
		client, err := scaleway.sharedClientForZone(zone)
		if err != nil {
			return err
		}
		err = f(client, zone)
		if err != nil {
			scaleway.l.Warningf("error running sweepZones, ignoring: %s", err)
		}
	}
	return nil
}
