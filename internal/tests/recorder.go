package tests

import (
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"github.com/dnaeon/go-vcr/cassette"
	"github.com/dnaeon/go-vcr/recorder"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/scaleway/scaleway-sdk-go/strcase"
	"github.com/scaleway/terraform-provider-scaleway/v2/internal/transport"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// UpdateCassettes will update all cassettes of a given test
var UpdateCassettes = flag.Bool("cassettes", os.Getenv("TF_UPDATE_CASSETTES") == "true", "Record Cassettes")

// getHTTPRecoder creates a new httpClient that records all HTTP requests in a cassette.
// This cassette is then replayed whenever tests are executed again. This means that once the
// requests are recorded in the cassette, no more real HTTP requests must be made to run the tests.
//
// It is important to add a `defer cleanup()` so the given cassette files are correctly
// closed and saved after the requests.
func getHTTPRecoder(t *testing.T, update bool) (client *http.Client, cleanup func(), err error) {
	t.Helper()
	recorderMode := recorder.ModeReplaying
	if update {
		recorderMode = recorder.ModeRecording
	}

	// Setup recorder and scw client
	r, err := recorder.NewAsMode(getTestFilePath(t, ".cassette"), recorderMode, nil)
	if err != nil {
		return nil, nil, err
	}

	// Add custom matcher for requests and cassettes
	r.SetMatcher(cassetteMatcher)

	// Add a filter which removes Authorization headers from all requests:
	r.AddFilter(func(i *cassette.Interaction) error {
		i.Request.Headers = i.Request.Headers.Clone()
		delete(i.Request.Headers, "x-auth-token")
		delete(i.Request.Headers, "X-Auth-Token")
		delete(i.Request.Headers, "Authorization")
		return nil
	})

	// Add a filter that will replace sensitive values with fixed values
	r.AddSaveFilter(cassetteSensitiveFieldsAnonymizer)

	retryOptions := transport.RetryableTransportOptions{}
	if !*UpdateCassettes {
		retryOptions.RetryWaitMax = scw.TimeDurationPtr(0)
	}

	return &http.Client{Transport: transport.NewRetryableTransportWithOptions(r, retryOptions)}, func() {
		require.NoError(t, r.Stop()) // Make sure recorder is stopped once done with it
	}, nil
}

// cassetteMatcher is a custom matcher that check equivalence of a played request against a recorded one
// It compares method, path and query but will remove unwanted values from query
func cassetteMatcher(actual *http.Request, expected cassette.Request) bool {
	expectedURL, _ := url.Parse(expected.URL)
	actualURL := actual.URL
	actualURLValues := actualURL.Query()
	expectedURLValues := expectedURL.Query()
	for _, query := range QueryMatcherIgnore {
		actualURLValues.Del(query)
		expectedURLValues.Del(query)
	}
	actualURL.RawQuery = actualURLValues.Encode()
	expectedURL.RawQuery = expectedURLValues.Encode()

	// Specific handling of s3 URLs
	// Url format is https://test-acc-scaleway-object-bucket-lifecycle-8445817190507446251.s3.fr-par.scw.cloud/?lifecycle=
	if strings.HasSuffix(actualURL.Host, "scw.cloud") {
		if !strings.HasSuffix(expectedURL.Host, "scw.cloud") {
			return false
		}
		actualS3Host := strings.Split(actualURL.Host, ".")
		expectedS3Host := strings.Split(expectedURL.Host, ".")

		if len(actualS3Host) >= 5 && len(expectedS3Host) >= 5 {
			// Host is bucket.s3.region.scw.cloud
			// it could be a host without bucket name (ex: function upload)
			actualBucket := actualS3Host[0]
			expectedBucket := expectedS3Host[0]

			// Remove random number at the end of the bucket name
			if strings.Contains(actualBucket, "-") {
				actualBucket = actualBucket[:strings.LastIndex(actualBucket, "-")]
			}
			if strings.Contains(expectedBucket, "-") {
				expectedBucket = expectedBucket[:strings.LastIndex(expectedBucket, "-")]
			}

			if actualBucket != expectedBucket {
				return false
			}
		}
	}

	return actual.Method == expected.Method &&
		actual.URL.Path == expectedURL.Path &&
		actualURL.RawQuery == expectedURL.RawQuery &&
		cassetteBodyMatcher(actual, expected)
}

func cassetteSensitiveFieldsAnonymizer(i *cassette.Interaction) error {
	var jsonBody map[string]interface{}
	err := json.Unmarshal([]byte(i.Response.Body), &jsonBody)
	if err != nil {
		//nolint:nilerr
		return nil
	}
	for key, value := range SensitiveFields {
		if _, ok := jsonBody[key]; ok {
			jsonBody[key] = value
		}
	}
	anonymizedBody, err := json.Marshal(jsonBody)
	if err != nil {
		return fmt.Errorf("failed to marshal anonymized body: %w", err)
	}
	i.Response.Body = string(anonymizedBody)
	return nil
}

// getTestFilePath returns a valid filename path based on the go test name and suffix. (Take care of non fs friendly char)
func getTestFilePath(t *testing.T, suffix string) string {
	t.Helper()
	specialChars := regexp.MustCompile(`[\\?%*:|"<>. ]`)

	// Replace nested tests separators.
	fileName := strings.ReplaceAll(t.Name(), "/", "-")

	fileName = strcase.ToBashArg(fileName)

	// Replace special characters.
	fileName = specialChars.ReplaceAllLiteralString(fileName, "") + suffix

	// Remove prefix to simplify
	fileName = strings.TrimPrefix(fileName, "test-acc-scaleway-")

	return filepath.Join(".", "testdata", fileName)
}

// SensitiveFields is a map with keys listing fields that should be anonymized
// value will be set in place of its old value
var SensitiveFields = map[string]interface{}{
	"secret_key": "00000000-0000-0000-0000-000000000000",
}

// QueryMatcherIgnore contains the list of query value that should be ignored when matching requests with cassettes
var QueryMatcherIgnore = []string{
	"organization_id",
}

// BodyMatcherIgnore contains the list of json body keys that should be ignored when matching requests with cassettes
var BodyMatcherIgnore = []string{
	"organization", // like organization_id but deprecated
	"organization_id",
	"project_id",
	"project", // like project_id but should be deprecated
}

func compareJSONFields(expected, actualI interface{}) bool {
	switch actual := actualI.(type) {
	case string:
		if _, isString := expected.(string); !isString {
			return false
		}
		return compareJSONFieldsStrings(expected.(string), actual)
	default:
		// Consider equality when not handled
		return true
	}
}

// Test Generated name has format: "{prefix}-{generated_number}
// example: test-acc-scaleway-project-3723338038624371236
func extractTestGeneratedNamePrefix(name string) string {
	// {prefix}-{generated}
	//         ^
	dashIndex := strings.LastIndex(name, "-")

	generated := name[dashIndex+1:]
	_, generatedToIntErr := strconv.ParseInt(generated, 10, 64)

	if dashIndex == -1 || generatedToIntErr != nil {
		// some are only {name}
		return name
	}

	// {prefix}
	return name[:dashIndex]
}

// Generated names have format: "tf-{prefix}-{generated1}-{generated2}"
// example: tf-sg-gifted-yonath
func extractGeneratedNamePrefix(name string) string {
	if strings.Count(name, "-") < 3 {
		return name
	}
	// tf-{prefix}-gifted-yonath
	name = strings.TrimPrefix(name, "tf-")

	// {prefix}-gifted-yonath
	//                ^
	dashIndex := strings.LastIndex(name, "-")
	name = name[:dashIndex]
	// {prefix}-gifted
	//         ^
	dashIndex = strings.LastIndex(name, "-")
	name = name[:dashIndex]
	return name
}

// compareJSONFieldsStrings compare two strings from request JSON bodies
// has special case when string are terraform generated names
func compareJSONFieldsStrings(expected, actual string) bool {
	expectedHandled := expected
	actualHandled := actual

	// Remove s3 url suffix to allow comparison
	if strings.HasSuffix(actual, ".s3-website.fr-par.scw.cloud") {
		actual = strings.TrimSuffix(actual, ".s3-website.fr-par.scw.cloud")
		expected = strings.TrimSuffix(expected, ".s3-website.fr-par.scw.cloud")
	}

	// Try to parse test generated name
	if strings.Contains(actual, "-") {
		expectedHandled = extractTestGeneratedNamePrefix(expected)
		actualHandled = extractTestGeneratedNamePrefix(actual)
	}

	// Try provider generated name
	if actualHandled == actual && strings.HasPrefix(actual, "tf-") {
		expectedHandled = extractGeneratedNamePrefix(expected)
		actualHandled = extractGeneratedNamePrefix(actual)
	}

	return expectedHandled == actualHandled
}

// compareJSONBodies compare two given maps that represent json bodies
// returns true if both json are equivalent
func compareJSONBodies(expected, actual map[string]interface{}) bool {
	// Check for each key in actual requests
	// Compare its value to cassette content if marshal-able to string
	for key := range actual {
		expectedValue, exists := expected[key]
		if !exists {
			// Actual request may contain a field that does not exist in cassette
			// New fields can appear in requests with new api features
			// We do not want to generate new cassettes for each new features
			continue
		}
		if !compareJSONFields(expectedValue, actual[key]) {
			return false
		}
	}

	for key := range expected {
		_, exists := actual[key]
		if !exists && expected[key] != nil {
			// Fails match if cassettes contains a field not in actual requests
			// Fields should not disappear from requests unless a sdk breaking change
			// We ignore if field is nil in cassette as it could be an old deprecated and unused field
			return false
		}
	}
	return true
}

// compareFormBodies compare two given url.Values
// returns true if both url.Values are equivalent
func compareFormBodies(expected, actual url.Values) bool {
	// Check for each key in actual requests
	// Compare its value to cassette content if marshal-able to string
	for key := range actual {
		expectedValue, exists := expected[key]
		if !exists {
			// Actual request may contain a field that does not exist in cassette
			// New fields can appear in requests with new api features
			// We do not want to generate new cassettes for each new features
			continue
		}
		if !compareJSONFields(expectedValue, actual[key]) {
			return false
		}
	}

	for key := range expected {
		_, exists := actual[key]
		if !exists && expected[key] != nil {
			// Fails match if cassettes contains a field not in actual requests
			// Fields should not disappear from requests unless a sdk breaking change
			// We ignore if field is nil in cassette as it could be an old deprecated and unused field
			return false
		}
	}

	return true
}

// cassetteMatcher is a custom matcher that will juste check equivalence of request bodies
func cassetteBodyMatcher(actualRequest *http.Request, cassetteRequest cassette.Request) bool {
	if actualRequest.Body == nil || actualRequest.ContentLength == 0 {
		if cassetteRequest.Body == "" {
			return true // Body match if both are empty
		} else if _, isFile := actualRequest.Body.(*os.File); isFile {
			return true // Body match if request is sending a file, maybe do more check here
		}
		return false
	}

	actualBody, err := actualRequest.GetBody()
	if err != nil {
		panic(fmt.Errorf("cassette body matcher: failed to copy actualRequest body: %w", err)) // lintignore: R009
	}
	actualRawBody, err := io.ReadAll(actualBody)
	if err != nil {
		panic(fmt.Errorf("cassette body matcher: failed to read actualRequest body: %w", err)) // lintignore: R009
	}

	// Try to match raw bodies if they are not JSON (ex: cloud-init config)
	if string(actualRawBody) == cassetteRequest.Body {
		return true
	}

	actualJSON := make(map[string]interface{})
	cassetteJSON := make(map[string]interface{})

	err = xml.Unmarshal(actualRawBody, new(interface{}))
	if err == nil {
		// match if content is xml
		return true
	}

	if !json.Valid(actualRawBody) {
		values, err := url.ParseQuery(string(actualRawBody))
		if err != nil {
			panic(fmt.Errorf("cassette body matcher: failed to parse body as url values: %w", err)) // lintignore: R009
		}

		// Remove keys that should be ignored during compare
		for _, key := range BodyMatcherIgnore {
			values.Del(key)
		}

		// Compare url values
		return compareFormBodies(values, cassetteRequest.Form)
	}

	err = json.Unmarshal(actualRawBody, &actualJSON)
	if err != nil {
		panic(fmt.Errorf("cassette body matcher: failed to parse json body: %w", err)) // lintignore: R009
	}

	err = json.Unmarshal([]byte(cassetteRequest.Body), &cassetteJSON)
	if err != nil {
		// actualRequest contains JSON but cassette may not contain JSON, this doesn't match in this case
		return false
	}

	// Remove keys that should be ignored during compare
	for _, key := range BodyMatcherIgnore {
		delete(actualJSON, key)
		delete(cassetteJSON, key)
	}

	return compareJSONBodies(cassetteJSON, actualJSON)
}
