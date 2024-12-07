package provider

import (
	"context"
	"fmt"

	restoreApi "github.com/c4pt0r/go-tidbcloud-sdk-v1/client/restore"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type restoreResourceData struct {
	ClusterId       types.String  `tfsdk:"cluster_id"`
	RestoreId       types.String  `tfsdk:"id"`
	ProjectId       string        `tfsdk:"project_id"`
	Name            string        `tfsdk:"name"`
	BackupId        string        `tfsdk:"backup_id"`
	Config          restoreConfig `tfsdk:"config"`
	CreateTimestamp types.String  `tfsdk:"create_timestamp"`
	Status          types.String  `tfsdk:"status"`
	Cluster         *cluster      `tfsdk:"cluster"`
	ErrorMessage    types.String  `tfsdk:"error_message"`
}

type restoreConfig struct {
	RootPassword types.String `tfsdk:"root_password"`
	Port         types.Int64  `tfsdk:"port"`
	Components   *components  `tfsdk:"components"`
	IPAccessList []ipAccess   `tfsdk:"ip_access_list"`
}

type cluster struct {
	Id     string `tfsdk:"id"`
	Name   string `tfsdk:"name"`
	Status string `tfsdk:"status"`
}

// Ensure provider defined types fully satisfy framework interfaces
var _ resource.Resource = &restoreResource{}

type restoreResource struct {
	provider *tidbcloudProvider
}

func NewRestoreResource() resource.Resource {
	return &restoreResource{}
}

func (r *restoreResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_restore"
}

func (r *restoreResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	var ok bool
	if r.provider, ok = req.ProviderData.(*tidbcloudProvider); !ok {
		resp.Diagnostics.AddError("Internal provider error",
			fmt.Sprintf("Error in Configure: expected %T but got %T", tidbcloudProvider{}, req.ProviderData))
	}
}

func (r *restoreResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "restore resource",
		Attributes: map[string]schema.Attribute{
			"cluster_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the cluster",
				Computed:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the restore",
				Computed:            true,
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the project. You can get the project ID from [tidbcloud_projects datasource](../data-sources/projects.md).",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the restore",
				Required:            true,
			},
			"backup_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The ID of the backup",
			},
			"config": schema.SingleNestedAttribute{
				MarkdownDescription: "The configuration of the cluster",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"root_password": schema.StringAttribute{
						MarkdownDescription: "The root password to access the cluster. It must be 8-64 characters.",
						Required:            true,
					},
					"port": schema.Int64Attribute{
						MarkdownDescription: "The TiDB port for connection. The port must be in the range of 1024-65535 except 10080, 4000 in default.\n" +
							"  - For a Serverless Tier cluster, only port 4000 is available.",
						Optional: true,
						Computed: true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
					"components": schema.SingleNestedAttribute{
						MarkdownDescription: "The components of the cluster.\n" +
							"  - For a Serverless Tier cluster, the components value can not be set." +
							"  - For a Dedicated Tier cluster, the components value must be set.",
						Required: true,
						Attributes: map[string]schema.Attribute{
							"tidb": schema.SingleNestedAttribute{
								Required: true,
								Attributes: map[string]schema.Attribute{
									"node_size": schema.StringAttribute{
										MarkdownDescription: "The size of the TiDB component in the cluster, You can get the available node size of each region from the [tidbcloud_cluster_specs datasource](./cluster_specs.md).\n" +
											"  - If the vCPUs of TiDB or TiKV component is 2 or 4, then their vCPUs need to be the same.\n" +
											"  - If the vCPUs of TiDB or TiKV component is 2 or 4, then the cluster does not support TiFlash.\n" +
											"  - Can not modify node_size of an existing cluster.",
										Required: true,
									},
									"node_quantity": schema.Int64Attribute{
										MarkdownDescription: "The number of nodes in the cluster. You can get the minimum and step of a node quantity from the [tidbcloud_cluster_specs datasource](./cluster_specs.md).",
										Required:            true,
									},
								},
							},
							"tikv": schema.SingleNestedAttribute{
								Required: true,
								Attributes: map[string]schema.Attribute{
									"node_size": schema.StringAttribute{
										MarkdownDescription: "The size of the TiKV component in the cluster, You can get the available node size of each region from the [tidbcloud_cluster_specs datasource](./cluster_specs.md).\n" +
											"  - If the vCPUs of TiDB or TiKV component is 2 or 4, then their vCPUs need to be the same.\n" +
											"  - If the vCPUs of TiDB or TiKV component is 2 or 4, then the cluster does not support TiFlash.\n" +
											"  - Can not modify node_size of an existing cluster.",
										Required: true,
									},
									"storage_size_gib": schema.Int64Attribute{
										MarkdownDescription: "The storage size of a node in the cluster. You can get the minimum and maximum of storage size from the [tidbcloud_cluster_specs datasource](./cluster_specs.md).\n" +
											"  - Can not modify storage_size_gib of an existing cluster.",
										Required: true,
									},
									"node_quantity": schema.Int64Attribute{
										MarkdownDescription: "The number of nodes in the cluster. You can get the minimum and step of a node quantity from the [tidbcloud_cluster_specs datasource](./cluster_specs.md).",
										Required:            true,
									},
								},
							},
							"tiflash": schema.SingleNestedAttribute{
								Optional: true,
								Attributes: map[string]schema.Attribute{
									"node_size": schema.StringAttribute{
										MarkdownDescription: "The size of the TiFlash component in the cluster, You can get the available node size of each region from the [tidbcloud_cluster_specs datasource](./cluster_specs.md).\n" +
											"  - If the vCPUs of TiDB or TiKV component is 2 or 4, then their vCPUs need to be the same.\n" +
											"  - If the vCPUs of TiDB or TiKV component is 2 or 4, then the cluster does not support TiFlash.\n" +
											"  - Can not modify node_size of an existing cluster.",
										Required: true,
									},
									"storage_size_gib": schema.Int64Attribute{
										MarkdownDescription: "The storage size of a node in the cluster. You can get the minimum and maximum of storage size from the [tidbcloud_cluster_specs datasource](./cluster_specs.md).\n" +
											"  - Can not modify storage_size_gib of an existing cluster.",
										Required: true,
									},
									"node_quantity": schema.Int64Attribute{
										MarkdownDescription: "The number of nodes in the cluster. You can get the minimum and step of a node quantity from the [tidbcloud_cluster_specs datasource](./cluster_specs.md).",
										Required:            true,
									},
								},
							},
						},
					},
					"ip_access_list": schema.ListNestedAttribute{
						MarkdownDescription: "A list of IP addresses and Classless Inter-Domain Routing (CIDR) addresses that are allowed to access the TiDB Cloud cluster via [standard connection](https://docs.pingcap.com/tidbcloud/connect-to-tidb-cluster#connect-via-standard-connection).",
						Optional:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"cidr": schema.StringAttribute{
									MarkdownDescription: "The IP address or CIDR range that you want to add to the cluster's IP access list.",
									Required:            true,
								},
								"description": schema.StringAttribute{
									MarkdownDescription: "Description that explains the purpose of the entry.",
									Required:            true,
								},
							},
						},
					},
				},
			},
			"status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Enum: \"PENDING\" \"RUNNING\" \"FAILED\" \"SUCCESS\"\nThe status of the restore task.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"create_timestamp": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The creation time of the backup in UTC.The time format follows the ISO8601 standard, which is YYYY-MM-DD (year-month-day) + T +HH:MM:SS (hour-minutes-seconds) + Z. For example, 2020-01-01T00:00:00Z.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cluster": schema.SingleNestedAttribute{
				MarkdownDescription: "The information of the restored cluster. The restored cluster is the new cluster your backup data is restored to.",
				Computed:            true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						MarkdownDescription: "The ID of the restored cluster. The restored cluster is the new cluster your backup data is restored to.",
						Computed:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"name": schema.StringAttribute{
						MarkdownDescription: "The name of the restored cluster. The restored cluster is the new cluster your backup data is restored to.",
						Computed:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"status": schema.StringAttribute{
						MarkdownDescription: "The status of the restored cluster. Possible values are \"AVAILABLE\", \"CREATING\", \"MODIFYING\", \"PAUSED\", \"RESUMING\",\"UNAVAILABLE\", \"IMPORTING\" and \"CLEARED\".",
						Computed:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
				},
			},
			"error_message": schema.StringAttribute{
				MarkdownDescription: "The error message of restore if failed.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r restoreResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data restoreResourceData

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "create restore resource")
	createRestoreTaskOK, err := r.provider.client.CreateRestoreTask(restoreApi.NewCreateRestoreTaskParams().WithProjectID(data.ProjectId).WithBody(buildCreateRestoreTaskBody(data)))
	if err != nil {
		resp.Diagnostics.AddError("Create Error", fmt.Sprintf("Unable to call CreateRestoreTask, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "read restore resource")
	getRestoreTaskOK, err := r.provider.client.GetRestoreTask(restoreApi.NewGetRestoreTaskParams().WithProjectID(data.ProjectId).WithRestoreID(createRestoreTaskOK.Payload.ID))
	if err != nil {
		resp.Diagnostics.AddError("Create Error", fmt.Sprintf("Unable to call GetRestoreTask, got error: %s", err))
		return
	}
	refreshRestoreResourceData(getRestoreTaskOK.Payload, &data)

	// save into the Terraform state.
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func buildCreateRestoreTaskBody(data restoreResourceData) restoreApi.CreateRestoreTaskBody {
	tidb := data.Config.Components.TiDB
	tikv := data.Config.Components.TiKV
	tiflash := data.Config.Components.TiFlash
	// required
	rootPassWord := data.Config.RootPassword.ValueString()
	payload := restoreApi.CreateRestoreTaskBody{
		BackupID: &data.BackupId,
		Name:     &data.Name,
		Config: &restoreApi.CreateRestoreTaskParamsBodyConfig{
			RootPassword: &rootPassWord,
			Components: &restoreApi.CreateRestoreTaskParamsBodyConfigComponents{
				Tidb: &restoreApi.CreateRestoreTaskParamsBodyConfigComponentsTidb{
					NodeSize:     &tidb.NodeSize,
					NodeQuantity: &tidb.NodeQuantity,
				},
				Tikv: &restoreApi.CreateRestoreTaskParamsBodyConfigComponentsTikv{
					NodeSize:       &tikv.NodeSize,
					StorageSizeGib: &tikv.StorageSizeGib,
					NodeQuantity:   &tikv.NodeQuantity,
				},
			},
		},
	}

	// port is optional
	if !data.Config.Port.IsNull() && !data.Config.Port.IsUnknown() {
		payload.Config.Port = int32(data.Config.Port.ValueInt64())
	}
	// tiflash is optional
	if tiflash != nil {
		payload.Config.Components.Tiflash = &restoreApi.CreateRestoreTaskParamsBodyConfigComponentsTiflash{
			NodeSize:       &tiflash.NodeSize,
			StorageSizeGib: &tiflash.StorageSizeGib,
			NodeQuantity:   &tiflash.NodeQuantity,
		}
	}
	// ip_access_list is optional
	if data.Config.IPAccessList != nil {
		var IPAccessList []*restoreApi.CreateRestoreTaskParamsBodyConfigIPAccessListItems0
		for _, key := range data.Config.IPAccessList {
			cidr := key.CIDR
			IPAccessList = append(IPAccessList, &restoreApi.CreateRestoreTaskParamsBodyConfigIPAccessListItems0{
				Cidr:        &cidr,
				Description: key.Description,
			})
		}
		payload.Config.IPAccessList = IPAccessList
	}

	return payload
}

func (r restoreResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data restoreResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "read restore resource")
	getRestoreTaskOK, err := r.provider.client.GetRestoreTask(restoreApi.NewGetRestoreTaskParams().WithProjectID(data.ProjectId).WithRestoreID(data.RestoreId.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Read Error", fmt.Sprintf("Unable to call GetRestoreTask, got error: %s", err))
		return
	}

	refreshRestoreResourceData(getRestoreTaskOK.Payload, &data)

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func refreshRestoreResourceData(resp *restoreApi.GetRestoreTaskOKBody, data *restoreResourceData) {
	data.ClusterId = types.StringValue(resp.ClusterID)
	data.RestoreId = types.StringValue(resp.ID)
	data.CreateTimestamp = types.StringValue(resp.CreateTimestamp.String())
	data.Status = types.StringValue(resp.Status)
	if resp.Cluster != nil {
		data.Cluster = &cluster{
			Id:     resp.Cluster.ID,
			Name:   resp.Cluster.Name,
			Status: resp.Cluster.Status,
		}
	}
	data.ErrorMessage = types.StringValue(resp.ErrorMessage)
}

func (r restoreResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Unsupported", "restore can't be updated")
}

func (r restoreResource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddWarning("Unsupported", "restore can't be deleted")
}
