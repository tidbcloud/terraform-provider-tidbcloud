package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type dedicatedVpcPeeringDataSourceData struct {
	VpcPeeringId types.String `tfsdk:"vpc_peering_id"`
	TiDBCloudRegionId         types.String `tfsdk:"tidb_cloud_region_id"`
	TiDBCloudCloudProvider    types.String `tfsdk:"tidb_cloud_cloud_provider"`
	TiDBCloudAccountId        types.String `tfsdk:"tidb_cloud_account_id"`
	TiDBCloudVpcId            types.String `tfsdk:"tidb_cloud_vpc_id"`
	TiDBCloudVpcCidr          types.String `tfsdk:"tidb_cloud_vpc_cidr"`
	CustomerRegionId          types.String `tfsdk:"customer_region_id"`
	CustomerAccountId         types.String `tfsdk:"customer_account_id"`
	CustomerVpcId             types.String `tfsdk:"customer_vpc_id"`
	CustomerVpcCidr           types.String `tfsdk:"customer_vpc_cidr"`
	State                     types.String `tfsdk:"state"`
	AWSVpcPeeringConnectionId types.String `tfsdk:"aws_vpc_peering_connection_id"`
	Labels                    types.Map    `tfsdk:"labels"`
}

var _ datasource.DataSource = &dedicatedVpcPeeringDataSource{}

type dedicatedVpcPeeringDataSource struct {
	provider *tidbcloudProvider
}

func NewDedicatedVpcPeeringDataSource() datasource.DataSource {
	return &dedicatedVpcPeeringDataSource{}
}

func (d *dedicatedVpcPeeringDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dedicated_vpc_peering"
}

func (d *dedicatedVpcPeeringDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	var ok bool
	if d.provider, ok = req.ProviderData.(*tidbcloudProvider); !ok {
		resp.Diagnostics.AddError("Internal provider error",
			fmt.Sprintf("Error in Configure: expected %T but got %T", tidbcloudProvider{}, req.ProviderData))
	}
}

func (d *dedicatedVpcPeeringDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "dedicated vpc peering data source",
		Attributes: map[string]schema.Attribute{
			"vpc_peering_id": schema.StringAttribute{
				Description: "The ID of the VPC Peering",
				Required:    true,
			},
			"tidb_cloud_region_id": schema.StringAttribute{
				Description: "The region ID of the TiDB Cloud",
				Computed:    true,
			},
			"tidb_cloud_cloud_provider": schema.StringAttribute{
				Description: "The cloud provider of the TiDB Cloud",
				Computed:    true,
			},
			"tidb_cloud_account_id": schema.StringAttribute{
				Description: "The account ID of the TiDB Cloud",
				Computed:    true,
			},
			"tidb_cloud_vpc_id": schema.StringAttribute{
				Description: "The VPC ID of the TiDB Cloud",
				Computed:    true,
			},
			"tidb_cloud_vpc_cidr": schema.StringAttribute{
				Description: "The VPC CIDR of the TiDB Cloud",
				Computed:    true,
			},
			"customer_region_id": schema.StringAttribute{
				Description: "The region ID of the AWS VPC",
				Computed:    true,
			},
			"customer_account_id": schema.StringAttribute{
				Description: "The account ID of the AWS VPC",
				Computed:    true,
			},
			"customer_vpc_id": schema.StringAttribute{
				Description: "The ID of the AWS VPC",
				Computed:    true,
			},
			"customer_vpc_cidr": schema.StringAttribute{
				Description: "The VPC CIDR of the AWS VPC",
				Computed:    true,
			},
			"state": schema.StringAttribute{
				Description: "The state of the VPC Peering",
				Computed:    true,
			},
			"aws_vpc_peering_connection_id": schema.StringAttribute{
				Description: "The ID of the AWS VPC Peering Connection",
				Computed:    true,
			},
			"labels": schema.MapAttribute{
				Description: "The labels for the vpc peering",
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (d *dedicatedVpcPeeringDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data dedicatedVpcPeeringDataSourceData
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "read  vpc peering data source")
	VpcPeering, err := d.provider.DedicatedClient.GetVPCPeering(ctx, data.VpcPeeringId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Read Error", fmt.Sprintf("Unable to call GetVPCPeering, got error: %s", err))
		return
	}

	labels, diag := types.MapValueFrom(ctx, types.StringType, *VpcPeering.Labels)
	if diag.HasError() {
		return
	}
	if VpcPeering.AwsVpcPeeringConnectionId.IsSet() {
		data.AWSVpcPeeringConnectionId = types.StringValue(*VpcPeering.AwsVpcPeeringConnectionId.Get())
	}
	data.TiDBCloudRegionId = types.StringValue(VpcPeering.TidbCloudRegionId)
	data.TiDBCloudCloudProvider = types.StringValue(string(*VpcPeering.TidbCloudCloudProvider))
	data.TiDBCloudAccountId = types.StringValue(*VpcPeering.TidbCloudAccountId)
	data.TiDBCloudVpcId = types.StringValue(*VpcPeering.TidbCloudVpcId)
	data.TiDBCloudVpcCidr = types.StringValue(*VpcPeering.TidbCloudVpcCidr)
	data.CustomerRegionId = types.StringValue(*VpcPeering.CustomerRegionId)
	data.CustomerAccountId = types.StringValue(VpcPeering.CustomerAccountId)
	data.CustomerVpcId = types.StringValue(VpcPeering.CustomerVpcId)
	data.CustomerVpcCidr = types.StringValue(VpcPeering.CustomerVpcCidr)
	data.State = types.StringValue(string(*VpcPeering.State))
	data.Labels = labels

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}
