package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/juju/errors"
	"github.com/tidbcloud/tidbcloud-cli/pkg/tidbcloud/v1beta1/dedicated"
)

type dedicatedVpcPeeringsDataSourceData struct {
	VpcPeerings []VpcPeeringItem `tfsdk:"vpc_peerings"`
}

type VpcPeeringItem struct {
	VpcPeeringId              types.String `tfsdk:"vpc_peering_id"`
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

var _ datasource.DataSource = &dedicatedVpcPeeringsDataSource{}

type dedicatedVpcPeeringsDataSource struct {
	provider *tidbcloudProvider
}

func NewDedicatedVpcPeeringsDataSource() datasource.DataSource {
	return &dedicatedVpcPeeringsDataSource{}
}

func (d *dedicatedVpcPeeringsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dedicated_vpc_peerings"
}

func (d *dedicatedVpcPeeringsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	var ok bool
	if d.provider, ok = req.ProviderData.(*tidbcloudProvider); !ok {
		resp.Diagnostics.AddError("Internal provider error",
			fmt.Sprintf("Error in Configure: expected %T but got %T", tidbcloudProvider{}, req.ProviderData))
	}
}

func (d *dedicatedVpcPeeringsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "dedicated vpc peerings data source",
		Attributes: map[string]schema.Attribute{
			"vpc_peerings": schema.ListNestedAttribute{
				MarkdownDescription: "The vpc peerings.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
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
				},
			},
		},
	}
}

func (d *dedicatedVpcPeeringsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data dedicatedVpcPeeringsDataSourceData
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "read vpc peerings data source")
	vpcPeerings, err := d.retrieveVPCPeerings(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Read Error", fmt.Sprintf("Unable to call ListVPCPeerings, got error: %s", err))
		return
	}

	var items []VpcPeeringItem
	for _, vpcPeering := range vpcPeerings {
		labels, diag := types.MapValueFrom(ctx, types.StringType, *vpcPeering.Labels)
		if diag.HasError() {
			return
		}
		var awsVpcPeeringConnectionId types.String
		if vpcPeering.AwsVpcPeeringConnectionId.IsSet() {
			awsVpcPeeringConnectionId = types.StringValue(*vpcPeering.AwsVpcPeeringConnectionId.Get())
		}

		items = append(items, VpcPeeringItem{
			TiDBCloudRegionId:         types.StringValue(vpcPeering.TidbCloudRegionId),
			TiDBCloudCloudProvider:    types.StringValue(string(*vpcPeering.TidbCloudCloudProvider)),
			TiDBCloudAccountId:        types.StringValue(*vpcPeering.TidbCloudAccountId),
			TiDBCloudVpcId:            types.StringValue(*vpcPeering.TidbCloudVpcId),
			TiDBCloudVpcCidr:          types.StringValue(*vpcPeering.TidbCloudVpcCidr),
			CustomerRegionId:          types.StringValue(*vpcPeering.CustomerRegionId),
			CustomerAccountId:         types.StringValue(vpcPeering.CustomerAccountId),
			CustomerVpcId:             types.StringValue(vpcPeering.CustomerVpcId),
			CustomerVpcCidr:           types.StringValue(vpcPeering.CustomerVpcCidr),
			State:                     types.StringValue(string(*vpcPeering.State)),
			AWSVpcPeeringConnectionId: awsVpcPeeringConnectionId,
			Labels:                    labels,
		})
	}
	data.VpcPeerings = items

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (d dedicatedVpcPeeringsDataSource) retrieveVPCPeerings(ctx context.Context) ([]dedicated.Dedicatedv1beta1VpcPeering, error) {
	var items []dedicated.Dedicatedv1beta1VpcPeering
	pageSizeInt32 := int32(DefaultPageSize)
	var pageToken *string
	for {
		vpcPeerings, err := d.provider.DedicatedClient.ListVPCPeerings(ctx, &pageSizeInt32, pageToken)
		if err != nil {
			return nil, errors.Trace(err)
		}
		items = append(items, vpcPeerings.VpcPeerings...)

		pageToken = vpcPeerings.NextPageToken
		if IsNilOrEmpty(pageToken) {
			break
		}
	}
	return items, nil
}
