package meta

// TerraformResourceData is an interface for *schema.ResourceData. (used for mock)
type TerraformResourceData interface {
	HasChange(string) bool
	GetOk(string) (interface{}, bool)
	Get(string) interface{}
	Id() string
}
