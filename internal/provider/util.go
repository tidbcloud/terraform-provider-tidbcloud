package provider

import (
	cryptorand "crypto/rand"
	"math/big"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	TiDBCloudPublicKey         string = "TIDBCLOUD_PUBLIC_KEY"
	TiDBCloudPrivateKey        string = "TIDBCLOUD_PRIVATE_KEY"
	TiDBCloudHost              string = "TIDBCLOUD_HOST"
	TiDBCloudDedicatedEndpoint string = "TIDBCLOUD_DEDICATED_ENDPOINT"
	TiDBCloudIAMEndpoint       string = "TIDBCLOUD_IAM_ENDPOINT"
	TiDBCloudProjectID         string = "TIDBCLOUD_PROJECT_ID"
	TiDBCloudClusterID         string = "TIDBCLOUD_CLUSTER_ID"
	UserAgent                  string = "terraform-provider-tidbcloud"
)

const ()

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

// func validateFieldChanges(plan, state interface{}, ignoredPaths []string) diag.Diagnostics {
// 	var diagnostics diag.Diagnostics

// 	ignoredFields := make(map[string]bool)
// 	for _, path := range ignoredPaths {
// 		ignoredFields[path] = true
// 	}

// 	// Create custom cmp.Options to ignore specified fields
// 	opts := cmp.FilterPath(func(p cmp.Path) bool {
// 		fieldPath := p.GoString()
// 		_, isIgnored := ignoredFields[fieldPath]
// 		return isIgnored
// 	}, cmp.Ignore())

// 	// Compare plan and state with options
// 	if diff := cmp.Diff(state, plan, opts); diff != "" {
// 		diagnostics.AddError(
// 			"Unexpected field Changes Detected",
// 			fmt.Sprintf("You cannot modify both %v and other fields at the same time. Unexpected field changes:\n%s", ignoredFields, diff),
// 		)
// 	}

// 	return diagnostics
// }
