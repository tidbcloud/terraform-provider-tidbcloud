package provider

import (
	"context"
	"strings"
	"sync"

	"github.com/juju/errors"
	"github.com/tidbcloud/terraform-provider-tidbcloud/tidbcloud"
)

// memberListCache lists the organization members once and serves the member
// lookups of a Terraform run from that list. Without it, every tidbcloud_member
// and tidbcloud_member read calls ListMembers on its own, so a plan with N
// members sends N requests and can exceed the per-API-key rate limit.
//
// The provider process lives for one Terraform command, so the list is as fresh
// as the refresh of that command. Writes to members call invalidate so that a
// later lookup lists the members again.
type memberListCache struct {
	mu       sync.Mutex
	loaded   bool
	members  []tidbcloud.OpenApiUser
	inflight *memberListing
}

// memberListing is one listing in progress. The lookups that arrive while it
// runs wait for it and share its result, an error included.
type memberListing struct {
	done    chan struct{}
	members []tidbcloud.OpenApiUser
	err     error
}

// list returns all members of the organization, listing them on the first
// call. Concurrent calls share one listing, and the mutex is not held while it
// runs, so a waiting call still returns when its context is done. A failed
// listing is returned to the calls that waited for it but is not cached, so a
// later call tries again; without sharing it, each waiting call would retry
// the failed listing in turn.
func (c *memberListCache) list(ctx context.Context, client tidbcloud.TiDBCloudIAMClient) ([]tidbcloud.OpenApiUser, error) {
	c.mu.Lock()
	if c.loaded {
		members := c.members
		c.mu.Unlock()
		return members, nil
	}
	if l := c.inflight; l != nil {
		c.mu.Unlock()
		select {
		case <-l.done:
			return l.members, l.err
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	l := &memberListing{done: make(chan struct{})}
	c.inflight = l
	c.mu.Unlock()

	l.members, l.err = listAllMembers(ctx, client)

	c.mu.Lock()
	// After an invalidate, the listing may predate a write, so it is only
	// returned to the calls that waited for it.
	if c.inflight == l {
		c.inflight = nil
		if l.err == nil {
			c.members, c.loaded = l.members, true
		}
	}
	c.mu.Unlock()
	close(l.done)
	return l.members, l.err
}

// find returns the member with the given email from the cached list, or nil if
// no member has that email.
func (c *memberListCache) find(ctx context.Context, client tidbcloud.TiDBCloudIAMClient, email string) (*tidbcloud.OpenApiUser, error) {
	members, err := c.list(ctx, client)
	if err != nil {
		return nil, err
	}
	for i := range members {
		if members[i].Email != nil && strings.EqualFold(*members[i].Email, email) {
			u := members[i]
			return &u, nil
		}
	}
	return nil, nil
}

func (c *memberListCache) invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.loaded, c.members, c.inflight = false, nil, nil
}

func listAllMembers(ctx context.Context, client tidbcloud.TiDBCloudIAMClient) ([]tidbcloud.OpenApiUser, error) {
	var items []tidbcloud.OpenApiUser

	pageSize := int32(DefaultPageSize)
	var pageToken *string
	for {
		rsp, err := client.ListMembers(ctx, &tidbcloud.ListMembersParams{
			PageSize:  &pageSize,
			PageToken: pageToken,
		})
		if err != nil {
			return nil, errors.Trace(err)
		}
		items = append(items, rsp.Users...)

		pageToken = rsp.NextPageToken
		if IsNilOrEmpty(pageToken) {
			break
		}
	}
	return items, nil
}
