package provider

import (
	cryptorand "crypto/rand"
	"math/big"
	"math/rand"
	"time"
)

const (
	TiDBCloudPublicKey  string = "TIDBCLOUD_PUBLIC_KEY"
	TiDBCloudPrivateKey string = "TIDBCLOUD_PRIVATE_KEY"
	TiDBCloudHOST       string = "TIDBCLOUD_HOST"
	TiDBCloudProjectID  string = "TIDBCLOUD_PROJECT_ID"
	TiDBCloudClusterID  string = "TIDBCLOUD_CLUSTER_ID"
	UserAgent           string = "terraform-provider-tidbcloud"
)

// HookGlobal sets `*ptr = val` and returns a closure for restoring `*ptr` to
// its original value. A runtime panic will occur if `val` is not assignable to
// `*ptr`.
func HookGlobal[T any](ptr *T, val T) func() {
	orig := *ptr
	*ptr = val
	return func() { *ptr = orig }
}

func Ptr[T any](v T) *T {
	return &v
}

func GenerateRandomString(n int) string {
	letters := "abcdefghijklmnopqrstuvwxyz"
	rand.Seed(time.Now().UnixNano())
	letterRunes := []rune(letters)
	b := make([]rune, n)
	for i := range b {
		randNum, _ := cryptorand.Int(cryptorand.Reader, big.NewInt(int64(len(letterRunes))))
		b[i] = letterRunes[randNum.Int64()]
	}
	return string(b)
}

type Knowable interface {
	IsUnknown() bool
	IsNull() bool
}

// IsKnown is a shortcut that checks in a value is neither null nor unknown.
func IsKnown(t Knowable) bool {
	return !t.IsUnknown() && !t.IsNull()
}
