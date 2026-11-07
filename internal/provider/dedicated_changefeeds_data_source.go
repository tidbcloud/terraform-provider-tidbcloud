package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/juju/errors"
	"github.com/tidbcloud/terraform-provider-tidbcloud/tidbcloud"
)

type dedicatedChangefeedsDataSourceData struct {
	ClusterId      types.String     `tfsdk:"cluster_id"`
	DownstreamType types.String     `tfsdk:"downstream_type"`
	Changefeeds    []ChangefeedItem `tfsdk:"changefeeds"`
}

type ChangefeedItem struct {
	ChangefeedId        types.String `tfsdk:"changefeed_id"`
	Name                types.String `tfsdk:"name"`
	ReplicationCapacity types.String `tfsdk:"replication_capacity"`
	DownstreamType      types.String `tfsdk:"downstream_type"`
	State               types.String `tfsdk:"state"`
	CreateTime          types.String `tfsdk:"create_time"`
	CheckpointTso       types.String `tfsdk:"checkpoint_tso"`
	CheckpointTs        types.String `tfsdk:"checkpoint_ts"`
	Error               types.String `tfsdk:"error"`
}

var _ datasource.DataSource = &dedicatedChangefeedsDataSource{}

type dedicatedChangefeedsDataSource struct {
	provider *tidbcloudProvider
}

func NewDedicatedChangefeedsDataSource() datasource.DataSource {
	return &dedicatedChangefeedsDataSource{}
}

func (d *dedicatedChangefeedsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dedicated_changefeeds"
}

func (d *dedicatedChangefeedsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	var ok bool
	if d.provider, ok = req.ProviderData.(*tidbcloudProvider); !ok {
		resp.Diagnostics.AddError("Internal provider error",
			fmt.Sprintf("Error in Configure: expected %T but got %T", tidbcloudProvider{}, req.ProviderData))
	}
}

func (d *dedicatedChangefeedsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "dedicated changefeeds data source",
		Attributes: map[string]schema.Attribute{
			"cluster_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the cluster.",
				Required:            true,
			},
			"downstream_type": schema.StringAttribute{
				MarkdownDescription: "The downstream type to filter by. If specified, only changefeeds of the specified downstream type will be returned. Available values: `KAFKA`, `MYSQL`, `S3`, `GCS`, `AZURE_BLOB`.",
				Optional:            true,
			},
			"changefeeds": schema.ListNestedAttribute{
				MarkdownDescription: "The changefeeds. Use the `tidbcloud_dedicated_changefeed` data source to read the full configuration of a single changefeed.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"changefeed_id": schema.StringAttribute{
							MarkdownDescription: "The ID of the changefeed.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "The user-defined name of the changefeed.",
							Computed:            true,
						},
						"replication_capacity": schema.StringAttribute{
							MarkdownDescription: "The replication capacity (RCU) of the changefeed.",
							Computed:            true,
						},
						"downstream_type": schema.StringAttribute{
							MarkdownDescription: "The downstream type of the changefeed.",
							Computed:            true,
						},
						"state": schema.StringAttribute{
							MarkdownDescription: "The current state of the changefeed.",
							Computed:            true,
						},
						"create_time": schema.StringAttribute{
							MarkdownDescription: "The creation time of the changefeed.",
							Computed:            true,
						},
						"checkpoint_tso": schema.StringAttribute{
							MarkdownDescription: "The checkpoint TSO of the changefeed.",
							Computed:            true,
						},
						"checkpoint_ts": schema.StringAttribute{
							MarkdownDescription: "The checkpoint timestamp of the changefeed.",
							Computed:            true,
						},
						"error": schema.StringAttribute{
							MarkdownDescription: "The error message when the changefeed is in the `FAILED` or `ERROR` state.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *dedicatedChangefeedsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data dedicatedChangefeedsDataSourceData
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "read dedicated changefeeds data source")
	changefeeds, err := d.retrieveChangefeeds(ctx, data.ClusterId.ValueString(), data.DownstreamType.ValueStringPointer())
	if err != nil {
		resp.Diagnostics.AddError("Read Error", fmt.Sprintf("Unable to call ListChangefeeds, got error: %s", err))
		return
	}

	var items []ChangefeedItem
	for _, changefeed := range changefeeds {
		items = append(items, ChangefeedItem{
			ChangefeedId:        types.StringPointerValue(changefeed.Id),
			Name:                types.StringValue(changefeed.Name),
			ReplicationCapacity: types.StringValue(changefeed.ReplicationCapacity),
			DownstreamType:      types.StringValue(changefeed.DownstreamType),
			State:               types.StringPointerValue(changefeed.State),
			CreateTime:          types.StringPointerValue(changefeed.CreateTime),
			CheckpointTso:       types.StringPointerValue(changefeed.CheckpointTso),
			CheckpointTs:        types.StringPointerValue(changefeed.CheckpointTs),
			Error:               types.StringPointerValue(changefeed.Error),
		})
	}
	data.Changefeeds = items

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (d dedicatedChangefeedsDataSource) retrieveChangefeeds(ctx context.Context, clusterId string, downstreamType *string) ([]tidbcloud.Changefeed, error) {
	var items []tidbcloud.Changefeed
	pageSizeInt32 := int32(DefaultPageSize)
	var pageToken *string
	for {
		changefeeds, err := d.provider.DedicatedClient.ListChangefeeds(ctx, &tidbcloud.ListChangefeedsParams{
			ClusterId:      clusterId,
			DownstreamType: downstreamType,
			PageSize:       &pageSizeInt32,
			PageToken:      pageToken,
		})
		if err != nil {
			return nil, errors.Trace(err)
		}
		items = append(items, changefeeds.Changefeeds...)

		pageToken = changefeeds.NextPageToken
		if IsNilOrEmpty(pageToken) {
			break
		}
	}
	return items, nil
}
