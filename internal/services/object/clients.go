package object

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/logging"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/locality"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/meta"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/project"
	"net/http"
	"os"
	"strings"
)

func NewS3Client(httpClient *http.Client, region, accessKey, secretKey string) (*s3.S3, error) {
	config := &aws.Config{}
	config.WithRegion(region)
	config.WithCredentials(credentials.NewStaticCredentials(accessKey, secretKey, ""))
	if ep := os.Getenv("SCW_S3_ENDPOINT"); ep != "" {
		config.WithEndpoint(ep)
	} else {
		config.WithEndpoint("https://s3." + region + ".scw.cloud")
	}
	config.WithHTTPClient(httpClient)
	if logging.IsDebugOrHigher() {
		config.WithLogLevel(aws.LogDebugWithHTTPBody)
	}

	s, err := session.NewSession(config)
	if err != nil {
		return nil, err
	}
	return s3.New(s), nil
}

func NewS3ClientFromMeta(meta *meta.Meta, region string) (*s3.S3, error) {
	accessKey, _ := meta.GetScwClient().GetAccessKey()
	secretKey, _ := meta.GetScwClient().GetSecretKey()

	projectID, _ := meta.GetScwClient().GetDefaultProjectID()
	if projectID != "" {
		accessKey = accessKeyWithProjectID(accessKey, projectID)
	}

	if region == "" {
		defaultRegion, _ := meta.GetScwClient().GetDefaultRegion()
		region = defaultRegion.String()
	}

	return NewS3Client(meta.GetHTTPClient(), region, accessKey, secretKey)
}

func S3ClientWithRegion(d *schema.ResourceData, m interface{}) (*s3.S3, scw.Region, error) {
	meta := m.(*meta.Meta)
	region, err := locality.ExtractRegion(d, meta)
	if err != nil {
		return nil, "", err
	}

	accessKey, _ := meta.GetScwClient().GetAccessKey()
	if projectID, _, err := project.ExtractProjectID(d, meta); err == nil {
		accessKey = accessKeyWithProjectID(accessKey, projectID)
	}
	secretKey, _ := meta.GetScwClient().GetSecretKey()

	s3Client, err := NewS3Client(meta.GetHTTPClient(), region.String(), accessKey, secretKey)
	if err != nil {
		return nil, "", err
	}

	return s3Client, region, err
}

func S3ClientWithRegionAndName(d *schema.ResourceData, m interface{}, id string) (*s3.S3, scw.Region, string, error) {
	meta := m.(*meta.Meta)
	region, name, err := locality.ParseRegionalID(id)
	if err != nil {
		return nil, "", "", err
	}

	parts := strings.Split(name, "@")
	if len(parts) > 2 {
		return nil, "", "", fmt.Errorf("invalid ID %q: expected ID in format <region>/<name>[@<project_id>]", id)
	}
	name = parts[0]

	d.SetId(fmt.Sprintf("%s/%s", region, name))

	accessKey, _ := meta.GetScwClient().GetAccessKey()
	secretKey, _ := meta.GetScwClient().GetSecretKey()

	if len(parts) == 2 {
		accessKey = accessKeyWithProjectID(accessKey, parts[1])
	} else {
		projectID, _, err := project.ExtractProjectID(d, meta)
		if err == nil {
			accessKey = accessKeyWithProjectID(accessKey, projectID)
		}
	}

	s3Client, err := NewS3Client(meta.GetHTTPClient(), region.String(), accessKey, secretKey)
	if err != nil {
		return nil, "", "", err
	}

	return s3Client, region, name, nil
}

func S3ClientWithRegionAndNestedName(d *schema.ResourceData, m interface{}, name string) (*s3.S3, scw.Region, string, string, error) {
	meta := m.(*meta.Meta)
	region, outerID, innerID, err := locality.ParseRegionalNestedID(name)
	if err != nil {
		return nil, "", "", "", err
	}

	accessKey, _ := meta.GetScwClient().GetAccessKey()
	if projectID, _, err := project.ExtractProjectID(d, meta); err == nil {
		accessKey = accessKeyWithProjectID(accessKey, projectID)
	}
	secretKey, _ := meta.GetScwClient().GetSecretKey()

	s3Client, err := NewS3Client(meta.GetHTTPClient(), region.String(), accessKey, secretKey)
	if err != nil {
		return nil, "", "", "", err
	}

	return s3Client, region, outerID, innerID, err
}

func S3ClientWithRegionWithNameACL(d *schema.ResourceData, m interface{}, name string) (*s3.S3, scw.Region, string, string, error) {
	meta := m.(*meta.Meta)
	region, name, outerID, err := locality.ParseLocalizedNestedOwnerID(name)
	if err != nil {
		return nil, "", name, "", err
	}

	accessKey, _ := meta.GetScwClient().GetAccessKey()
	if projectID, _, err := project.ExtractProjectID(d, meta); err == nil {
		accessKey = accessKeyWithProjectID(accessKey, projectID)
	}
	secretKey, _ := meta.GetScwClient().GetSecretKey()

	s3Client, err := NewS3Client(meta.GetHTTPClient(), region, accessKey, secretKey)
	if err != nil {
		return nil, "", "", "", err
	}
	return s3Client, scw.Region(region), name, outerID, err
}

func S3ClientForceRegion(d *schema.ResourceData, m interface{}, region string) (*s3.S3, error) {
	meta := m.(*meta.Meta)

	accessKey, _ := meta.GetScwClient().GetAccessKey()
	if projectID, _, err := project.ExtractProjectID(d, meta); err == nil {
		accessKey = accessKeyWithProjectID(accessKey, projectID)
	}
	secretKey, _ := meta.GetScwClient().GetSecretKey()

	s3Client, err := NewS3Client(meta.GetHTTPClient(), region, accessKey, secretKey)
	if err != nil {
		return nil, err
	}

	return s3Client, err
}

func accessKeyWithProjectID(accessKey string, projectID string) string {
	return accessKey + "@" + projectID
}
