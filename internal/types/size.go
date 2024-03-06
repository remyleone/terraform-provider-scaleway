package types

import "github.com/scaleway/scaleway-sdk-go/scw"

const Gb uint64 = 1000 * 1000 * 1000

func FlattenSize(size *scw.Size) interface{} {
	if size == nil {
		return 0
	}
	return *size
}
