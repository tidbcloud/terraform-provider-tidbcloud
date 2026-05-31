package tidbcloud

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
)

// TestUpdateUserReqClearsRoles verifies that an update with empty (or nil) role
// lists serializes the role fields as `[]` rather than omitting them, so the
// API actually clears the member's roles instead of leaving them unchanged.
func TestUpdateUserReqClearsRoles(t *testing.T) {
	cases := map[string]OpenApiUpdateUserReq{
		"nil roles": {
			OrgRole: &OpenApiRbacRole{RbacRole: "org:member"},
		},
		"empty roles": {
			OrgRole:       &OpenApiRbacRole{RbacRole: "org:member"},
			ProjectRoles:  []OpenApiRbacRole{},
			InstanceRoles: []OpenApiRbacRole{},
		},
	}
	for name, req := range cases {
		req := req
		ensureNonNilUpdateRoles(&req)
		b, err := json.Marshal(req)
		if err != nil {
			t.Fatalf("%s: marshal failed: %v", name, err)
		}
		got := string(b)
		if !strings.Contains(got, `"projectRoles":[]`) {
			t.Errorf("%s: expected projectRoles to serialize as [], got %s", name, got)
		}
		if !strings.Contains(got, `"instanceRoles":[]`) {
			t.Errorf("%s: expected instanceRoles to serialize as [], got %s", name, got)
		}
	}
}

func TestIsNotFoundError(t *testing.T) {
	if !IsNotFoundError(&MemberAPIError{StatusCode: http.StatusNotFound}) {
		t.Error("expected 404 MemberAPIError to be reported as not found")
	}
	if IsNotFoundError(&MemberAPIError{StatusCode: http.StatusInternalServerError}) {
		t.Error("expected 500 MemberAPIError not to be reported as not found")
	}
	if IsNotFoundError(errors.New("some other error")) {
		t.Error("expected a plain error not to be reported as not found")
	}
	if IsNotFoundError(nil) {
		t.Error("expected nil error not to be reported as not found")
	}
}
