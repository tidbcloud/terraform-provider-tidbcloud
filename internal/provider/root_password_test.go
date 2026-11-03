package provider

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	clusterAPI "github.com/c4pt0r/go-tidbcloud-sdk-v1/client/cluster"
	"github.com/golang/mock/gomock"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	mockClient "github.com/tidbcloud/terraform-provider-tidbcloud/mock"
	"github.com/tidbcloud/tidbcloud-cli/pkg/tidbcloud/v1beta1/dedicated"
)

func TestDedicatedClusterRootPasswordCreatePayload(t *testing.T) {
	for _, tc := range []struct {
		name      string
		password  types.String
		wantError bool
	}{
		{"omitted", types.StringNull(), false},
		{"valid", types.StringValue("example-password"), false},
		{"empty", types.StringValue(""), true},
		{"short", types.StringValue("short"), true},
		{"long", types.StringValue(strings.Repeat("x", 65)), true},
		{"unknown", types.StringUnknown(), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body, err := buildCreateDedicatedClusterBody(dedicatedClusterResourceData{RootPassword: tc.password})
			if (err != nil) != tc.wantError {
				t.Fatalf("error = %v, want error %v", err, tc.wantError)
			}
			if tc.wantError {
				return
			}
			encoded, err := json.Marshal(body)
			if err != nil {
				t.Fatal(err)
			}
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(encoded, &fields); err != nil {
				t.Fatal(err)
			}
			if tc.password.IsNull() {
				if _, present := fields["rootPassword"]; present {
					t.Fatal("omitted password was included in the create request")
				}
			} else if body.RootPassword == nil || *body.RootPassword != tc.password.ValueString() {
				t.Fatal("configured password was not preserved")
			}
		})
	}
}

func TestClusterRootPasswordCreate(t *testing.T) {
	for _, tc := range []struct {
		name     string
		password types.String
		valid    bool
	}{
		{"omitted", types.StringNull(), false},
		{"unknown", types.StringUnknown(), false},
		{"empty", types.StringValue(""), false},
		{"short", types.StringValue("short"), false},
		{"long", types.StringValue(strings.Repeat("x", 65)), false},
		{"valid", types.StringValue("example-password"), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			client := mockClient.NewMockTiDBCloudClient(gomock.NewController(t))
			r := clusterResource{provider: &tidbcloudProvider{configured: true, client: client}}
			data := rootPasswordLegacyFixture(tc.password)
			state := rootPasswordState(t, &r, &data)
			if tc.valid {
				client.EXPECT().CreateCluster(gomock.Any()).DoAndReturn(func(params *clusterAPI.CreateClusterParams, _ ...clusterAPI.ClientOption) (*clusterAPI.CreateClusterOK, error) {
					if params.Body.Config.RootPassword == nil || *params.Body.Config.RootPassword != tc.password.ValueString() {
						t.Fatal("create request lost the configured password")
					}
					return nil, errors.New("stop after inspecting create request")
				})
			}
			resp := resource.CreateResponse{State: state}
			r.Create(ctx, resource.CreateRequest{Config: tfsdk.Config{Raw: state.Raw, Schema: state.Schema}}, &resp)
			if !resp.Diagnostics.HasError() {
				t.Fatal("expected a diagnostic before completing create")
			}
			if !tc.valid && !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), "root_password") {
				t.Fatalf("unexpected diagnostic: %v", resp.Diagnostics)
			}
		})
	}
}

func TestDedicatedClusterRootPasswordUpdate(t *testing.T) {
	for _, tc := range []struct {
		name      string
		password  types.String
		wantReset bool
		wantError bool
	}{
		{"removed", types.StringNull(), false, false},
		{"unchanged", types.StringValue("old-password"), false, false},
		{"rotated", types.StringValue("new-password"), true, false},
		{"rotation failed", types.StringValue("new-password"), true, true},
		{"empty", types.StringValue(""), false, true},
		{"short", types.StringValue("short"), false, true},
		{"long", types.StringValue(strings.Repeat("x", 65)), false, true},
		{"unknown", types.StringUnknown(), false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			client := mockClient.NewMockTiDBCloudDedicatedClient(gomock.NewController(t))
			r := dedicatedClusterResource{provider: &tidbcloudProvider{DedicatedClient: client}}
			var cluster dedicated.TidbCloudOpenApidedicatedv1beta1Cluster
			if err := cluster.UnmarshalJSON([]byte(testUTTidbCloudOpenApidedicatedv1beta1Cluster("cluster-id", "example", "ACTIVE", "2C4G", "2 vCPU, 4 GiB"))); err != nil {
				t.Fatal(err)
			}
			data := dedicatedClusterResourceData{RootPassword: types.StringValue("old-password"), Paused: types.BoolValue(false)}
			if diags := refreshDedicatedClusterResourceData(ctx, &cluster, &data); diags.HasError() {
				t.Fatal(diags)
			}
			state := rootPasswordState(t, &r, &data)
			data.RootPassword = tc.password
			planned := rootPasswordState(t, &r, &data)
			if tc.wantReset {
				var resetError error
				if tc.wantError {
					resetError = errors.New("password reset failed")
				} else {
					client.EXPECT().UpdateCluster(gomock.Any(), "cluster-id", gomock.Any()).Return(&cluster, nil)
				}
				client.EXPECT().ChangeClusterRootPassword(gomock.Any(), "cluster-id", &dedicated.V1beta1ClusterServiceResetRootPasswordBody{RootPassword: tc.password.ValueString()}).Return(resetError)
			}
			if !tc.wantError {
				client.EXPECT().GetCluster(gomock.Any(), "cluster-id").Return(&cluster, nil)
			}
			resp := resource.UpdateResponse{State: state}
			r.Update(ctx, resource.UpdateRequest{Plan: tfsdk.Plan{Raw: planned.Raw, Schema: planned.Schema}, State: state}, &resp)
			if resp.Diagnostics.HasError() != tc.wantError {
				t.Fatalf("diagnostics = %v, want error %v", resp.Diagnostics, tc.wantError)
			}
			var got types.String
			if diags := resp.State.GetAttribute(ctx, path.Root("root_password"), &got); diags.HasError() {
				t.Fatal(diags)
			}
			want := tc.password
			if tc.wantError {
				want = types.StringValue("old-password")
			}
			if !got.Equal(want) {
				t.Fatal("password state does not match the completed operation")
			}
		})
	}
}

func TestClusterRootPasswordUpdateRejected(t *testing.T) {
	for _, password := range []types.String{types.StringValue("new-password"), types.StringValue(""), types.StringUnknown()} {
		ctx := context.Background()
		client := mockClient.NewMockTiDBCloudClient(gomock.NewController(t))
		r := clusterResource{provider: &tidbcloudProvider{client: client}}
		data := rootPasswordLegacyFixture(types.StringValue("old-password"))
		state := rootPasswordState(t, &r, &data)
		data.Config.RootPassword = password
		planned := rootPasswordState(t, &r, &data)
		resp := resource.UpdateResponse{State: state}
		r.Update(ctx, resource.UpdateRequest{Plan: tfsdk.Plan{Raw: planned.Raw, Schema: planned.Schema}, State: state}, &resp)
		if !resp.Diagnostics.HasError() {
			t.Fatal("password change was not rejected")
		}
		if !resp.State.Raw.Equal(state.Raw) {
			t.Fatal("rejected password change modified state")
		}
	}
}

func TestClusterRootPasswordRemovalAndUnchangedUpdates(t *testing.T) {
	for _, tc := range []struct {
		name             string
		password         types.String
		updateComponents bool
	}{
		{"removed", types.StringNull(), false},
		{"removed with component update", types.StringNull(), true},
		{"unchanged with component update", types.StringValue("old-password"), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			client := mockClient.NewMockTiDBCloudClient(gomock.NewController(t))
			r := clusterResource{provider: &tidbcloudProvider{client: client}}
			data := rootPasswordLegacyFixture(types.StringValue("old-password"))
			state := rootPasswordState(t, &r, &data)
			data.Config.RootPassword = tc.password
			if tc.updateComponents {
				data.Config.Components.TiDB.NodeQuantity = 2
			}
			planned := rootPasswordState(t, &r, &data)
			if tc.updateComponents {
				client.EXPECT().UpdateCluster(gomock.Any()).DoAndReturn(func(params *clusterAPI.UpdateClusterParams, _ ...clusterAPI.ClientOption) (*clusterAPI.UpdateClusterOK, error) {
					if params.Body.Config.Components == nil || *params.Body.Config.Components.Tidb.NodeQuantity != 2 {
						t.Fatal("component update was lost")
					}
					return &clusterAPI.UpdateClusterOK{}, nil
				})
			}
			var cluster clusterAPI.GetClusterOKBody
			if err := json.Unmarshal([]byte(`{
				"id":"cluster-id","project_id":"project-id","name":"example","cluster_type":"DEDICATED","cloud_provider":"AWS","region":"us-west-2",
				"config":{"port":4000,"components":{"tidb":{"node_size":"2C4G","node_quantity":2},"tikv":{"node_size":"2C4G","node_quantity":3,"storage_size_gib":100}}},
				"status":{"cluster_status":"AVAILABLE","connection_strings":{"default_user":"root"}}
			}`), &cluster); err != nil {
				t.Fatal(err)
			}
			cluster.Config.Components.Tidb.NodeQuantity = Ptr(data.Config.Components.TiDB.NodeQuantity)
			client.EXPECT().GetCluster(gomock.Any()).Return(&clusterAPI.GetClusterOK{Payload: &cluster}, nil)
			resp := resource.UpdateResponse{State: state}
			r.Update(ctx, resource.UpdateRequest{Plan: tfsdk.Plan{Raw: planned.Raw, Schema: planned.Schema}, State: state}, &resp)
			if resp.Diagnostics.HasError() {
				t.Fatal(resp.Diagnostics)
			}
			var got types.String
			if diags := resp.State.GetAttribute(ctx, path.Root("config").AtName("root_password"), &got); diags.HasError() {
				t.Fatal(diags)
			}
			if !got.Equal(tc.password) {
				t.Fatal("password state does not match configuration")
			}
		})
	}
}

func TestClusterRootPasswordSchemaValidation(t *testing.T) {
	ctx := context.Background()
	var legacySchema, dedicatedSchema resource.SchemaResponse
	(&clusterResource{}).Schema(ctx, resource.SchemaRequest{}, &legacySchema)
	(&dedicatedClusterResource{}).Schema(ctx, resource.SchemaRequest{}, &dedicatedSchema)
	attributes := []schema.StringAttribute{
		legacySchema.Schema.Attributes["config"].(schema.SingleNestedAttribute).Attributes["root_password"].(schema.StringAttribute),
		dedicatedSchema.Schema.Attributes["root_password"].(schema.StringAttribute),
	}
	for _, attribute := range attributes {
		if !attribute.Sensitive {
			t.Error("root password must be sensitive")
		}
		for _, tc := range []struct {
			password  types.String
			wantError bool
		}{
			{types.StringNull(), false}, {types.StringUnknown(), false}, {types.StringValue(""), true},
			{types.StringValue("1234567"), true}, {types.StringValue("12345678"), false},
			{types.StringValue(strings.Repeat("x", 64)), false}, {types.StringValue(strings.Repeat("x", 65)), true},
		} {
			var resp validator.StringResponse
			for _, v := range attribute.Validators {
				v.ValidateString(ctx, validator.StringRequest{Path: path.Root("root_password"), ConfigValue: tc.password}, &resp)
			}
			if resp.Diagnostics.HasError() != tc.wantError {
				t.Errorf("validation error = %v, want %v", resp.Diagnostics.HasError(), tc.wantError)
			}
		}
	}
}

func rootPasswordState(t *testing.T, r resource.Resource, data interface{}) tfsdk.State {
	t.Helper()
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)
	state := tfsdk.State{Schema: resp.Schema}
	if diags := state.Set(context.Background(), data); diags.HasError() {
		t.Fatal(diags)
	}
	return state
}

func rootPasswordLegacyFixture(password types.String) clusterResourceData {
	return clusterResourceData{
		ClusterId: types.StringValue("cluster-id"), ProjectId: "project-id", Name: "example", ClusterType: ded,
		CloudProvider: "AWS", Region: "us-west-2",
		Config: clusterConfig{RootPassword: password, Components: &components{
			TiDB: &componentTiDB{NodeSize: "2C4G", NodeQuantity: 1},
			TiKV: &componentTiKV{NodeSize: "2C4G", NodeQuantity: 3, StorageSizeGib: 100},
		}},
	}
}
