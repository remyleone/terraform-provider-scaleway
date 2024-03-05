package scaleway

import (
	"context"
	"fmt"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/object"

	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/project"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

func ResourceBucketWebsiteConfiguration() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceBucketWebsiteConfigurationCreate,
		ReadContext:   resourceBucketWebsiteConfigurationRead,
		UpdateContext: resourceBucketWebsiteConfigurationUpdate,
		DeleteContext: resourceBucketWebsiteConfigurationDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"bucket": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateFunc:     validation.StringLenBetween(1, 63),
				Description:      "The bucket's name or regional ID.",
				DiffSuppressFunc: diffSuppressFuncLocality,
			},
			"index_document": {
				Type:     schema.TypeList,
				Required: true,
				MinItems: 1,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"suffix": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
				Description: "The name of the index document for the website.",
			},
			"error_document": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"key": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
				Description: "The name of the error document for the website.",
			},
			"website_endpoint": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The domain of the website endpoint.",
			},
			"website_domain": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The website endpoint.",
			},
			"region":     regionSchema(),
			"project_id": project.ProjectIDSchema(),
		},
	}
}

func resourceBucketWebsiteConfigurationCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, region, err := s3ClientWithRegion(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}

	regionalID := expandRegionalID(d.Get("bucket"))
	bucket := regionalID.ID
	bucketRegion := regionalID.Region

	if bucketRegion != "" && bucketRegion != region {
		conn, err = s3ClientForceRegion(d, meta, bucketRegion.String())
		if err != nil {
			return diag.FromErr(err)
		}
		region = bucketRegion
	}

	websiteConfig := &s3.WebsiteConfiguration{
		IndexDocument: expandBucketWebsiteConfigurationIndexDocument(d.Get("index_document").([]interface{})),
	}

	if v, ok := d.GetOk("error_document"); ok && len(v.([]interface{})) > 0 && v.([]interface{})[0] != nil {
		websiteConfig.ErrorDocument = expandBucketWebsiteConfigurationErrorDocument(v.([]interface{}))
	}

	_, err = conn.ListObjectsWithContext(ctx, &s3.ListObjectsInput{
		Bucket: scw.StringPtr(bucket),
	})
	if err != nil {
		return diag.FromErr(fmt.Errorf("couldn't read bucket: %s", err))
	}

	input := &s3.PutBucketWebsiteInput{
		Bucket:               aws.String(bucket),
		WebsiteConfiguration: websiteConfig,
	}

	_, err = conn.PutBucketWebsiteWithContext(ctx, input)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error creating object bucket (%s) website configuration: %w", bucket, err))
	}

	d.SetId(newRegionalIDString(region, bucket))

	return resourceBucketWebsiteConfigurationRead(ctx, d, meta)
}

func resourceBucketWebsiteConfigurationRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, region, bucket, err := s3ClientWithRegionAndName(d, meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	input := &s3.GetBucketWebsiteInput{
		Bucket: aws.String(bucket),
	}

	// expectedBucketOwner and routing not supported

	_, err = conn.ListObjectsWithContext(ctx, &s3.ListObjectsInput{
		Bucket: scw.StringPtr(bucket),
	})
	if err != nil {
		if s3err, ok := err.(awserr.Error); ok && s3err.Code() == s3.ErrCodeNoSuchBucket {
			tflog.Error(ctx, fmt.Sprintf("Bucket %q was not found - removing from state!", bucket))
			d.SetId("")
			return nil
		}
		return diag.FromErr(fmt.Errorf("couldn't read bucket: %s", err))
	}

	output, err := conn.GetBucketWebsiteWithContext(ctx, input)
	if !d.IsNewResource() && ErrCodeEquals(err, s3.ErrCodeNoSuchBucket, object.ErrCodeNoSuchWebsiteConfiguration) {
		tflog.Debug(ctx, fmt.Sprintf("[WARN] Object Bucket Website Configuration (%s) not found, removing from state", d.Id()))
		d.SetId("")
		return nil
	}

	if output == nil {
		if d.IsNewResource() {
			return diag.FromErr(fmt.Errorf("error reading object bucket website configuration (%s): empty output", d.Id()))
		}
		tflog.Info(ctx, fmt.Sprintf("[WARN] object Bucket Website Configuration (%s) not found, removing from state", d.Id()))
		d.SetId("")
		return nil
	}

	_ = d.Set("bucket", bucket)
	_ = d.Set("index_document", flattenBucketWebsiteConfigurationIndexDocument(output.IndexDocument))

	if err := d.Set("error_document", flattenBucketWebsiteConfigurationErrorDocument(output.ErrorDocument)); err != nil {
		return diag.FromErr(fmt.Errorf("error setting error_document: %w", err))
	}

	websiteEndpoint := WebsiteEndpoint(bucket, region)

	if websiteEndpoint != nil {
		_ = d.Set("website_endpoint", websiteEndpoint.Endpoint)
		_ = d.Set("website_domain", websiteEndpoint.Domain)
	}

	acl, err := conn.GetBucketAclWithContext(ctx, &s3.GetBucketAclInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		return diag.FromErr(fmt.Errorf("couldn't read bucket acl: %s", err))
	}
	_ = d.Set("project_id", normalizeOwnerID(acl.Owner.ID))

	return nil
}

func resourceBucketWebsiteConfigurationUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, _, bucket, err := s3ClientWithRegionAndName(d, meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	websiteConfig := &s3.WebsiteConfiguration{
		IndexDocument: expandBucketWebsiteConfigurationIndexDocument(d.Get("index_document").([]interface{})),
	}

	if v, ok := d.GetOk("error_document"); ok && len(v.([]interface{})) > 0 && v.([]interface{})[0] != nil {
		websiteConfig.ErrorDocument = expandBucketWebsiteConfigurationErrorDocument(v.([]interface{}))
	}

	input := &s3.PutBucketWebsiteInput{
		Bucket:               aws.String(bucket),
		WebsiteConfiguration: websiteConfig,
	}

	_, err = conn.PutBucketWebsiteWithContext(ctx, input)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error updating Object bucket website configuration (%s): %w", d.Id(), err))
	}

	return resourceBucketWebsiteConfigurationRead(ctx, d, meta)
}

func resourceBucketWebsiteConfigurationDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, _, bucket, err := s3ClientWithRegionAndName(d, meta, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	input := &s3.DeleteBucketWebsiteInput{
		Bucket: aws.String(bucket),
	}

	_, err = conn.DeleteBucketWebsiteWithContext(ctx, input)

	if ErrCodeEquals(err, s3.ErrCodeNoSuchBucket, object.ErrCodeNoSuchWebsiteConfiguration) {
		return nil
	}

	if err != nil {
		return diag.FromErr(fmt.Errorf("error deleting Object bucket website configuration (%s): %w", d.Id(), err))
	}

	return nil
}

func expandBucketWebsiteConfigurationErrorDocument(l []interface{}) *s3.ErrorDocument {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	tfMap, ok := l[0].(map[string]interface{})
	if !ok {
		return nil
	}

	result := &s3.ErrorDocument{}

	if v, ok := tfMap["key"].(string); ok && v != "" {
		result.Key = aws.String(v)
	}

	return result
}

func expandBucketWebsiteConfigurationIndexDocument(l []interface{}) *s3.IndexDocument {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	tfMap, ok := l[0].(map[string]interface{})
	if !ok {
		return nil
	}

	result := &s3.IndexDocument{}

	if v, ok := tfMap["suffix"].(string); ok && v != "" {
		result.Suffix = aws.String(v)
	}

	return result
}

func flattenBucketWebsiteConfigurationIndexDocument(i *s3.IndexDocument) []interface{} {
	if i == nil {
		return []interface{}{}
	}

	m := make(map[string]interface{})

	if i.Suffix != nil {
		m["suffix"] = aws.StringValue(i.Suffix)
	}

	return []interface{}{m}
}

func flattenBucketWebsiteConfigurationErrorDocument(e *s3.ErrorDocument) []interface{} {
	if e == nil {
		return []interface{}{}
	}

	m := make(map[string]interface{})

	if e.Key != nil {
		m["key"] = aws.StringValue(e.Key)
	}

	return []interface{}{m}
}
