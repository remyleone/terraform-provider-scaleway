package scaleway

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/locality"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/transport"
	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/verify"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/meta"

	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/locality/zonal"

	"github.com/scaleway/terraform-provider-scaleway/v2/scaleway/types"

	"github.com/hashicorp/aws-sdk-go-base/tfawserr"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

// Service information constants
const (
	ServiceName = "scw"       // Name of service.
	EndpointsID = ServiceName // ID to look up a service endpoint with.
)

// RegionalID represents an ID that is linked with a region, eg fr-par/11111111-1111-1111-1111-111111111111
type RegionalID struct {
	ID     string
	Region scw.Region
}

func (z RegionalID) String() string {
	return fmt.Sprintf("%s/%s", z.Region, z.ID)
}

func newRegionalID(region scw.Region, id string) RegionalID {
	return RegionalID{
		ID:     id,
		Region: region,
	}
}

func expandRegionalID(id interface{}) RegionalID {
	regionalID := RegionalID{}
	tab := strings.Split(id.(string), "/")
	if len(tab) != 2 {
		regionalID.ID = id.(string)
	} else {
		region, _ := scw.ParseRegion(tab[0])
		regionalID.ID = tab[1]
		regionalID.Region = region
	}

	return regionalID
}

// ZonedID represents an ID that is linked with a zone, eg fr-par-1/11111111-1111-1111-1111-111111111111
type ZonedID struct {
	ID   string
	Zone scw.Zone
}

func (z ZonedID) String() string {
	return fmt.Sprintf("%s/%s", z.Zone, z.ID)
}

func newZonedID(zone scw.Zone, id string) ZonedID {
	return ZonedID{
		ID:   id,
		Zone: zone,
	}
}

func expandZonedID(id interface{}) ZonedID {
	zonedID := ZonedID{}
	tab := strings.Split(id.(string), "/")
	if len(tab) != 2 {
		zonedID.ID = id.(string)
	} else {
		zone, _ := scw.ParseZone(tab[0])
		zonedID.ID = tab[1]
		zonedID.Zone = zone
	}

	return zonedID
}

// parseZonedNestedID parses a zonedNestedID and extracts the resource zone ,inner and outer ID.
func parseZonedNestedID(zonedNestedID string) (zone scw.Zone, outerID, innerID string, err error) {
	locality, innerID, outerID, err := locality.ParseLocalizedNestedID(zonedNestedID)
	if err != nil {
		return
	}

	zone, err = scw.ParseZone(locality)
	return
}

// parseRegionalNestedID parses a regionalNestedID and extracts the resource region, inner and outer ID.
func parseRegionalNestedID(regionalNestedID string) (region scw.Region, outerID, innerID string, err error) {
	locality, innerID, outerID, err := locality.ParseLocalizedNestedID(regionalNestedID)
	if err != nil {
		return
	}

	region, err = scw.ParseRegion(locality)
	return
}

// NewZonedNestedIDString constructs a unique identifier based on resource zone, inner and outer IDs
func NewZonedNestedIDString(zone scw.Zone, outerID, innerID string) string {
	return fmt.Sprintf("%s/%s/%s", zone, outerID, innerID)
}

// newRegionalIDString constructs a unique identifier based on resource region and id
func newRegionalIDString(region scw.Region, id string) string {
	return fmt.Sprintf("%s/%s", region, id)
}

// ErrRegionNotFound is returned when no region can be detected
var ErrRegionNotFound = errors.New("could not detect region")

// extractRegion will try to guess the region from the following:
//   - region field of the resource data
//   - default region from config
func extractRegion(d meta.TerraformResourceData, meta *meta.Meta) (scw.Region, error) {
	rawRegion, exist := d.GetOk("region")
	if exist {
		return scw.ParseRegion(rawRegion.(string))
	}

	region, exist := meta.GetScwClient().GetDefaultRegion()
	if exist {
		return region, nil
	}

	return "", ErrRegionNotFound
}

// extractRegion will try to guess the region from the following:
//   - region field of the resource data
//   - default region given in argument
//   - default region from config
func extractRegionWithDefault(d meta.TerraformResourceData, meta *meta.Meta, defaultRegion scw.Region) (scw.Region, error) {
	rawRegion, exist := d.GetOk("region")
	if exist {
		return scw.ParseRegion(rawRegion.(string))
	}

	if defaultRegion != "" {
		return defaultRegion, nil
	}

	region, exist := meta.GetScwClient().GetDefaultRegion()
	if exist {
		return region, nil
	}

	return "", ErrRegionNotFound
}

// ErrProjectIDNotFound is returned when no region can be detected
var ErrProjectIDNotFound = errors.New("could not detect project id")

// extractProjectID will try to guess the project id from the following:
//   - project_id field of the resource data
//   - default project id from config
func extractProjectID(d meta.TerraformResourceData, meta *meta.Meta) (projectID string, isDefault bool, err error) {
	rawProjectID, exist := d.GetOk("project_id")
	if exist {
		return rawProjectID.(string), false, nil
	}

	defaultProjectID, exist := meta.GetScwClient().GetDefaultProjectID()
	if exist {
		return defaultProjectID, true, nil
	}

	return "", false, ErrProjectIDNotFound
}

func allRegions() []string {
	regions := make([]string, 0, len(scw.AllRegions))
	for _, z := range scw.AllRegions {
		regions = append(regions, z.String())
	}

	return regions
}

// regionSchema returns a standard schema for a zone
func regionSchema() *schema.Schema {
	return &schema.Schema{
		Type:             schema.TypeString,
		Description:      "The region you want to attach the resource to",
		Optional:         true,
		ForceNew:         true,
		Computed:         true,
		ValidateDiagFunc: verify.StringInSliceWithWarning(allRegions(), "region"),
	}
}

// zoneComputedSchema returns a standard schema for a zone
func zoneComputedSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeString,
		Description: "The zone of the resource",
		Computed:    true,
	}
}

// regionComputedSchema returns a standard schema for a region
func regionComputedSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeString,
		Description: "The region of the resource",
		Computed:    true,
	}
}

const gb uint64 = 1000 * 1000 * 1000

func flattenTime(date *time.Time) interface{} {
	if date != nil {
		return date.Format(time.RFC3339)
	}
	return ""
}

func flattenDuration(duration *time.Duration) interface{} {
	if duration != nil {
		return duration.String()
	}
	return ""
}

func expandDuration(data interface{}) (*time.Duration, error) {
	if data == nil || data == "" {
		return nil, nil
	}
	d, err := time.ParseDuration(data.(string))
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func expandStringWithDefault(data interface{}, defaultValue string) string {
	if data == nil || data.(string) == "" {
		return defaultValue
	}
	return data.(string)
}

func expandStrings(data interface{}) []string {
	stringSlice := make([]string, 0, len(data.([]interface{})))
	for _, s := range data.([]interface{}) {
		// zero-value is nil, ["foo", ""]
		if s == nil {
			s = ""
		}
		stringSlice = append(stringSlice, s.(string))
	}
	return stringSlice
}

func expandStringsPtr(data interface{}) *[]string {
	stringSlice := make([]string, 0, len(data.([]interface{})))
	if _, ok := data.([]interface{}); !ok || data == nil {
		return nil
	}
	for _, s := range data.([]interface{}) {
		// zero-value is nil, ["foo", ""]
		if s == nil {
			s = ""
		}
		stringSlice = append(stringSlice, s.(string))
	}
	if len(stringSlice) == 0 {
		return nil
	}

	return &stringSlice
}

// expandUpdatedStringsPtr expands a string slice but will default to an empty list.
// Should be used on schema update so emptying a list will update resource.
func expandUpdatedStringsPtr(data interface{}) *[]string {
	stringSlice := []string{}
	if _, ok := data.([]interface{}); !ok || data == nil {
		return &stringSlice
	}
	for _, s := range data.([]interface{}) {
		// zero-value is nil, ["foo", ""]
		if s == nil {
			s = ""
		}
		stringSlice = append(stringSlice, s.(string))
	}
	return &stringSlice
}

func expandSliceIDsPtr(rawIDs interface{}) *[]string {
	stringSlice := make([]string, 0, len(rawIDs.([]interface{})))
	if _, ok := rawIDs.([]interface{}); !ok || rawIDs == nil {
		return &stringSlice
	}
	for _, s := range rawIDs.([]interface{}) {
		stringSlice = append(stringSlice, locality.ExpandID(s.(string)))
	}
	return &stringSlice
}

func expandStringsOrEmpty(data interface{}) []string {
	stringSlice := make([]string, 0, len(data.([]interface{})))
	if _, ok := data.([]interface{}); !ok || data == nil {
		return stringSlice
	}
	for _, s := range data.([]interface{}) {
		stringSlice = append(stringSlice, s.(string))
	}
	return stringSlice
}

func expandSliceStringPtr(data interface{}) []*string {
	if data == nil {
		return nil
	}
	stringSlice := []*string(nil)
	for _, s := range data.([]interface{}) {
		stringSlice = append(stringSlice, types.ExpandStringPtr(s))
	}
	return stringSlice
}

func flattenIPPtr(ip *net.IP) interface{} {
	if ip == nil {
		return ""
	}
	return ip.String()
}

func flattenStringPtr(s *string) interface{} {
	if s == nil {
		return ""
	}
	return *s
}

func flattenSliceStringPtr(s []*string) interface{} {
	res := make([]interface{}, 0, len(s))
	for _, strPtr := range s {
		res = append(res, flattenStringPtr(strPtr))
	}
	return res
}

func flattenSliceString(s []string) interface{} {
	res := make([]interface{}, 0, len(s))
	for _, strPtr := range s {
		res = append(res, strPtr)
	}
	return res
}

func flattenSliceIDs(certificates []string, zone scw.Zone) interface{} {
	res := []interface{}(nil)
	for _, certificateID := range certificates {
		res = append(res, zonal.NewZonedIDString(zone, certificateID))
	}

	return res
}

func flattenBoolPtr(b *bool) interface{} {
	if b == nil {
		return nil
	}
	return *b
}

func expandUpdatedStringPtr(data interface{}) *string {
	str := ""
	if data != nil {
		str = data.(string)
	}
	return &str
}

func expandBoolPtr(data interface{}) *bool {
	if data == nil {
		return nil
	}
	return scw.BoolPtr(data.(bool))
}

func flattenInt32Ptr(i *int32) interface{} {
	if i == nil {
		return 0
	}
	return *i
}

func flattenUint32Ptr(i *uint32) interface{} {
	if i == nil {
		return 0
	}
	return *i
}

func expandInt32Ptr(data interface{}) *int32 {
	if data == nil || data == "" {
		return nil
	}
	return scw.Int32Ptr(int32(data.(int)))
}

func expandUint32Ptr(data interface{}) *uint32 {
	if data == nil || data == "" {
		return nil
	}
	return scw.Uint32Ptr(uint32(data.(int)))
}

func expandIPNet(raw string) (scw.IPNet, error) {
	if raw == "" {
		return scw.IPNet{}, nil
	}
	var ipNet scw.IPNet
	err := json.Unmarshal([]byte(strconv.Quote(raw)), &ipNet)
	if err != nil {
		return scw.IPNet{}, fmt.Errorf("%s could not be marshaled: %v", raw, err)
	}

	return ipNet, nil
}

func flattenIPNet(ipNet scw.IPNet) (string, error) {
	raw, err := json.Marshal(ipNet)
	if err != nil {
		return "", err
	}
	return string(raw[1 : len(raw)-1]), nil // remove quotes
}

func validateDuration() schema.SchemaValidateFunc {
	return func(i interface{}, _ string) (strings []string, errors []error) {
		str, isStr := i.(string)
		if !isStr {
			return nil, []error{fmt.Errorf("%v is not a string", i)}
		}
		_, err := time.ParseDuration(str)
		if err != nil {
			return nil, []error{fmt.Errorf("cannot parse duration for value %s", str)}
		}
		return nil, nil
	}
}

func flattenMap(m map[string]string) interface{} {
	if m == nil {
		return nil
	}
	flattenedMap := make(map[string]interface{})
	for k, v := range m {
		flattenedMap[k] = v
	}
	return flattenedMap
}

func flattenMapStringStringPtr(m map[string]*string) interface{} {
	if m == nil {
		return nil
	}
	flattenedMap := make(map[string]interface{})
	for k, v := range m {
		if v != nil {
			flattenedMap[k] = *v
		} else {
			flattenedMap[k] = ""
		}
	}
	return flattenedMap
}

func diffSuppressFuncDuration(_, oldValue, newValue string, _ *schema.ResourceData) bool {
	if oldValue == newValue {
		return true
	}
	d1, err1 := time.ParseDuration(oldValue)
	d2, err2 := time.ParseDuration(newValue)
	if err1 != nil || err2 != nil {
		return false
	}
	return d1 == d2
}

func diffSuppressFuncTimeRFC3339(_, oldValue, newValue string, _ *schema.ResourceData) bool {
	if oldValue == newValue {
		return true
	}
	t1, err1 := time.Parse(time.RFC3339, oldValue)
	t2, err2 := time.Parse(time.RFC3339, newValue)
	if err1 != nil || err2 != nil {
		return false
	}
	return t1.Equal(t2)
}

func diffSuppressFuncIgnoreCase(_, oldValue, newValue string, _ *schema.ResourceData) bool {
	return strings.EqualFold(oldValue, newValue)
}

func diffSuppressFuncIgnoreCaseAndHyphen(_, oldValue, newValue string, _ *schema.ResourceData) bool {
	return strings.ReplaceAll(strings.ToLower(oldValue), "-", "_") == strings.ReplaceAll(strings.ToLower(newValue), "-", "_")
}

// diffSuppressFuncLocality is a SuppressDiffFunc to remove the locality from an ID when checking diff.
// e.g. 2c1a1716-5570-4668-a50a-860c90beabf6 == fr-par-1/2c1a1716-5570-4668-a50a-860c90beabf6
func diffSuppressFuncLocality(_, oldValue, newValue string, _ *schema.ResourceData) bool {
	return locality.ExpandID(oldValue) == locality.ExpandID(newValue)
}

// TimedOut returns true if the error represents a "wait timed out" condition.
// Specifically, TimedOut returns true if the error matches all these conditions:
//   - err is of type resource.TimeoutError
//   - TimeoutError.LastError is nil
func TimedOut(err error) bool {
	// This explicitly does *not* match wrapped TimeoutErrors
	timeoutErr, ok := err.(*resource.TimeoutError) //nolint:errorlint // Explicitly does *not* match wrapped TimeoutErrors
	return ok && timeoutErr.LastError == nil
}

func expandMapPtrStringString(data interface{}) *map[string]string {
	if data == nil {
		return nil
	}
	m := make(map[string]string)
	for k, v := range data.(map[string]interface{}) {
		m[k] = v.(string)
	}
	return &m
}

func expandMapStringStringPtr(data interface{}) map[string]*string {
	if data == nil {
		return nil
	}
	m := make(map[string]*string)
	for k, v := range data.(map[string]interface{}) {
		m[k] = types.ExpandStringPtr(v)
	}
	return m
}

func expandMapStringString(data any) map[string]string {
	if data == nil {
		return nil
	}
	m := make(map[string]string)
	for k, v := range data.(map[string]interface{}) {
		m[k] = v.(string)
	}
	return m
}

func errorCheck(err error, message string) bool {
	return strings.Contains(err.Error(), message)
}

// ErrCodeEquals returns true if the error matches all these conditions:
//   - err is of type scw.Error
//   - Error.Error() equals one of the passed codes
func ErrCodeEquals(err error, codes ...string) bool {
	var scwErr scw.SdkError
	if errors.As(err, &scwErr) {
		for _, code := range codes {
			if scwErr.Error() == code {
				return true
			}
		}
	}
	return false
}

func getBool(d *schema.ResourceData, key string) interface{} {
	val, ok := d.GetOkExists(key)
	if !ok {
		return nil
	}
	return val
}

// validateDate will validate that field is a valid ISO 8601
// It is the same as RFC3339
func validateDate() schema.SchemaValidateDiagFunc {
	return func(i interface{}, _ cty.Path) diag.Diagnostics {
		date, isStr := i.(string)
		if !isStr {
			return diag.Errorf("%v is not a string", date)
		}
		_, err := time.Parse(time.RFC3339, date)
		if err != nil {
			return diag.FromErr(err)
		}
		return nil
	}
}

func flattenSize(size *scw.Size) interface{} {
	if size == nil {
		return 0
	}
	return *size
}

type ServiceErrorCheckFunc func(*testing.T) resource.ErrorCheckFunc

var serviceErrorCheckFunc map[string]ServiceErrorCheckFunc

func ErrorCheck(t *testing.T, endpointIDs ...string) resource.ErrorCheckFunc {
	t.Helper()
	return func(err error) error {
		if err == nil {
			return nil
		}

		for _, endpointID := range endpointIDs {
			if f, ok := serviceErrorCheckFunc[endpointID]; ok {
				ef := f(t)
				err = ef(err)
			}

			if err == nil {
				break
			}
		}

		return err
	}
}

func validateMapKeyLowerCase() schema.SchemaValidateDiagFunc {
	return func(i interface{}, _ cty.Path) diag.Diagnostics {
		m := expandMapStringStringPtr(i)
		for k := range m {
			if strings.ToLower(k) != k {
				return diag.Diagnostics{diag.Diagnostic{
					Severity:      diag.Error,
					AttributePath: cty.IndexStringPath(k),
					Summary:       "Invalid map content",
					Detail:        fmt.Sprintf("key (%s) should be lowercase", k),
				}}
			}
		}
		return nil
	}
}

func retryOnTransientStateError[T any, U any](action func() (T, error), waiter func() (U, error)) (T, error) { //nolint:ireturn
	t, err := action()
	var transientStateError *scw.TransientStateError
	if errors.As(err, &transientStateError) {
		_, err := waiter()
		if err != nil {
			return t, err
		}
		return retryOnTransientStateError(action, waiter)
	}
	return t, err
}

// compareLocalities compare two localities
// They are equal if they are the same or if one is a zone contained in a region
func compareLocalities(loc1, loc2 string) bool {
	if loc1 == loc2 {
		return true
	}
	if strings.HasPrefix(loc1, loc2) || strings.HasPrefix(loc2, loc1) {
		return true
	}
	return false
}

// expandListKeys return the list of keys for an attribute in a list
// example for private-networks.#.id in a list of size 2
// will return private-networks.0.id and private-networks.1.id
// additional_volume_ids.#
// will return additional_volume_ids.0 and additional_volume_ids.1
func expandListKeys(key string, diff *schema.ResourceDiff) []string {
	addr := strings.Split(key, ".")
	// index of # in the addr
	index := 0

	for i := range addr {
		if addr[i] == "#" {
			index = i
		}
	}

	// get attribute.#
	listKey := key[:strings.Index(key, "#")+1]
	listLength := diff.Get(listKey).(int)

	keys := make([]string, 0, listLength)

	for i := 0; i < listLength; i++ {
		addr[index] = strconv.FormatInt(int64(i), 10)
		keys = append(keys, strings.Join(addr, "."))
	}

	return keys
}

// getLocality find the locality of a resource
// Will try to get the zone if available then use region
// Will also use default zone or region if available
func getLocality(diff *schema.ResourceDiff, meta *meta.Meta) string {
	var locality string

	rawStateType := diff.GetRawState().Type()

	if rawStateType.HasAttribute("zone") {
		zone, _ := zonal.ExtractZone(diff, meta)
		locality = zone.String()
	} else if rawStateType.HasAttribute("region") {
		region, _ := extractRegion(diff, meta)
		locality = region.String()
	}
	return locality
}

// customizeDiffLocalityCheck create a function that will validate locality IDs stored in given keys
// This locality IDs should have the same locality as the resource
// It will search for zone or region in resource.
// Should not be used on computed keys, if a computed key is going to change on zone/region change
// this function will still block the terraform plan
func customizeDiffLocalityCheck(keys ...string) schema.CustomizeDiffFunc {
	return func(_ context.Context, diff *schema.ResourceDiff, i interface{}) error {
		parsedLocality := getLocality(diff, i.(*meta.Meta))

		if parsedLocality == "" {
			return errors.New("missing locality zone or region to check IDs")
		}

		for _, key := range keys {
			// Handle values in lists
			if strings.Contains(key, "#") {
				listKeys := expandListKeys(key, diff)

				for _, listKey := range listKeys {
					IDLocality, _, err := locality.ParseLocalizedID(diff.Get(listKey).(string))
					if err == nil && !compareLocalities(IDLocality, parsedLocality) {
						return fmt.Errorf("given %s %s has different locality than the resource %q", listKey, diff.Get(listKey), parsedLocality)
					}
				}
			} else {
				IDLocality, _, err := locality.ParseLocalizedID(diff.Get(key).(string))
				if err == nil && !compareLocalities(IDLocality, parsedLocality) {
					return fmt.Errorf("given %s %s has different locality than the resource %q", key, diff.Get(key), parsedLocality)
				}
			}
		}
		return nil
	}
}

type TooManyResultsError struct {
	Count       int
	LastRequest interface{}
}

func (e *TooManyResultsError) Error() string {
	return fmt.Sprintf("too many results: wanted 1, got %d", e.Count)
}

func (e *TooManyResultsError) Is(err error) bool {
	_, ok := err.(*TooManyResultsError) //nolint:errorlint // Explicitly does *not* match down the error tree
	return ok
}

func (e *TooManyResultsError) As(target interface{}) bool {
	t, ok := target.(**retry.NotFoundError)
	if !ok {
		return false
	}

	*t = &retry.NotFoundError{
		Message:     e.Error(),
		LastRequest: e.LastRequest,
	}

	return true
}

var ErrTooManyResults = &TooManyResultsError{}

// SingularDataSourceFindError returns a standard error message for a singular data source's non-nil resource find error.
func SingularDataSourceFindError(resourceType string, err error) error {
	if NotFound(err) {
		if errors.Is(err, &TooManyResultsError{}) {
			return fmt.Errorf("multiple %[1]ss matched; use additional constraints to reduce matches to a single %[1]s", resourceType)
		}

		return fmt.Errorf("no matching %[1]s found", resourceType)
	}

	return fmt.Errorf("reading %s: %w", resourceType, err)
}

// NotFound returns true if the error represents a "resource not found" condition.
// Specifically, NotFound returns true if the error or a wrapped error is of type
// retry.NotFoundError.
func NotFound(err error) bool {
	var e *retry.NotFoundError // nosemgrep:ci.is-not-found-error
	return errors.As(err, &e)
}

type RetryWhenConfig[T any] struct {
	Timeout  time.Duration
	Interval time.Duration
	Function func() (T, error)
}

var ErrRetryWhenTimeout = errors.New("timeout reached")

// retryWhen executes the function passed in the configuration object until the timeout is reached or the context is cancelled.
// It will retry if the shouldRetry function returns true. It will stop if the shouldRetry function returns false.
func retryWhen[T any](ctx context.Context, config *RetryWhenConfig[T], shouldRetry func(error) bool) (T, error) { //nolint: ireturn
	retryInterval := config.Interval
	if transport.DefaultWaitRetryInterval != nil {
		retryInterval = *transport.DefaultWaitRetryInterval
	}

	timer := time.NewTimer(config.Timeout)

	for {
		result, err := config.Function()
		if shouldRetry(err) {
			select {
			case <-timer.C:
				return result, ErrRetryWhenTimeout
			case <-ctx.Done():
				return result, ctx.Err()
			default:
				time.Sleep(retryInterval) // lintignore:R018
				continue
			}
		}

		return result, err
	}
}

// retryWhenAWSErrCodeEquals retries a function when it returns a specific AWS error
func retryWhenAWSErrCodeEquals[T any](ctx context.Context, codes []string, config *RetryWhenConfig[T]) (T, error) { //nolint: ireturn
	return retryWhen(ctx, config, func(err error) bool {
		return tfawserr.ErrCodeEquals(err, codes...)
	})
}

// retryWhenAWSErrCodeNotEquals retries a function until it returns a specific AWS error
func retryWhenAWSErrCodeNotEquals[T any](ctx context.Context, codes []string, config *RetryWhenConfig[T]) (T, error) { //nolint: ireturn
	return retryWhen(ctx, config, func(err error) bool {
		if err == nil {
			return true
		}

		return !tfawserr.ErrCodeEquals(err, codes...)
	})
}

func sliceContainsString(slice []string, str string) bool {
	for _, v := range slice {
		if v == str {
			return true
		}
	}
	return false
}

// testAccCheckScalewayResourceIDPersisted checks that the ID of the resource is the same throughout tests of migration or mutation
// It can be used to check that no ForceNew has been done
func testAccCheckScalewayResourceIDPersisted(resourceName string, resourceID *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource was not found: %s", resourceName)
		}
		if *resourceID != "" && *resourceID != rs.Primary.ID {
			return errors.New("resource ID changed when it should have persisted")
		}
		*resourceID = rs.Primary.ID
		return nil
	}
}

// testAccCheckScalewayResourceIDChanged checks that the ID of the resource has indeed changed, in case of ForceNew for example.
// It will fail if resourceID is empty so be sure to use testAccCheckScalewayResourceIDPersisted first in a test suite.
func testAccCheckScalewayResourceIDChanged(resourceName string, resourceID *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if resourceID == nil || *resourceID == "" {
			return errors.New("resourceID was not set")
		}
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource was not found: %s", resourceName)
		}
		if *resourceID == rs.Primary.ID {
			return errors.New("resource ID persisted when it should have changed")
		}
		*resourceID = rs.Primary.ID
		return nil
	}
}

// testAccCheckScalewayResourceRawIDMatches asserts the equality of IDs from two specified attributes of two Scaleway resources.
func testAccCheckScalewayResourceRawIDMatches(res1, attr1, res2, attr2 string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs1, ok1 := s.RootModule().Resources[res1]
		if !ok1 {
			return fmt.Errorf("not found: %s", res1)
		}

		rs2, ok2 := s.RootModule().Resources[res2]
		if !ok2 {
			return fmt.Errorf("not found: %s", res2)
		}

		id1 := locality.ExpandID(rs1.Primary.Attributes[attr1])
		id2 := locality.ExpandID(rs2.Primary.Attributes[attr2])

		if id1 != id2 {
			return fmt.Errorf("ID mismatch: %s from resource %s does not match ID %s from resource %s", id1, res1, id2, res2)
		}

		return nil
	}
}

// findExact finds the first element in 'slice' matching the condition defined by 'finder'.
// It returns the first matching element and an error if either no match is found or multiple matches are found.
func findExact[T any](slice []T, finder func(T) bool, searchName string) (T, error) { //nolint:ireturn
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
