package difffuncs

import (
	"context"
	"errors"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/meta"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/types"
	"strings"
)

// DiffSuppressFuncLocality is a SuppressDiffFunc to remove the locality from an ID when checking diff.
// e.g. 2c1a1716-5570-4668-a50a-860c90beabf6 == fr-par-1/2c1a1716-5570-4668-a50a-860c90beabf6
func DiffSuppressFuncLocality(_, oldValue, newValue string, _ *schema.ResourceData) bool {
	return locality.ExpandID(oldValue) == locality.ExpandID(newValue)
}

// CustomizeDiffLocalityCheck create a function that will validate locality IDs stored in given keys
// This locality IDs should have the same locality as the resource
// It will search for zone or region in resource.
// Should not be used on computed keys, if a computed key is going to change on zone/region change
// this function will still block the terraform plan
func CustomizeDiffLocalityCheck(keys ...string) schema.CustomizeDiffFunc {
	return func(_ context.Context, diff *schema.ResourceDiff, i interface{}) error {
		parsedLocality := locality.GetLocality(diff, i.(*meta.Meta))

		if parsedLocality == "" {
			return errors.New("missing locality zone or region to check IDs")
		}

		for _, key := range keys {
			// Handle values in lists
			if strings.Contains(key, "#") {
				listKeys := types.ExpandListKeys(key, diff)

				for _, listKey := range listKeys {
					IDLocality, _, err := locality.ParseLocalizedID(diff.Get(listKey).(string))
					if err == nil && !locality.CompareLocalities(IDLocality, parsedLocality) {
						return fmt.Errorf("given %s %s has different locality than the resource %q", listKey, diff.Get(listKey), parsedLocality)
					}
				}
			} else {
				IDLocality, _, err := locality.ParseLocalizedID(diff.Get(key).(string))
				if err == nil && !locality.CompareLocalities(IDLocality, parsedLocality) {
					return fmt.Errorf("given %s %s has different locality than the resource %q", key, diff.Get(key), parsedLocality)
				}
			}
		}
		return nil
	}
}
