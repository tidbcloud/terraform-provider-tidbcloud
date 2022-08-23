package provider

import (
	"context"
	"fmt"
	"github.com/tidbcloud/terraform-provider-tidbcloud/tidbcloud"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type restoreResourceData struct {
	ClusterId       types.String  `tfsdk:"cluster_id"`
	RestoreId       types.String  `tfsdk:"id"`
	ProjectId       string        `tfsdk:"project_id"`
	Name            string        `tfsdk:"name"`
	BackupId        string        `tfsdk:"backup_id"`
	Config          ClusterConfig `tfsdk:"config"`
	CreateTimestamp types.String  `tfsdk:"create_timestamp"`
	Status          types.String  `tfsdk:"status"`
	Cluster         *cluster      `tfsdk:"cluster"`
	ErrorMessage    types.String  `tfsdk:"error_message"`
}

type cluster struct {
	Id     string `tfsdk:"id"`
	Name   string `tfsdk:"name"`
	Status string `tfsdk:"status"`
}

// Ensure provider defined types fully satisfy framework interfaces
var _ provider.ResourceType = restoreResourceType{}
var _ resource.Resource = restoreResource{}
var _ resource.ResourceWithImportState = restoreResource{}

type restoreResourceType struct{}

func (t restoreResourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		MarkdownDescription: "restore resource",
		Attributes: map[string]tfsdk.Attribute{
			"cluster_id": {
				MarkdownDescription: "The ID of the cluster",
				Computed:            true,
				Type:                types.StringType,
			},
			"id": {
				MarkdownDescription: "The ID of the restore",
				Computed:            true,
				Type:                types.StringType,
			},
			"project_id": {
				MarkdownDescription: "The ID of the project. You can get the project ID from [tidbcloud_project datasource](../project).",
				Required:            true,
				Type:                types.StringType,
			},
			"name": {
				MarkdownDescription: "The name of the restore",
				Required:            true,
				Type:                types.StringType,
			},
			"backup_id": {
				Required:            true,
				MarkdownDescription: "The ID of the backup",
				Type:                types.StringType,
			},
			"config": {
				MarkdownDescription: "The configuration of the cluster",
				Required:            true,
				Attributes: tfsdk.SingleNestedAttributes(map[string]tfsdk.Attribute{
					"root_password": {
						MarkdownDescription: "The root password to access the cluster. It must be 8-64 characters.",
						Required:            true,
						Type:                types.StringType,
					},
					"port": {
						MarkdownDescription: "The TiDB port for connection. The port must be in the range of 1024-65535 except 10080, 4000 in default.\n" +
							"  - For a Developer Tier cluster, only port 4000 is available.",
						Optional: true,
						Computed: true,
						Type:     types.Int64Type,
						PlanModifiers: tfsdk.AttributePlanModifiers{
							resource.UseStateForUnknown(),
						},
					},
					"components": {
						MarkdownDescription: "The components of the cluster.\n" +
							"  - For a Developer Tier cluster, the components value can not be set." +
							"  - For a Dedicated Tier cluster, the components value must be set.",
						Required: true,
						Attributes: tfsdk.SingleNestedAttributes(map[string]tfsdk.Attribute{
							"tidb": {
								Required: true,
								Attributes: tfsdk.SingleNestedAttributes(map[string]tfsdk.Attribute{
									"node_size": {
										MarkdownDescription: "The size of the TiDB component in the cluster, You can get the available node size of each region from the [tidbcloud_cluster_spec datasource](./cluster_spec.md).\n" +
											"  - If the vCPUs of TiDB or TiKV component is 2 or 4, then their vCPUs need to be the same.\n" +
											"  - If the vCPUs of TiDB or TiKV component is 2 or 4, then the cluster does not support TiFlash.\n" +
											"  - Can not modify node_size of an existing cluster.",
										Required: true,
										Type:     types.StringType,
									},
									"node_quantity": {
										MarkdownDescription: "The number of nodes in the cluster. You can get the minimum and step of a node quantity from the [tidbcloud_cluster_spec datasource](./cluster_spec.md).",
										Required:            true,
										Type:                types.Int64Type,
									},
								}),
							},
							"tikv": {
								Required: true,
								Attributes: tfsdk.SingleNestedAttributes(map[string]tfsdk.Attribute{
									"node_size": {
										MarkdownDescription: "The size of the TiKV component in the cluster, You can get the available node size of each region from the [tidbcloud_cluster_spec datasource](./cluster_spec.md).\n" +
											"  - If the vCPUs of TiDB or TiKV component is 2 or 4, then their vCPUs need to be the same.\n" +
											"  - If the vCPUs of TiDB or TiKV component is 2 or 4, then the cluster does not support TiFlash.\n" +
											"  - Can not modify node_size of an existing cluster.",
										Required: true,
										Type:     types.StringType,
									},
									"storage_size_gib": {
										MarkdownDescription: "The storage size of a node in the cluster. You can get the minimum and maximum of storage size from the [tidbcloud_cluster_spec datasource](./cluster_spec.md).\n" +
											"  - Can not modify storage_size_gib of an existing cluster.",
										Required: true,
										Type:     types.Int64Type,
									},
									"node_quantity": {
										MarkdownDescription: "The number of nodes in the cluster. You can get the minimum and step of a node quantity from the [tidbcloud_cluster_spec datasource](./cluster_spec.md).",
										Required:            true,
										Type:                types.Int64Type,
									},
								}),
							},
							"tiflash": {
								Optional: true,
								Attributes: tfsdk.SingleNestedAttributes(map[string]tfsdk.Attribute{
									"node_size": {
										MarkdownDescription: "The size of the TiFlash component in the cluster, You can get the available node size of each region from the [tidbcloud_cluster_spec datasource](./cluster_spec.md).\n" +
											"  - If the vCPUs of TiDB or TiKV component is 2 or 4, then their vCPUs need to be the same.\n" +
											"  - If the vCPUs of TiDB or TiKV component is 2 or 4, then the cluster does not support TiFlash.\n" +
											"  - Can not modify node_size of an existing cluster.",
										Required: true,
										Type:     types.StringType,
									},
									"storage_size_gib": {
										MarkdownDescription: "The storage size of a node in the cluster. You can get the minimum and maximum of storage size from the [tidbcloud_cluster_spec datasource](./cluster_spec.md).\n" +
											"  - Can not modify storage_size_gib of an existing cluster.",
										Required: true,
										Type:     types.Int64Type,
									},
									"node_quantity": {
										MarkdownDescription: "The number of nodes in the cluster. You can get the minimum and step of a node quantity from the [tidbcloud_cluster_spec datasource](./cluster_spec.md).",
										Required:            true,
										Type:                types.Int64Type,
									},
								}),
							},
						}),
					},
					"ip_access_list": {
						MarkdownDescription: "A list of IP addresses and Classless Inter-Domain Routing (CIDR) addresses that are allowed to access the TiDB Cloud cluster via [standard connection](https://docs.pingcap.com/tidbcloud/connect-to-tidb-cluster#connect-via-standard-connection).",
						Optional:            true,
						Attributes: tfsdk.ListNestedAttributes(map[string]tfsdk.Attribute{
							"cidr": {
								MarkdownDescription: "The IP address or CIDR range that you want to add to the cluster's IP access list.",
								Required:            true,
								Type:                types.StringType,
							},
							"description": {
								MarkdownDescription: "Description that explains the purpose of the entry.",
								Required:            true,
								Type:                types.StringType,
							},
						}),
					},
				}),
			},
			"status": {
				Computed:            true,
				MarkdownDescription: "Enum: \"PENDING\" \"RUNNING\" \"FAILED\" \"SUCCESS\"\nThe status of the restore task.",
				Type:                types.StringType,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					resource.UseStateForUnknown(),
				},
			},
			"create_timestamp": {
				Computed:            true,
				MarkdownDescription: "The creation time of the backup in UTC.The time format follows the ISO8601 standard, which is YYYY-MM-DD (year-month-day) + T +HH:MM:SS (hour-minutes-seconds) + Z. For example, 2020-01-01T00:00:00Z.",
				Type:                types.StringType,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					resource.UseStateForUnknown(),
				},
			},
			"cluster": {
				MarkdownDescription: "The information of the restored cluster. The restored cluster is the new cluster your backup data is restored to.",
				Computed:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					resource.UseStateForUnknown(),
				},
				Attributes: tfsdk.SingleNestedAttributes(map[string]tfsdk.Attribute{
					"id": {
						MarkdownDescription: "The ID of the restored cluster. The restored cluster is the new cluster your backup data is restored to.",
						Computed:            true,
						Type:                types.StringType,
						PlanModifiers: tfsdk.AttributePlanModifiers{
							resource.UseStateForUnknown(),
						},
					},
					"name": {
						MarkdownDescription: "The name of the restored cluster. The restored cluster is the new cluster your backup data is restored to.",
						Computed:            true,
						Type:                types.StringType,
						PlanModifiers: tfsdk.AttributePlanModifiers{
							resource.UseStateForUnknown(),
						},
					},
					"status": {
						MarkdownDescription: "The status of the restored cluster. Possible values are \"AVAILABLE\", \"CREATING\", \"MODIFYING\", \"PAUSED\", \"RESUMING\", and \"CLEARED\".",
						Computed:            true,
						Type:                types.StringType,
						PlanModifiers: tfsdk.AttributePlanModifiers{
							resource.UseStateForUnknown(),
						},
					},
				}),
			},
			"error_message": {
				MarkdownDescription: "The error message of restore if failed.",
				Computed:            true,
				Type:                types.StringType,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					resource.UseStateForUnknown(),
				},
			},
		},
	}, nil
}

func (t restoreResourceType) NewResource(ctx context.Context, in provider.Provider) (resource.Resource, diag.Diagnostics) {
	provider, diags := convertProviderType(in)

	return restoreResource{
		provider: provider,
	}, diags
}

type restoreResource struct {
	provider tidbcloudProvider
}

func (r restoreResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data restoreResourceData

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "create restore resource")

	createRestoreTaskResp, err := r.provider.client.CreateRestoreTask(data.ProjectId, buildCreateRestoreTaskReq(data))
	if err != nil {
		resp.Diagnostics.AddError("Create Error", fmt.Sprintf("Unable to call CreateRestoreTask, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "read restore resource")
	getRestoreTaskResp, err := r.provider.client.GetRestoreTask(data.ProjectId, createRestoreTaskResp.Id)
	if err != nil {
		resp.Diagnostics.AddError("Create Error", fmt.Sprintf("Unable to call GetRestoreTask, got error: %s", err))
		return
	}
	refreshRestoreResourceData(getRestoreTaskResp, &data)

	// save into the Terraform state.
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func buildCreateRestoreTaskReq(data restoreResourceData) tidbcloud.CreateRestoreTaskReq {
	tidb := data.Config.Components.TiDB
	tikv := data.Config.Components.TiKV
	tiflash := data.Config.Components.TiFlash
	// required
	payload := tidbcloud.CreateRestoreTaskReq{
		BackupId: data.BackupId,
		Name:     data.Name,
		Config: tidbcloud.ClusterConfig{
			RootPassword: data.Config.RootPassword.Value,
			Components: tidbcloud.Components{
				TiDB: tidbcloud.ComponentTiDB{
					NodeSize:     tidb.NodeSize,
					NodeQuantity: tidb.NodeQuantity,
				},
				TiKV: tidbcloud.ComponentTiKV{
					NodeSize:       tikv.NodeSize,
					StorageSizeGib: tikv.StorageSizeGib,
					NodeQuantity:   tikv.NodeQuantity,
				},
			},
		},
	}

	// port is optional
	if !data.Config.Port.IsNull() && !data.Config.Port.IsUnknown() {
		payload.Config.Port = int(data.Config.Port.Value)
	}
	// tiflash is optional
	if tiflash != nil {
		payload.Config.Components.TiFlash = &tidbcloud.ComponentTiFlash{
			NodeSize:       tiflash.NodeSize,
			StorageSizeGib: tiflash.StorageSizeGib,
			NodeQuantity:   tiflash.NodeQuantity,
		}
	}
	// ip_access_list  is optional
	if data.Config.IPAccessList != nil {
		var IPAccessList []tidbcloud.IPAccess
		for _, key := range data.Config.IPAccessList {
			IPAccessList = append(IPAccessList, tidbcloud.IPAccess{
				CIDR:        key.CIDR,
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
	getRestoreTaskResp, err := r.provider.client.GetRestoreTask(data.ProjectId, data.RestoreId.Value)
	if err != nil {
		resp.Diagnostics.AddError("Read Error", fmt.Sprintf("Unable to call GetRestoreTask, got error: %s", err))
		return
	}

	refreshRestoreResourceData(getRestoreTaskResp, &data)

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func refreshRestoreResourceData(resp *tidbcloud.GetRestoreTaskResp, data *restoreResourceData) {
	data.ClusterId = types.String{Value: resp.ClusterId}
	data.RestoreId = types.String{Value: resp.Id}
	data.CreateTimestamp = types.String{Value: resp.CreateTimestamp}
	data.Status = types.String{Value: resp.Status}
	data.Cluster = &cluster{
		Id:     resp.Cluster.Id,
		Name:   resp.Cluster.Name,
		Status: resp.Cluster.Status,
	}
	data.ErrorMessage = types.String{Value: resp.ErrorMessage}
}

func (r restoreResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Unsupported", fmt.Sprintf("restore can't be updated"))
}

func (r restoreResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddError("Unsupported", fmt.Sprintf("restore can't be delete"))
}

func (r restoreResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
}
