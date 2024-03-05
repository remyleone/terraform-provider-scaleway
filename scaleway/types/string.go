package types

import (
	"github.com/scaleway/scaleway-sdk-go/namegenerator"
	"github.com/scaleway/scaleway-sdk-go/scw"
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
