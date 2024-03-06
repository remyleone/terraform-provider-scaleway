package types

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/scaleway/scaleway-sdk-go/namegenerator"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"hash/crc32"
	"strconv"
	"strings"
)

func ExpandStringPtr(data interface{}) *string {
	if data == nil || data == "" {
		return nil
	}
	return scw.StringPtr(data.(string))
}

func ExpandOrGenerateString(data interface{}, prefix string) string {
	if data == nil || data == "" {
		return NewRandomName(prefix)
	}
	return data.(string)
}

// NewRandomName returns a random name prefixed for terraform.
func NewRandomName(prefix string) string {
	return namegenerator.GetRandomName("tf", prefix)
}

func ExpandStrings(data interface{}) []string {
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

func ExpandStringsPtr(data interface{}) *[]string {
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

// ExpandUpdatedStringsPtr expands a string slice but will default to an empty list.
// Should be used on schema update so emptying a list will update resource.
func ExpandUpdatedStringsPtr(data interface{}) *[]string {
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

func FlattenStringPtr(s *string) interface{} {
	if s == nil {
		return ""
	}
	return *s
}

func FlattenSliceStringPtr(s []*string) interface{} {
	res := make([]interface{}, 0, len(s))
	for _, strPtr := range s {
		res = append(res, FlattenStringPtr(strPtr))
	}
	return res
}

func FlattenSliceString(s []string) interface{} {
	res := make([]interface{}, 0, len(s))
	for _, strPtr := range s {
		res = append(res, strPtr)
	}
	return res
}

func ExpandStringsOrEmpty(data interface{}) []string {
	stringSlice := make([]string, 0, len(data.([]interface{})))
	if _, ok := data.([]interface{}); !ok || data == nil {
		return stringSlice
	}
	for _, s := range data.([]interface{}) {
		stringSlice = append(stringSlice, s.(string))
	}
	return stringSlice
}

func ExpandSliceStringPtr(data interface{}) []*string {
	if data == nil {
		return nil
	}
	stringSlice := []*string(nil)
	for _, s := range data.([]interface{}) {
		stringSlice = append(stringSlice, ExpandStringPtr(s))
	}
	return stringSlice
}

// ExpandListKeys return the list of keys for an attribute in a list
// example for private-networks.#.id in a list of size 2
// will return private-networks.0.id and private-networks.1.id
// additional_volume_ids.#
// will return additional_volume_ids.0 and additional_volume_ids.1
func ExpandListKeys(key string, diff *schema.ResourceDiff) []string {
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

func ExpandUpdatedStringPtr(data interface{}) *string {
	str := ""
	if data != nil {
		str = data.(string)
	}
	return &str
}

// StringHashcode hashes a string to a unique hashcode.
//
// crc32 returns a uint32, but for our use we need
// and non-negative integer. Here we cast to an integer
// and invert it if the result is negative.
func StringHashcode(s string) int {
	v := int(crc32.ChecksumIEEE([]byte(s)))
	if v >= 0 {
		return v
	}
	if -v >= 0 {
		return -v
	}
	// v == MinInt
	return 0
}

func ExpandStringWithDefault(data interface{}, defaultValue string) string {
	if data == nil || data.(string) == "" {
		return defaultValue
	}
	return data.(string)
}
