package domain

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	domain "github.com/scaleway/scaleway-sdk-go/api/domain/v2beta1"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/datasource"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/types"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/verify"
)

func DataSourceScalewayDomainRecord() *schema.Resource {
	// Generate datasource schema from resource
	dsSchema := datasource.DatasourceSchemaFromResourceSchema(ResourceScalewayDomainRecord().Schema)

	// Set 'Optional' schema elements
	datasource.AddOptionalFieldsToSchema(dsSchema, "dns_zone", "name", "type", "data", "project_id")

	dsSchema["name"].ConflictsWith = []string{"record_id"}
	dsSchema["type"].ConflictsWith = []string{"record_id"}
	dsSchema["data"].ConflictsWith = []string{"record_id"}
	dsSchema["record_id"] = &schema.Schema{
		Type:          schema.TypeString,
		Optional:      true,
		Description:   "The ID of the record",
		ValidateFunc:  verify.UUID(),
		ConflictsWith: []string{"name", "type", "data"},
	}

	return &schema.Resource{
		ReadContext: DataSourceScalewayDomainRecordRead,
		Schema:      dsSchema,
	}
}

func DataSourceScalewayDomainRecordRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	domainAPI := newDomainAPI(meta)

	recordID, ok := d.GetOk("record_id")
	if !ok { // Get Record by dns_zone, name, type and data.
		res, err := domainAPI.ListDNSZoneRecords(&domain.ListDNSZoneRecordsRequest{
			DNSZone:   d.Get("dns_zone").(string),
			Name:      d.Get("name").(string),
			Type:      domain.RecordType(d.Get("type").(string)),
			ProjectID: types.ExpandStringPtr(d.Get("project_id")),
		}, scw.WithContext(ctx), scw.WithAllPages())
		if err != nil {
			return diag.FromErr(err)
		}
		if len(res.Records) == 0 {
			return diag.FromErr(fmt.Errorf("no record found with the type %s", d.Get("type")))
		}
		var record *domain.Record
		for i := range res.Records {
			if res.Records[i].Data == d.Get("data").(string) {
				if record != nil {
					return diag.FromErr(fmt.Errorf("more than one record found with this name: %s, type: %s and data: %s", d.Get("name"), d.Get("type"), d.Get("data")))
				}
				record = res.Records[i]
			}
		}
		if record == nil {
			return diag.FromErr(fmt.Errorf("no record found with the type this name: %s, type: %s and data: %s", d.Get("name"), d.Get("type"), d.Get("data")))
		}
		recordID = record.ID
	}

	d.SetId(fmt.Sprintf("%s/%s", d.Get("dns_zone"), recordID.(string)))
	return ResourceScalewayDomainRecordRead(ctx, d, meta)
}
