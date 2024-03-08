package lb

import (
	"context"
	"errors"
	"fmt"
	"github.com/hashicorp/go-cty/cty"
	"github.com/scaleway/scaleway-sdk-go/validation"
	locality2 "github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
)

func LbUpgradeV1SchemaType() cty.Type {
	return cty.Object(map[string]cty.Type{
		"id": cty.String,
	})
}

// LbUpgradeV1SchemaUpgradeFunc allow upgrade the from regional to a zoned resource.
func LbUpgradeV1SchemaUpgradeFunc(_ context.Context, rawState map[string]interface{}, _ interface{}) (map[string]interface{}, error) {
	var err error
	// element id: upgrade
	ID, exist := rawState["id"]
	if !exist {
		return nil, errors.New("upgrade: id not exist")
	}
	rawState["id"], err = LbUpgradeV1RegionalToZonedID(ID.(string))
	if err != nil {
		return nil, err
	}
	// return rawState updated
	return rawState, nil
}

func LbUpgradeV1RegionalToZonedID(element string) (string, error) {
	locality, id, err := locality2.ParseLocalizedID(element)
	// return error if can't parse
	if err != nil {
		return "", fmt.Errorf("upgrade: could not retrieve the locality from `%s`", element)
	}
	// if locality is already zoned return
	if validation.IsZone(locality) {
		return element, nil
	}
	//  append zone 1 as default: e.g. fr-par-1
	return fmt.Sprintf("%s-1/%s", locality, id), nil
}
