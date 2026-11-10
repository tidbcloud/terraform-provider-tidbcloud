package tidbcloud

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

// MemberAPIError wraps a non-2xx response from the hand-written member REST
// client, preserving the HTTP status code so callers can react to it (for
// example, treating 404 on delete as success).
type MemberAPIError struct {
	StatusCode int
	Err        error
}

func (e *MemberAPIError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return fmt.Sprintf("unexpected status: %d", e.StatusCode)
}

func (e *MemberAPIError) Unwrap() error { return e.Err }

// IsNotFoundError reports whether err is a MemberAPIError with a 404 status.
func IsNotFoundError(err error) bool {
	var apiErr *MemberAPIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusNotFound
	}
	return false
}

// The IAM Go SDK (github.com/tidbcloud/tidbcloud-cli/.../v1beta1/iam) does not
// expose the organization member API, so the request/response models below are
// hand-written to match the IAM v1beta1 OpenAPI spec
// (https://docs-download.pingcap.com/api/tidbcloud-oas-v1beta1-iam.swagger.json).

// OpenApiRbacRole is a single RBAC role assignment. ScopeId identifies the
// organization, project, or instance the role applies to. For organization
// roles the scope is implicit (the API key's organization) and may be empty.
type OpenApiRbacRole struct {
	RbacRole string  `json:"rbacRole,omitempty"`
	ScopeId  *string `json:"scopeId,omitempty"`
}

// OpenApiUser is a member of an organization.
type OpenApiUser struct {
	UserId        *string           `json:"userId,omitempty"`
	Email         *string           `json:"email,omitempty"`
	FirstName     *string           `json:"firstName,omitempty"`
	LastName      *string           `json:"lastName,omitempty"`
	Status        *string           `json:"status,omitempty"`
	InviteTime    *string           `json:"inviteTime,omitempty"`
	LastLoginTime *string           `json:"lastLoginTime,omitempty"`
	OrgRole       *OpenApiRbacRole  `json:"orgRole,omitempty"`
	InstanceRoles []OpenApiRbacRole `json:"instanceRoles,omitempty"`
	ProjectRoles  []OpenApiRbacRole `json:"projectRoles,omitempty"`
}

// OpenApiListUsersRsp is the response of GET /members.
type OpenApiListUsersRsp struct {
	Users         []OpenApiUser `json:"users,omitempty"`
	NextPageToken *string       `json:"nextPageToken,omitempty"`
	TotalSize     *int64        `json:"totalSize,omitempty"`
}

// OpenApiInviteUsersReq is the body of POST /members.
type OpenApiInviteUsersReq struct {
	Emails        []string          `json:"emails"`
	OrgRole       *OpenApiRbacRole  `json:"orgRole"`
	InstanceRoles []OpenApiRbacRole `json:"instanceRoles,omitempty"`
	ProjectRoles  []OpenApiRbacRole `json:"projectRoles,omitempty"`
}

// OpenApiInviteUserResult maps an invited email to its assigned member ID.
type OpenApiInviteUserResult struct {
	Email  *string `json:"email,omitempty"`
	UserId *string `json:"userId,omitempty"`
}

// OpenApiInviteUsersRsp is the response of POST /members.
type OpenApiInviteUsersRsp struct {
	Success *bool                     `json:"success,omitempty"`
	Message *string                   `json:"message,omitempty"`
	Users   []OpenApiInviteUserResult `json:"users,omitempty"`
}

// OpenApiUpdateUserReq is the body of PATCH /members/{user_id}. The API fully
// replaces a role list only when the field is present, so the role fields are
// intentionally NOT omitempty: an empty (non-nil) slice serializes as `[]` and
// clears the member's roles, which is required for Terraform to remove them.
type OpenApiUpdateUserReq struct {
	OrgRole       *OpenApiRbacRole  `json:"orgRole,omitempty"`
	InstanceRoles []OpenApiRbacRole `json:"instanceRoles"`
	ProjectRoles  []OpenApiRbacRole `json:"projectRoles"`
}

// ListMembersParams holds the optional filters/pagination for GET /members.
type ListMembersParams struct {
	PageSize  *int32
	PageToken *string
	Email     *string
}

func (d *IAMClientDelegate) ListMembers(ctx context.Context, params *ListMembersParams) (*OpenApiListUsersRsp, error) {
	q := url.Values{}
	if params != nil {
		if params.PageSize != nil {
			q.Set("pageSize", strconv.FormatInt(int64(*params.PageSize), 10))
		}
		if !IsNilOrEmptyStr(params.PageToken) {
			q.Set("pageToken", *params.PageToken)
		}
		if !IsNilOrEmptyStr(params.Email) {
			q.Set("email", *params.Email)
		}
	}
	var out OpenApiListUsersRsp
	if err := d.doMemberRequest(ctx, http.MethodGet, "/members", q, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (d *IAMClientDelegate) InviteMembers(ctx context.Context, body *OpenApiInviteUsersReq) (*OpenApiInviteUsersRsp, error) {
	var out OpenApiInviteUsersRsp
	if err := d.doMemberRequest(ctx, http.MethodPost, "/members", nil, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (d *IAMClientDelegate) UpdateMember(ctx context.Context, userID string, body *OpenApiUpdateUserReq) error {
	ensureNonNilUpdateRoles(body)
	return d.doMemberRequest(ctx, http.MethodPatch, "/members/"+url.PathEscape(userID), nil, body, nil)
}

// ensureNonNilUpdateRoles normalizes nil role lists to empty slices so they
// serialize as `[]` (which clears the member's roles) rather than `null`, which
// the API may treat as "unchanged".
func ensureNonNilUpdateRoles(body *OpenApiUpdateUserReq) {
	if body == nil {
		return
	}
	if body.ProjectRoles == nil {
		body.ProjectRoles = []OpenApiRbacRole{}
	}
	if body.InstanceRoles == nil {
		body.InstanceRoles = []OpenApiRbacRole{}
	}
}

func (d *IAMClientDelegate) DeleteMember(ctx context.Context, userID string) error {
	return d.doMemberRequest(ctx, http.MethodDelete, "/members/"+url.PathEscape(userID), nil, nil, nil)
}

// doMemberRequest issues a single JSON request against the IAM member API using
// the shared digest-authenticated HTTP client. Non-2xx responses are turned
// into errors via parseError so they read the same as SDK-backed calls.
func (d *IAMClientDelegate) doMemberRequest(ctx context.Context, method, path string, query url.Values, body, out interface{}) error {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reqBody = bytes.NewReader(b)
	}

	u := d.memberBaseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, u, reqBody)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return parseError(err, resp)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		statusCode := resp.StatusCode
		return &MemberAPIError{
			StatusCode: statusCode,
			Err:        parseError(fmt.Errorf("unexpected status: %s", resp.Status), resp),
		}
	}
	defer resp.Body.Close()

	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil && err != io.EOF {
			return err
		}
	}
	return nil
}

// IsNilOrEmptyStr reports whether a *string is nil or points to an empty string.
func IsNilOrEmptyStr(s *string) bool {
	return s == nil || *s == ""
}
