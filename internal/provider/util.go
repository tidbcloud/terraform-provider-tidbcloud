package provider

import (
	"context"
	cryptorand "crypto/rand"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"math/big"
	"math/rand"
	"sync"
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

// RetryWithInterval is a wrapper of retry.RetryContext, interval: (0s,180s)
func RetryWithInterval(ctx context.Context, timeout time.Duration, interval time.Duration, f retry.RetryFunc) error {
	// These are used to pull the error out of the function; need a mutex to
	// avoid a data race.
	var resultErr error
	var resultErrMu sync.Mutex

	c := &retry.StateChangeConf{
		Pending:      []string{"retryableerror"},
		Target:       []string{"success"},
		Timeout:      timeout,
		MinTimeout:   500 * time.Millisecond,
		PollInterval: interval,
		Refresh: func() (interface{}, string, error) {
			rerr := f()

			resultErrMu.Lock()
			defer resultErrMu.Unlock()

			if rerr == nil {
				resultErr = nil
				return 42, "success", nil
			}

			resultErr = rerr.Err

			if rerr.Retryable {
				return 42, "retryableerror", nil
			}
			return nil, "quit", rerr.Err
		},
	}

	_, waitErr := c.WaitForStateContext(ctx)

	// Need to acquire the lock here to be able to avoid race using resultErr as
	// the return value
	resultErrMu.Lock()
	defer resultErrMu.Unlock()

	// resultErr may be nil because the wait timed out and resultErr was never
	// set; this is still an error
	if resultErr == nil {
		return waitErr
	}
	// resultErr takes precedence over waitErr if both are set because it is
	// more likely to be useful
	return resultErr
}
