package object

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/types"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/workerpool"
	"runtime"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	awspolicy "github.com/hashicorp/awspolicyequivalence"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

const (
	defaultObjectBucketTimeout = 10 * time.Minute

	maxObjectVersionDeletionWorkers = 8

	objectTestsMainRegion      = "nl-ams"
	objectTestsSecondaryRegion = "pl-waw"

	errCodeForbidden = "Forbidden"
)

func flattenObjectBucketTags(tagsSet []*s3.Tag) map[string]interface{} {
	tags := map[string]interface{}{}

	for _, tagSet := range tagsSet {
		var key string
		var value string
		if tagSet.Key != nil {
			key = *tagSet.Key
		}
		if tagSet.Value != nil {
			value = *tagSet.Value
		}
		tags[key] = value
	}

	return tags
}

func expandObjectBucketTags(tags interface{}) []*s3.Tag {
	tagsSet := []*s3.Tag(nil)
	for key, value := range tags.(map[string]interface{}) {
		tagsSet = append(tagsSet, &s3.Tag{
			Key:   scw.StringPtr(key),
			Value: types.ExpandStringPtr(value),
		})
	}

	return tagsSet
}

func objectBucketEndpointURL(bucketName string, region scw.Region) string {
	return fmt.Sprintf("https://%s.s3.%s.scw.cloud", bucketName, region)
}

func objectBucketAPIEndpointURL(region scw.Region) string {
	return fmt.Sprintf("https://s3.%s.scw.cloud", region)
}

// Returns true if the error matches all these conditions:
//   - err is of type aws err.Error
//   - Error.Code() matches code
//   - Error.Message() contains message
func isS3Err(err error, code string, message string) bool {
	var awsErr awserr.Error
	if errors.As(err, &awsErr) {
		return awsErr.Code() == code && strings.Contains(awsErr.Message(), message)
	}
	return false
}

func flattenObjectBucketVersioning(versioningResponse *s3.GetBucketVersioningOutput) []map[string]interface{} {
	vcl := []map[string]interface{}{{}}
	vcl[0]["enabled"] = versioningResponse.Status != nil && *versioningResponse.Status == s3.BucketVersioningStatusEnabled
	return vcl
}

func expandObjectBucketVersioning(v []interface{}) *s3.VersioningConfiguration {
	vc := &s3.VersioningConfiguration{}
	vc.Status = scw.StringPtr(s3.BucketVersioningStatusSuspended)
	if len(v) > 0 {
		if c := v[0].(map[string]interface{}); c["enabled"].(bool) {
			vc.Status = scw.StringPtr(s3.BucketVersioningStatusEnabled)
		}
	}
	return vc
}

func flattenBucketCORS(corsResponse interface{}) []map[string]interface{} {
	corsRules := make([]map[string]interface{}, 0)
	if cors, ok := corsResponse.(*s3.GetBucketCorsOutput); ok && len(cors.CORSRules) > 0 {
		corsRules = make([]map[string]interface{}, 0, len(cors.CORSRules))
		for _, ruleObject := range cors.CORSRules {
			rule := make(map[string]interface{})
			rule["allowed_headers"] = types.FlattenSliceStringPtr(ruleObject.AllowedHeaders)
			rule["allowed_methods"] = types.FlattenSliceStringPtr(ruleObject.AllowedMethods)
			rule["allowed_origins"] = types.FlattenSliceStringPtr(ruleObject.AllowedOrigins)
			// Both the "ExposeHeaders" and "MaxAgeSeconds" might not be set.
			if ruleObject.AllowedOrigins != nil {
				rule["expose_headers"] = types.FlattenSliceStringPtr(ruleObject.ExposeHeaders)
			}
			if ruleObject.MaxAgeSeconds != nil {
				rule["max_age_seconds"] = int(*ruleObject.MaxAgeSeconds)
			}
			corsRules = append(corsRules, rule)
		}
	}
	return corsRules
}

func expandBucketCORS(ctx context.Context, rawCors []interface{}, bucket string) []*s3.CORSRule {
	rules := make([]*s3.CORSRule, 0, len(rawCors))
	for _, cors := range rawCors {
		corsMap := cors.(map[string]interface{})
		r := &s3.CORSRule{}
		for k, v := range corsMap {
			tflog.Debug(ctx, fmt.Sprintf("S3 bucket: %s, put CORS: %#v, %#v", bucket, k, v))
			if k == "max_age_seconds" {
				r.MaxAgeSeconds = scw.Int64Ptr(int64(v.(int)))
			} else {
				vMap := make([]*string, len(v.([]interface{})))
				for i, vv := range v.([]interface{}) {
					if str, ok := vv.(string); ok {
						vMap[i] = scw.StringPtr(str)
					}
				}
				switch k {
				case "allowed_headers":
					r.AllowedHeaders = vMap
				case "allowed_methods":
					r.AllowedMethods = vMap
				case "allowed_origins":
					r.AllowedOrigins = vMap
				case "expose_headers":
					r.ExposeHeaders = vMap
				}
			}
		}
		rules = append(rules, r)
	}
	return rules
}

func deleteS3ObjectVersion(conn *s3.S3, bucketName string, key string, versionID string, force bool) error {
	input := &s3.DeleteObjectInput{
		Bucket: scw.StringPtr(bucketName),
		Key:    scw.StringPtr(key),
	}
	if versionID != "" {
		input.VersionId = scw.StringPtr(versionID)
	}
	if force {
		input.BypassGovernanceRetention = scw.BoolPtr(force)
	}

	_, err := conn.DeleteObject(input)
	return err
}

// removeS3ObjectVersionLegalHold remove legal hold from an ObjectVersion if it is on
// returns true if legal hold was removed
func removeS3ObjectVersionLegalHold(conn *s3.S3, bucketName string, objectVersion *s3.ObjectVersion) (bool, error) {
	objectHead, err := conn.HeadObject(&s3.HeadObjectInput{
		Bucket:    scw.StringPtr(bucketName),
		Key:       objectVersion.Key,
		VersionId: objectVersion.VersionId,
	})
	if err != nil {
		err = fmt.Errorf("failed to get S3 object meta data: %s", err)
		return false, err
	}
	if aws.StringValue(objectHead.ObjectLockLegalHoldStatus) != s3.ObjectLockLegalHoldStatusOn {
		return false, nil
	}
	_, err = conn.PutObjectLegalHold(&s3.PutObjectLegalHoldInput{
		Bucket:    scw.StringPtr(bucketName),
		Key:       objectVersion.Key,
		VersionId: objectVersion.VersionId,
		LegalHold: &s3.ObjectLockLegalHold{
			Status: scw.StringPtr(s3.ObjectLockLegalHoldStatusOff),
		},
	})
	if err != nil {
		err = fmt.Errorf("failed to put S3 object legal hold: %s", err)
		return false, err
	}
	return true, nil
}

func deleteS3ObjectVersions(ctx context.Context, conn *s3.S3, bucketName string, force bool) error {
	var globalErr error
	listInput := &s3.ListObjectVersionsInput{
		Bucket: scw.StringPtr(bucketName),
	}

	deletionWorkers := runtime.NumCPU()
	if deletionWorkers > maxObjectVersionDeletionWorkers {
		deletionWorkers = maxObjectVersionDeletionWorkers
	}

	listErr := conn.ListObjectVersionsPagesWithContext(ctx, listInput, func(page *s3.ListObjectVersionsOutput, _ bool) bool {
		pool := workerpool.NewWorkerPool(deletionWorkers)

		for _, objectVersion := range page.Versions {
			objectVersion := objectVersion

			pool.AddTask(func() error {
				objectKey := aws.StringValue(objectVersion.Key)
				objectVersionID := aws.StringValue(objectVersion.VersionId)
				err := deleteS3ObjectVersion(conn, bucketName, objectKey, objectVersionID, force)

				if isS3Err(err, ErrCodeAccessDenied, "") && force {
					legalHoldRemoved, errLegal := removeS3ObjectVersionLegalHold(conn, bucketName, objectVersion)
					if errLegal != nil {
						return fmt.Errorf("failed to remove legal hold: %s", errLegal)
					}

					if legalHoldRemoved {
						err = deleteS3ObjectVersion(conn, bucketName, objectKey, objectVersionID, force)
					}
				}

				if err != nil {
					return fmt.Errorf("failed to delete S3 object: %s", err)
				}

				return nil
			})
		}

		errs := pool.CloseAndWait()
		if len(errs) > 0 {
			globalErr = multierror.Append(nil, errs...)
			return false
		}

		return true
	})
	if listErr != nil {
		return fmt.Errorf("error listing S3 objects: %s", globalErr)
	}
	if globalErr != nil {
		return globalErr
	}

	listErr = conn.ListObjectVersionsPagesWithContext(ctx, listInput, func(page *s3.ListObjectVersionsOutput, _ bool) bool {
		pool := workerpool.NewWorkerPool(deletionWorkers)

		for _, deleteMarkerEntry := range page.DeleteMarkers {
			deleteMarkerEntry := deleteMarkerEntry

			pool.AddTask(func() error {
				deleteMarkerKey := aws.StringValue(deleteMarkerEntry.Key)
				deleteMarkerVersionsID := aws.StringValue(deleteMarkerEntry.VersionId)
				err := deleteS3ObjectVersion(conn, bucketName, deleteMarkerKey, deleteMarkerVersionsID, force)
				if err != nil {
					return fmt.Errorf("failed to delete S3 object delete marker: %s", err)
				}

				return nil
			})
		}

		errs := pool.CloseAndWait()
		if len(errs) > 0 {
			globalErr = multierror.Append(nil, errs...)
			return false
		}

		return true
	})
	if listErr != nil {
		return fmt.Errorf("error listing S3 objects for delete markers: %s", globalErr)
	}
	if globalErr != nil {
		return globalErr
	}

	return nil
}

func transitionHash(v interface{}) int {
	var buf bytes.Buffer
	m, ok := v.(map[string]interface{})

	if !ok {
		return 0
	}

	if v, ok := m["days"]; ok {
		buf.WriteString(fmt.Sprintf("%d-", v.(int)))
	}
	if v, ok := m["storage_class"]; ok {
		buf.WriteString(v.(string) + "-")
	}
	return types.StringHashcode(buf.String())
}

const (
	// TransitionStorageClassStandard is a TransitionStorageClass enum value
	TransitionStorageClassStandard = "STANDARD"

	// TransitionStorageClassGlacier is a TransitionStorageClass enum value
	TransitionStorageClassGlacier = "GLACIER"

	// TransitionStorageClassOnezoneIa is a TransitionStorageClass enum value
	TransitionStorageClassOnezoneIa = "ONEZONE_IA"
)

// TransitionSCWStorageClassValues returns all elements of the TransitionStorageClass enum supported by scaleway
func TransitionSCWStorageClassValues() []string {
	return []string{
		TransitionStorageClassStandard,
		TransitionStorageClassGlacier,
		TransitionStorageClassOnezoneIa,
	}
}

func SuppressEquivalentPolicyDiffs(k, old, newP string, _ *schema.ResourceData) bool {
	tflog.Debug(context.Background(),
		fmt.Sprintf("[DEBUG] suppress policy on key: %s, old: %s new: %s", k, old, newP))
	if strings.TrimSpace(old) == "" && strings.TrimSpace(newP) == "" {
		return true
	}

	if strings.TrimSpace(old) == "{}" && strings.TrimSpace(newP) == "" {
		return true
	}

	if strings.TrimSpace(old) == "" && strings.TrimSpace(newP) == "{}" {
		return true
	}

	if strings.TrimSpace(old) == "{}" && strings.TrimSpace(newP) == "{}" {
		return true
	}

	equivalent, err := awspolicy.PoliciesAreEquivalent(old, newP)
	if err != nil {
		return false
	}

	return equivalent
}

func SecondJSONUnlessEquivalent(old, newP string) (string, error) {
	// valid empty JSON is "{}" not "" so handle special case to avoid
	// Error unmarshalling policy: unexpected end of JSON input
	if strings.TrimSpace(newP) == "" {
		return "", nil
	}

	if strings.TrimSpace(newP) == "{}" {
		return "{}", nil
	}

	if strings.TrimSpace(old) == "" || strings.TrimSpace(old) == "{}" {
		return newP, nil
	}

	equivalent, err := awspolicy.PoliciesAreEquivalent(old, newP)
	if err != nil {
		return "", err
	}

	if equivalent {
		return old, nil
	}

	return newP, nil
}

type S3Website struct {
	Endpoint, Domain string
}

func WebsiteEndpoint(bucket string, region scw.Region) *S3Website {
	domain := WebsiteDomainURL(region.String())
	return &S3Website{Endpoint: fmt.Sprintf("%s.%s", bucket, domain), Domain: domain}
}

func WebsiteDomainURL(region string) string {
	// Different regions have different syntax for website endpoints
	// https://docs.aws.amazon.com/AmazonS3/latest/dev/WebsiteEndpoints.html
	// https://docs.aws.amazon.com/general/latest/gr/rande.html#s3_website_region_endpoints
	return fmt.Sprintf("s3-website.%s.scw.cloud", region)
}

func buildBucketOwnerID(id *string) *string {
	s := fmt.Sprintf("%[1]s:%[1]s", *id)
	return &s
}

func normalizeOwnerID(id *string) *string {
	tab := strings.Split(*id, ":")
	if len(tab) != 2 {
		return id
	}

	return &tab[0]
}

func addReadBucketErrorDiagnostic(diags *diag.Diagnostics, err error, resource string, awsResourceNotFoundCode string) (bucketFound bool, resourceFound bool) {
	switch {
	case isS3Err(err, s3.ErrCodeNoSuchBucket, ""):
		*diags = append(*diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  "Bucket not found",
			Detail:   "Got 404 error while reading bucket, removing from state",
		})
		return false, false

	case isS3Err(err, awsResourceNotFoundCode, ""):
		return true, false

	case isS3Err(err, ErrCodeAccessDenied, ""):
		d := diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  fmt.Sprintf("Cannot read bucket %s: Forbidden", resource),
			Detail:   fmt.Sprintf("Got 403 error while reading bucket %s, please check your IAM permissions and your bucket policy", resource),
		}

		attributes := map[string]string{
			"acl":                       "acl",
			"object lock configuration": "object_lock_enabled",
			"objects":                   "",
			"tags":                      "tags",
			"CORS configuration":        "cors_rule",
			"versioning":                "versioning",
			"lifecycle configuration":   "lifecycle_rule",
		}
		if attributeName, ok := attributes[resource]; ok {
			d.AttributePath = cty.GetAttrPath(attributeName)
		}

		*diags = append(*diags, d)
		return true, true

	default:
		*diags = append(*diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Errorf("couldn't read bucket %s: %w", resource, err).Error(),
		})
		return true, true
	}
}
