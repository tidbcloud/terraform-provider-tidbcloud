package provider

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"

	clusterApi "github.com/c4pt0r/go-tidbcloud-sdk-v1/client/cluster"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type clusterSpecsDataSourceData struct {
	Id    types.String      `tfsdk:"id"`
	Items []clusterSpecItem `tfsdk:"items"`
	Total types.Int64       `tfsdk:"total"`
}

type clusterSpecItem struct {
	ClusterType   string        `tfsdk:"cluster_type"`
	CloudProvider string        `tfsdk:"cloud_provider"`
	Region        string        `tfsdk:"region"`
	Tidb          []tidbSpec    `tfsdk:"tidb"`
	Tikv          []tikvSpec    `tfsdk:"tikv"`
	Tifalsh       []tiflashSpec `tfsdk:"tiflash"`
}

type tidbSpec struct {
	NodeSize          string            `tfsdk:"node_size"`
	NodeQuantityRange nodeQuantityRange `tfsdk:"node_quantity_range"`
}

type tikvSpec struct {
	NodeSize           string             `tfsdk:"node_size"`
	StorageSizeGiRange storageSizeGiRange `tfsdk:"storage_size_gib_range"`
	NodeQuantityRange  nodeQuantityRange  `tfsdk:"node_quantity_range"`
}

type tiflashSpec struct {
	NodeSize           string             `tfsdk:"node_size"`
	StorageSizeGiRange storageSizeGiRange `tfsdk:"storage_size_gib_range"`
	NodeQuantityRange  nodeQuantityRange  `tfsdk:"node_quantity_range"`
}

type nodeQuantityRange struct {
	Min  int32 `tfsdk:"min"`
	Step int32 `tfsdk:"step"`
}

type storageSizeGiRange struct {
	Min int32 `tfsdk:"min"`
	Max int32 `tfsdk:"max"`
}

// Ensure provider defined types fully satisfy framework interfaces
// Ensure provider defined types fully satisfy framework interfaces
var _ datasource.DataSource = &clusterSpecsDataSource{}

type clusterSpecsDataSource struct {
	provider *tidbcloudProvider
}

func NewClusterSpecsDataSource() datasource.DataSource {
	return &clusterSpecsDataSource{}
}

func (d *clusterSpecsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cluster_specs"
}

func (d *clusterSpecsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	var ok bool
	if d.provider, ok = req.ProviderData.(*tidbcloudProvider); !ok {
		resp.Diagnostics.AddError("Internal provider error",
			fmt.Sprintf("Error in Configure: expected %T but got %T", tidbcloudProvider{}, req.ProviderData))
	}
}

func (d *clusterSpecsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "cluster_specs data source",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "data source ID.",
				Computed:            true,
			},
			"total": schema.Int64Attribute{
				MarkdownDescription: "the total number of the spec.",
				Computed:            true,
			},
			"items": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"cluster_type": schema.StringAttribute{
							MarkdownDescription: "Enum: \"DEDICATED\" \"DEVELOPER\", The cluster type.",
							Computed:            true,
						},
						"cloud_provider": schema.StringAttribute{
							MarkdownDescription: "Enum: \"AWS\" \"GCP\", The cloud provider on which your TiDB cluster is hosted.",
							Computed:            true,
						},
						"region": schema.StringAttribute{
							MarkdownDescription: "the region value should match the cloud provider's region code.",
							Computed:            true,
						},
						"tidb": schema.ListNestedAttribute{
							MarkdownDescription: "The list of TiDB specifications in the region.",
							Computed:            true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"node_size": schema.StringAttribute{
										MarkdownDescription: "The size of the TiDB component in the cluster.",
										Computed:            true,
									},
									"node_quantity_range": schema.SingleNestedAttribute{
										MarkdownDescription: "The range and step of node quantity of the TiDB component in the cluster.",
										Computed:            true,
										Attributes: map[string]schema.Attribute{
											"min": schema.Int64Attribute{
												MarkdownDescription: "The minimum node quantity of the component in the cluster.",
												Computed:            true,
											},
											"step": schema.Int64Attribute{
												MarkdownDescription: "The step of node quantity of the component in the cluster.",
												Computed:            true,
											},
										},
									},
								},
							},
						},
						"tikv": schema.ListNestedAttribute{
							MarkdownDescription: "The list of TiKV specifications in the region.",
							Computed:            true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"node_size": schema.StringAttribute{
										MarkdownDescription: "The size of the TiKV component in the cluster.",
										Computed:            true,
									},
									"node_quantity_range": schema.SingleNestedAttribute{
										MarkdownDescription: "The range and step of node quantity of the TiKV component in the cluster.",
										Computed:            true,
										Attributes: map[string]schema.Attribute{
											"min": schema.Int64Attribute{
												MarkdownDescription: "The minimum node quantity of the component in the cluster.",
												Computed:            true,
											},
											"step": schema.Int64Attribute{
												MarkdownDescription: "The step of node quantity of the component in the cluster.",
												Computed:            true,
											},
										},
									},
									"storage_size_gib_range": schema.SingleNestedAttribute{
										MarkdownDescription: "The storage size range for each node of the TiKV component in the cluster.",
										Computed:            true,
										Attributes: map[string]schema.Attribute{
											"min": schema.Int64Attribute{
												MarkdownDescription: "The minimum storage size for each node of the component in the cluster.",
												Computed:            true,
											},
											"max": schema.Int64Attribute{
												MarkdownDescription: "The maximum storage size for each node of the component in the cluster.",
												Computed:            true,
											},
										},
									},
								},
							},
						},
						"tiflash": schema.ListNestedAttribute{
							MarkdownDescription: "The list of TiFlash specifications in the region.",
							Computed:            true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"node_size": schema.StringAttribute{
										MarkdownDescription: "The size of the TiFlash component in the cluster.",
										Computed:            true,
									},
									"node_quantity_range": schema.SingleNestedAttribute{
										MarkdownDescription: "The range and step of node quantity of the TiFlash component in the cluster.",
										Computed:            true,
										Attributes: map[string]schema.Attribute{
											"min": schema.Int64Attribute{
												MarkdownDescription: "The minimum node quantity of the component in the cluster.",
												Computed:            true,
											},
											"step": schema.Int64Attribute{
												MarkdownDescription: "The step of node quantity of the component in the cluster.",
												Computed:            true,
											},
										},
									},
									"storage_size_gib_range": schema.SingleNestedAttribute{
										MarkdownDescription: "The storage size range for each node of the TiFlash component in the cluster.",
										Computed:            true,
										Attributes: map[string]schema.Attribute{
											"min": schema.Int64Attribute{
												MarkdownDescription: "The minimum storage size for each node of the component in the cluster.",
												Computed:            true,
											},
											"max": schema.Int64Attribute{
												MarkdownDescription: "The maximum storage size for each node of the component in the cluster.",
												Computed:            true,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (d *clusterSpecsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data clusterSpecsDataSourceData
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "read cluster_specs data source")
	listProviderRegionsOK, err := d.provider.client.ListProviderRegions(clusterApi.NewListProviderRegionsParams())
	if err != nil {
		resp.Diagnostics.AddError("Read Error", fmt.Sprintf("Unable to call read specifications, got error: %s", err))
		return
	}

	var items []clusterSpecItem
	for _, key := range listProviderRegionsOK.Payload.Items {
		var tidbs []tidbSpec
		for _, tidb := range key.Tidb {
			tidbs = append(tidbs, tidbSpec{
				NodeSize: tidb.NodeSize,
				NodeQuantityRange: nodeQuantityRange{
					Min:  tidb.NodeQuantityRange.Min,
					Step: tidb.NodeQuantityRange.Step,
				},
			})
		}
		var tikvs []tikvSpec
		for _, tikv := range key.Tikv {
			tikvs = append(tikvs, tikvSpec{
				NodeSize: tikv.NodeSize,
				NodeQuantityRange: nodeQuantityRange{
					Min:  tikv.NodeQuantityRange.Min,
					Step: tikv.NodeQuantityRange.Step,
				},
				StorageSizeGiRange: storageSizeGiRange{
					Min: tikv.StorageSizeGibRange.Min,
					Max: tikv.StorageSizeGibRange.Max,
				},
			})
		}
		var tiflashs []tiflashSpec
		for _, tiflash := range key.Tiflash {
			tiflashs = append(tiflashs, tiflashSpec{
				NodeSize: tiflash.NodeSize,
				NodeQuantityRange: nodeQuantityRange{
					Min:  tiflash.NodeQuantityRange.Min,
					Step: tiflash.NodeQuantityRange.Step,
				},
				StorageSizeGiRange: storageSizeGiRange{
					Min: tiflash.StorageSizeGibRange.Min,
					Max: tiflash.StorageSizeGibRange.Max,
				},
			})
		}
		items = append(items, clusterSpecItem{
			ClusterType:   key.ClusterType,
			CloudProvider: key.CloudProvider,
			Region:        key.Region,
			Tidb:          tidbs,
			Tikv:          tikvs,
			Tifalsh:       tiflashs,
		})
	}

	data.Items = items
	data.Total = types.Int64Value(int64(len(items)))
	data.Id = types.StringValue(strconv.FormatInt(rand.Int63(), 10))

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}
