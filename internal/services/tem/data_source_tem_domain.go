package tem

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	tem "github.com/scaleway/scaleway-sdk-go/api/tem/v1alpha1"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/datasource"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/types"
)

func DataSourceScalewayTemDomain() *schema.Resource {
	// Generate datasource schema from resource
	dsSchema := datasource.DatasourceSchemaFromResourceSchema(ResourceScalewayTemDomain().Schema)

	// Set 'Optional' schema elements
	datasource.AddOptionalFieldsToSchema(dsSchema, "name", "region", "project_id")

	dsSchema["name"].ConflictsWith = []string{"domain_id"}
	dsSchema["domain_id"] = &schema.Schema{
		Type:          schema.TypeString,
		Optional:      true,
		Description:   "The ID of the tem domain",
		ValidateFunc:  locality.UUIDorUUIDWithLocality(),
		ConflictsWith: []string{"name"},
	}

	return &schema.Resource{
		ReadContext: DataSourceScalewayTemDomainRead,

		Schema: dsSchema,
	}
}

func DataSourceScalewayTemDomainRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api, region, err := TemAPIWithRegion(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	domainID, ok := d.GetOk("domain_id")
	if !ok {
		res, err := api.ListDomains(&tem.ListDomainsRequest{
			Region:    region,
			Name:      types.ExpandStringPtr(d.Get("name")),
			ProjectID: types.ExpandStringPtr(d.Get("project_id")),
		}, scw.WithContext(ctx))
		if err != nil {
			return diag.FromErr(err)
		}

		for _, domain := range res.Domains {
			if domain.Status == tem.DomainStatusRevoked {
				continue
			}

			if domain.Name == d.Get("name").(string) {
				if domainID != "" {
					return diag.FromErr(fmt.Errorf("more than 1 server found with the same name %s", d.Get("name")))
				}

				domainID = domain.ID
			}
		}

		if domainID == "" {
			return diag.FromErr(fmt.Errorf("no domain found with the name %s", d.Get("name")))
		}
	}

	regionalID := locality.DatasourceNewRegionalID(domainID, region)
	d.SetId(regionalID)
	err = d.Set("domain_id", regionalID)
	if err != nil {
		return diag.FromErr(err)
	}

	diags := ResourceScalewayTemDomainRead(ctx, d, meta)
	if diags != nil {
		return append(diags, diag.Errorf("failed to read tem domain state")...)
	}

	if d.Id() == "" {
		return diag.Errorf("tem domain (%s) not found", regionalID)
	}

	return nil
}
