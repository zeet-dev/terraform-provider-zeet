package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/zeet-dev/cli/pkg/api"
)

// Ensure ZeetProvider satisfies various provider interfaces.
var _ provider.Provider = &ZeetProvider{}

// ZeetProvider defines the provider implementation.
type ZeetProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// ZeetProviderModel describes the provider data model.
type ZeetProviderModel struct {
	ApiUrl types.String `tfsdk:"api_url"`
	Token  types.String `tfsdk:"token"`
}

func (p *ZeetProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "zeet"
	resp.Version = p.version
}

func (p *ZeetProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"api_url": schema.StringAttribute{
				MarkdownDescription: "The URL of the Zeet API Server.",
				Optional:            true,
			},
			"token": schema.StringAttribute{
				MarkdownDescription: "The Zeet API token.",
				Optional:            true,
			},
		},
	}
}

func (p *ZeetProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data ZeetProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Configuration values are now available.
	apiURL := os.Getenv("ZEET_API_URL")
	token := os.Getenv("ZEET_TOKEN")

	if !data.ApiUrl.IsNull() {
		apiURL = data.ApiUrl.ValueString()
	}
	if !data.Token.IsNull() {
		token = data.Token.ValueString()
	}

	client := api.New(
		apiURL,
		token,
		"terraform-"+p.version,
		false,
	)

	// Client configuration for data sources and resources
	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *ZeetProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewGroupResource,
	}
}

func (p *ZeetProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewGroupDataSource,
		NewTeamDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &ZeetProvider{
			version: version,
		}
	}
}
