package provider

import (
	"context"
	"fmt"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/tidbcloud/terraform-provider-tidbcloud/tidbcloud"
)

// Ensure the implementation satisfies the provider.Provider interface.
var _ provider.Provider = &tidbcloudProvider{}

// NewClient overrides the NewClientDelegate method for testing.
var NewClient = tidbcloud.NewClientDelegate

var NewDedicatedClient = tidbcloud.NewDedicatedClientDelegate

var NewServerlessClient = tidbcloud.NewServerlessClientDelegate

var NewIAMClient = tidbcloud.NewIAMClientDelegate

// provider satisfies the tfsdk.Provider interface and usually is included
// with all Resource and DataSource implementations.
type tidbcloudProvider struct {
	// client can contain the upstream provider SDK or HTTP client used to
	// communicate with the upstream service. Resource and DataSource
	// implementations can then make calls using this client.
	client tidbcloud.TiDBCloudClient

	DedicatedClient tidbcloud.TiDBCloudDedicatedClient

	ServerlessClient tidbcloud.TiDBCloudServerlessClient

	IAMClient tidbcloud.TiDBCloudIAMClient

	// configured is set to true at the end of the Configure method.
	// This can be used in Resource and DataSource implementations to verify
	// that the provider was previously configured.
	configured bool

	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string

	sync bool
}

// providerData can be used to store data from the Terraform configuration.
type providerData struct {
	PublicKey  types.String `tfsdk:"public_key"`
	PrivateKey types.String `tfsdk:"private_key"`
	Sync       types.Bool   `tfsdk:"sync"`
}

func (p *tidbcloudProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "tidbcloud"
	resp.Version = p.version
}

func (p *tidbcloudProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	// get providerData
	var data providerData
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	// User must provide a public_key to the provider
	var publicKey string
	if !IsKnown(data.PublicKey) {
		publicKey = os.Getenv(TiDBCloudPublicKey)
	} else {
		publicKey = data.PublicKey.ValueString()
	}
	if publicKey == "" {
		// Error vs warning - empty value must stop execution
		resp.Diagnostics.AddError(
			"Unable to find public_key",
			"public_key cannot be an empty string",
		)
		return
	}

	// User must provide a private_key to the provider
	var privateKey string
	if !IsKnown(data.PrivateKey) {
		privateKey = os.Getenv(TiDBCloudPrivateKey)
	} else {
		privateKey = data.PrivateKey.ValueString()
	}
	if privateKey == "" {
		// Error vs warning - empty value must stop execution
		resp.Diagnostics.AddError(
			"Unable to find private_key",
			"private_key cannot be an empty string",
		)
		return
	}

	// Create a new tidb client and set it to the provider client
	var host = tidbcloud.DefaultApiUrl
	if os.Getenv(TiDBCloudHost) != "" {
		host = os.Getenv(TiDBCloudHost)
	}
	c, err := NewClient(publicKey, privateKey, host, fmt.Sprintf("%s/%s", UserAgent, p.version))
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create client",
			"Unable to create TiDB client:\n\n"+err.Error(),
		)
		return
	}

	// Create a new dedicated client and set it to the provider dedicated client
	dc, err := NewDedicatedClient(publicKey, privateKey, os.Getenv(TiDBCloudDedicatedEndpoint), fmt.Sprintf("%s/%s", UserAgent, p.version))
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create client",
			"Unable to create TiDB Cloud Dedicated client:\n\n"+err.Error(),
		)
		return
	}

	// Create a new serverless client and set it to the provider serverless client
	sc, err := NewServerlessClient(publicKey, privateKey, os.Getenv(TiDBCloudServerlessEndpoint), fmt.Sprintf("%s/%s", UserAgent, p.version))
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create client",
			"Unable to create TiDB Cloud Serverless client:\n\n"+err.Error(),
		)
		return
	}

	ic, err := NewIAMClient(publicKey, privateKey, os.Getenv(TiDBCloudIAMEndpoint), fmt.Sprintf("%s/%s", UserAgent, p.version))
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create client",
			"Unable to create TiDB Cloud IAM client:\n\n"+err.Error(),
		)
	}
	// sync
	p.sync = data.Sync.ValueBool()
	p.client = c
	p.DedicatedClient = dc
	p.ServerlessClient = sc
	p.IAMClient = ic
	p.configured = true
	resp.ResourceData = p
	resp.DataSourceData = p
}

func (p *tidbcloudProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewClusterResource,
		NewBackupResource,
		NewRestoreResource,
		NewImportResource,
		
		// new version with openapi v1beta1
		NewSQLUserResource,

		NewDedicatedClusterResource,
		NewDedicatedNodeGroupResource,
		NewDedicatedPrivateEndpointConnectionResource,
		NewDedicatedNetworkContainerResource,
		NewDedicatedVpcPeeringResource,

		NewServerlessClusterResource,
		NewServerlessExportResource,
		NewServerlessBranchResource,
	}
}

func (p *tidbcloudProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewProjectsDataSource,
		NewClusterSpecsDataSource,
		NewBackupsDataSource,
		NewRestoresDataSource,
		NewClustersDataSource,

		// new version with openapi v1beta1
		NewSQLUsersDataSource,
		NewSQLUserDataSource,

		NewDedicatedRegionsDataSource,
		NewDedicatedRegionDataSource,
		NewDedicatedCloudProvidersDataSource,
		NewDedicatedClusterDataSource,
		NewDedicatedClustersDataSource,
		NewDedicatedNodeGroupDataSource,
		NewDedicatedNodeGroupsDataSource,
		NewDedicatedPrivateEndpointConnectionDataSource,
		NewDedicatedPrivateEndpointConnectionsDataSource,
		NewDedicatedNetworkContainerDataSource,
		NewDedicatedNetworkContainersDataSource,
		NewDedicatedVpcPeeringDataSource,
		NewDedicatedVpcPeeringsDataSource,

		NewServerlessClusterDataSource,
		NewServerlessClustersDataSource,
		NewServerlessRegionsDataSource,
		NewServerlessExportDataSource,
		NewServerlessExportsDataSource,
		NewServerlessBranchDataSource,
		NewServerlessBranchesDataSource,
	}
}

func (p *tidbcloudProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"public_key": schema.StringAttribute{
				MarkdownDescription: "Public Key",
				Optional:            true,
				Sensitive:           true,
			},
			"private_key": schema.StringAttribute{
				MarkdownDescription: "Private Key",
				Optional:            true,
				Sensitive:           true,
			},
			"sync": schema.BoolAttribute{
				MarkdownDescription: "Whether to create or update the cluster resource synchronously",
				Optional:            true,
				Sensitive:           false,
			},
		},
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &tidbcloudProvider{
			version: version,
		}
	}
}
