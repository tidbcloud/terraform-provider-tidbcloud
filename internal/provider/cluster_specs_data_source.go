package provider

import (
	"context"
	"fmt"
	clusterApi "github.com/c4pt0r/go-tidbcloud-sdk-v1/client/cluster"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"math/rand"
	"strconv"
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
var _ provider.DataSourceType = clusterSpecsDataSourceType{}
var _ datasource.DataSource = clusterSpecsDataSource{}

type clusterSpecsDataSourceType struct{}

func (t clusterSpecsDataSourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		MarkdownDescription: "cluster_specs data source",
		Attributes: map[string]tfsdk.Attribute{
			"id": {
				MarkdownDescription: "data source ID.",
				Computed:            true,
				Type:                types.StringType,
			},
			"total": {
				MarkdownDescription: "the total number of the spec.",
				Computed:            true,
				Type:                types.Int64Type,
			},
			"items": {
				Computed: true,
				Attributes: tfsdk.ListNestedAttributes(map[string]tfsdk.Attribute{
					"cluster_type": {
						MarkdownDescription: "Enum: \"DEDICATED\" \"DEVELOPER\", The cluster type.",
						Computed:            true,
						Type:                types.StringType,
					},
					"cloud_provider": {
						MarkdownDescription: "Enum: \"AWS\" \"GCP\", The cloud provider on which your TiDB cluster is hosted.",
						Computed:            true,
						Type:                types.StringType,
					},
					"region": {
						MarkdownDescription: "the region value should match the cloud provider's region code.",
						Computed:            true,
						Type:                types.StringType,
					},
					"tidb": {
						MarkdownDescription: "The list of TiDB specifications in the region.",
						Computed:            true,
						Attributes: tfsdk.ListNestedAttributes(map[string]tfsdk.Attribute{
							"node_size": {
								MarkdownDescription: "The size of the TiDB component in the cluster.",
								Computed:            true,
								Type:                types.StringType,
							},
							"node_quantity_range": {
								MarkdownDescription: "The range and step of node quantity of the TiDB component in the cluster.",
								Computed:            true,
								Attributes: tfsdk.SingleNestedAttributes(map[string]tfsdk.Attribute{
									"min": {
										MarkdownDescription: "The minimum node quantity of the component in the cluster.",
										Computed:            true,
										Type:                types.Int64Type,
									},
									"step": {
										MarkdownDescription: "The step of node quantity of the component in the cluster.",
										Computed:            true,
										Type:                types.Int64Type,
									},
								}),
							},
						}),
					},
					"tikv": {
						MarkdownDescription: "The list of TiKV specifications in the region.",
						Computed:            true,
						Attributes: tfsdk.ListNestedAttributes(map[string]tfsdk.Attribute{
							"node_size": {
								MarkdownDescription: "The size of the TiKV component in the cluster.",
								Computed:            true,
								Type:                types.StringType,
							},
							"node_quantity_range": {
								MarkdownDescription: "The range and step of node quantity of the TiKV component in the cluster.",
								Computed:            true,
								Attributes: tfsdk.SingleNestedAttributes(map[string]tfsdk.Attribute{
									"min": {
										MarkdownDescription: "The minimum node quantity of the component in the cluster.",
										Computed:            true,
										Type:                types.Int64Type,
									},
									"step": {
										MarkdownDescription: "The step of node quantity of the component in the cluster.",
										Computed:            true,
										Type:                types.Int64Type,
									},
								}),
							},
							"storage_size_gib_range": {
								MarkdownDescription: "The storage size range for each node of the TiKV component in the cluster.",
								Computed:            true,
								Attributes: tfsdk.SingleNestedAttributes(map[string]tfsdk.Attribute{
									"min": {
										MarkdownDescription: "The minimum storage size for each node of the component in the cluster.",
										Computed:            true,
										Type:                types.Int64Type,
									},
									"max": {
										MarkdownDescription: "The maximum storage size for each node of the component in the cluster.",
										Computed:            true,
										Type:                types.Int64Type,
									},
								}),
							},
						}),
					},
					"tiflash": {
						MarkdownDescription: "The list of TiFlash specifications in the region.",
						Computed:            true,
						Attributes: tfsdk.ListNestedAttributes(map[string]tfsdk.Attribute{
							"node_size": {
								MarkdownDescription: "The size of the TiFlash component in the cluster.",
								Computed:            true,
								Type:                types.StringType,
							},
							"node_quantity_range": {
								MarkdownDescription: "The range and step of node quantity of the TiFlash component in the cluster.",
								Computed:            true,
								Attributes: tfsdk.SingleNestedAttributes(map[string]tfsdk.Attribute{
									"min": {
										MarkdownDescription: "The minimum node quantity of the component in the cluster.",
										Computed:            true,
										Type:                types.Int64Type,
									},
									"step": {
										MarkdownDescription: "The step of node quantity of the component in the cluster.",
										Computed:            true,
										Type:                types.Int64Type,
									},
								}),
							},
							"storage_size_gib_range": {
								MarkdownDescription: "The storage size range for each node of the TiFlash component in the cluster.",
								Computed:            true,
								Attributes: tfsdk.SingleNestedAttributes(map[string]tfsdk.Attribute{
									"min": {
										MarkdownDescription: "The minimum storage size for each node of the component in the cluster.",
										Computed:            true,
										Type:                types.Int64Type,
									},
									"max": {
										MarkdownDescription: "The maximum storage size for each node of the component in the cluster.",
										Computed:            true,
										Type:                types.Int64Type,
									},
								}),
							},
						}),
					},
				}),
			},
		},
	}, nil
}

func (t clusterSpecsDataSourceType) NewDataSource(ctx context.Context, in provider.Provider) (datasource.DataSource, diag.Diagnostics) {
	provider, diags := convertProviderType(in)

	return clusterSpecsDataSource{
		provider: provider,
	}, diags
}

type clusterSpecsDataSource struct {
	provider tidbcloudProvider
}

func (d clusterSpecsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
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
	data.Total = types.Int64{Value: int64(len(items))}
	data.Id = types.String{Value: strconv.FormatInt(rand.Int63(), 10)}

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}
