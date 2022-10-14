package provider

import (
	"context"
	"fmt"
	"os"

	"github.com/tidbcloud/terraform-provider-tidbcloud/tidbcloud"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces
var _ provider.Provider = &tidbcloudProvider{}

// provider satisfies the tfsdk.Provider interface and usually is included
// with all Resource and DataSource implementations.
type tidbcloudProvider struct {
	// client can contain the upstream provider SDK or HTTP client used to
	// communicate with the upstream service. Resource and DataSource
	// implementations can then make calls using this client.
	client *tidbcloud.TiDBCloudClient

	// configured is set to true at the end of the Configure method.
	// This can be used in Resource and DataSource implementations to verify
	// that the provider was previously configured.
	configured bool

	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// providerData can be used to store data from the Terraform configuration.
type providerData struct {
	PublicKey  types.String `tfsdk:"public_key"`
	PrivateKey types.String `tfsdk:"private_key"`
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
	if data.PublicKey.Unknown {
		// Cannot connect to client with an unknown value
		resp.Diagnostics.AddWarning(
			"Unable to create client",
			"Cannot use unknown value as public_key",
		)
		return
	}

	if data.PublicKey.Null {
		publicKey = os.Getenv("TIDBCLOUD_PUBLIC_KEY")
	} else {
		publicKey = data.PublicKey.Value
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
	if data.PrivateKey.Unknown {
		// Cannot connect to client with an unknown value
		resp.Diagnostics.AddError(
			"Unable to create client",
			"Cannot use unknown value as private_key",
		)
		return
	}

	if data.PrivateKey.Null {
		privateKey = os.Getenv("TIDBCLOUD_PRIVATE_KEY")
	} else {
		privateKey = data.PrivateKey.Value
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
	c, err := tidbcloud.NewTiDBCloudClient(publicKey, privateKey)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create client",
			"Unable to create tidb client:\n\n"+err.Error(),
		)
		return
	}

	p.client = c
	p.configured = true
}

func (p *tidbcloudProvider) GetResources(ctx context.Context) (map[string]provider.ResourceType, diag.Diagnostics) {
	return map[string]provider.ResourceType{
		"tidbcloud_cluster": clusterResourceType{},
		"tidbcloud_backup":  backupResourceType{},
		"tidbcloud_restore": restoreResourceType{},
	}, nil
}

func (p *tidbcloudProvider) GetDataSources(ctx context.Context) (map[string]provider.DataSourceType, diag.Diagnostics) {
	return map[string]provider.DataSourceType{
		"tidbcloud_project":      projectDataSourceType{},
		"tidbcloud_cluster_spec": clusterSpecDataSourceType{},
		"tidbcloud_backup":       backupDataSourceType{},
		"tidbcloud_restore":      restoreDataSourceType{},
	}, nil
}

func (p *tidbcloudProvider) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"public_key": {
				MarkdownDescription: "Public Key",
				Type:                types.StringType,
				Optional:            true,
				Sensitive:           true,
			},
			"private_key": {
				MarkdownDescription: "Private Key",
				Type:                types.StringType,
				Optional:            true,
				Sensitive:           true,
			},
		},
	}, nil
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &tidbcloudProvider{
			version: version,
		}
	}
}

// convertProviderType is a helper function for NewResource and NewDataSource
// implementations to associate the concrete provider type. Alternatively,
// this helper can be skipped and the provider type can be directly type
// asserted (e.g. provider: in.(*scaffoldingProvider)), however using this can prevent
// potential panics.
func convertProviderType(in provider.Provider) (tidbcloudProvider, diag.Diagnostics) {
	var diags diag.Diagnostics

	p, ok := in.(*tidbcloudProvider)

	if !ok {
		diags.AddError(
			"Unexpected Provider Instance Type",
			fmt.Sprintf("While creating the data source or resource, an unexpected provider type (%T) was received. This is always a bug in the provider code and should be reported to the provider developers.", p),
		)
		return tidbcloudProvider{}, diags
	}

	if p == nil {
		diags.AddError(
			"Unexpected Provider Instance Type",
			"While creating the data source or resource, an unexpected empty provider instance was received. This is always a bug in the provider code and should be reported to the provider developers.",
		)
		return tidbcloudProvider{}, diags
	}

	return *p, diags
}
