package provider

const (
	TiDBCloudPublicKey  string = "TIDBCLOUD_PUBLIC_KEY"
	TiDBCloudPrivateKey string = "TIDBCLOUD_PRIVATE_KEY"
	TiDBCloudHOST       string = "TIDBCLOUD_HOST"
	TiDBCloudProjectID  string = "TIDBCLOUD_PROJECT_ID"
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
