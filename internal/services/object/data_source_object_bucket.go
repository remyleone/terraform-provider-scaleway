package object

import (
	"context"
	"fmt"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/datasource"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DataSourceScalewayObjectBucket() *schema.Resource {
	// Generate datasource schema from resource
	dsSchema := datasource.DatasourceSchemaFromResourceSchema(ResourceScalewayObjectBucket().Schema)

	// Set 'Optional' schema elements
	datasource.AddOptionalFieldsToSchema(dsSchema, "name", "region", "project_id")

	return &schema.Resource{
		ReadContext: DataSourceScalewayObjectStorageRead,
		Schema:      dsSchema,
	}
}

func DataSourceScalewayObjectStorageRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	s3Client, region, err := S3ClientWithRegion(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	regionalID := locality.ExpandRegionalID(d.Get("name"))
	bucket := regionalID.ID
	bucketRegion := regionalID.Region

	if bucketRegion != "" && bucketRegion != region {
		s3Client, err = S3ClientForceRegion(d, meta, bucketRegion.String())
		if err != nil {
			return diag.FromErr(err)
		}
		region = bucketRegion
	}

	input := &s3.HeadBucketInput{
		Bucket: aws.String(bucket),
	}

	log.Printf("[DEBUG] Reading Object Storage bucket: %s", input)
	_, err = s3Client.HeadBucketWithContext(ctx, input)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed getting Object Storage bucket (%s): %w", bucket, err))
	}

	acl, err := s3Client.GetBucketAclWithContext(ctx, &s3.GetBucketAclInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		return diag.FromErr(fmt.Errorf("couldn't read bucket acl: %s", err))
	}
	_ = d.Set("project_id", normalizeOwnerID(acl.Owner.ID))

	bucketRegionalID := locality.NewRegionalIDString(region, bucket)
	d.SetId(bucketRegionalID)
	return ResourceScalewayObjectBucketRead(ctx, d, meta)
}
