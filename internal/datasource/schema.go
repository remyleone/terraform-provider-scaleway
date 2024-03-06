package datasource

import (
	"errors"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/errs"
)

////
// The below methods are imported from Google's terraform provider.
// source: https://github.com/terraform-providers/terraform-provider-google/blob/master/google/datasource_helpers.go
////

// DatasourceSchemaFromResourceSchema is a recursive func that
// converts an existing Resource schema to a Datasource schema.
// All schema elements are copied, but certain attributes are ignored or changed:
// - all attributes have Computed = true
// - all attributes have ForceNew, Required = false
// - Validation funcs and attributes (e.g. MaxItems) are not copied
func DatasourceSchemaFromResourceSchema(rs map[string]*schema.Schema) map[string]*schema.Schema {
	ds := make(map[string]*schema.Schema, len(rs))
	for k, v := range rs {
		dv := &schema.Schema{
			Computed:    true,
			ForceNew:    false,
			Description: v.Description,
			Type:        v.Type,
		}

		switch v.Type {
		case schema.TypeSet:
			dv.Set = v.Set
			fallthrough
		case schema.TypeList:
			// List & Set types are generally used for 2 cases:
			// - a list/set of simple primitive values (e.g. list of strings)
			// - a sub resource
			if elem, ok := v.Elem.(*schema.Resource); ok {
				// handle the case where the Element is a sub-resource
				dv.Elem = &schema.Resource{
					Schema: DatasourceSchemaFromResourceSchema(elem.Schema),
				}
			} else {
				// handle simple primitive case
				dv.Elem = v.Elem
			}

		default:
			// Elem of all other types are copied as-is
			dv.Elem = v.Elem
		}
		ds[k] = dv
	}
	return ds
}

// FixDatasourceSchemaFlags is a convenience func that toggles the Computed,
// Optional + Required flags on a schema element. This is useful when the schema
// has been generated (using `datasourceSchemaFromResourceSchema` above for
// example) and therefore the attribute flags were not set appropriately when
// first added to the schema definition. Currently only supports top-level
// schema elements.
func FixDatasourceSchemaFlags(schema map[string]*schema.Schema, required bool, keys ...string) {
	for _, v := range keys {
		schema[v].Computed = false
		schema[v].Optional = !required
		schema[v].Required = required
	}
}

func AddOptionalFieldsToSchema(schema map[string]*schema.Schema, keys ...string) {
	FixDatasourceSchemaFlags(schema, false, keys...)
}

// FindExact finds the first element in 'slice' matching the condition defined by 'finder'.
// It returns the first matching element and an error if either no match is found or multiple matches are found.
func FindExact[T any](slice []T, finder func(T) bool, searchName string) (T, error) { //nolint:ireturn
	var found T
	var foundFlag bool

	for _, elem := range slice {
		if finder(elem) {
			if foundFlag {
				// More than one element found with the same search name
				var zero T
				return zero, fmt.Errorf("multiple elements found with the name %s", searchName)
			}
			found = elem
			foundFlag = true
		}
	}

	if !foundFlag {
		var zero T
		return zero, fmt.Errorf("no element found with the name %s", searchName)
	}

	return found, nil
}

// SingularDataSourceFindError returns a standard error message for a singular data source's non-nil resource find error.
func SingularDataSourceFindError(resourceType string, err error) error {
	if errs.NotFound(err) {
		if errors.Is(err, &errs.TooManyResultsError{}) {
			return fmt.Errorf("multiple %[1]ss matched; use additional constraints to reduce matches to a single %[1]s", resourceType)
		}

		return fmt.Errorf("no matching %[1]s found", resourceType)
	}

	return fmt.Errorf("reading %s: %w", resourceType, err)
}
