package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/revosai/terraform-provider-revos/internal/client"
)

// Ensure RevosProvider satisfies various provider interfaces.
var _ provider.Provider = &RevosProvider{}

// RevosProvider defines the provider implementation.
type RevosProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// RevosProviderModel describes the provider data model.
type RevosProviderModel struct {
	APIURL types.String `tfsdk:"api_url"`
	Token  types.String `tfsdk:"token"`
}

func New() provider.Provider {
	return &RevosProvider{
		version: "dev",
	}
}

func (p *RevosProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "revos"
	resp.Version = p.version
}

func (p *RevosProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"api_url": schema.StringAttribute{
				Optional:    true,
				Description: "The URL of the Revos API. Defaults to REVOSAI_API_URL environment variable.",
			},
			"token": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "The authentication token. Defaults to REVOSAI_TOKEN environment variable.",
			},
		},
	}
}

func (p *RevosProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data RevosProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	apiURL := os.Getenv("REVOSAI_API_URL")
	token := os.Getenv("REVOSAI_TOKEN")

	if !data.APIURL.IsNull() {
		apiURL = data.APIURL.ValueString()
	}

	if !data.Token.IsNull() {
		token = data.Token.ValueString()
	}

	if apiURL == "" {
		// Default to something if not set? Or error?
		// CLI usually has a config. The user can provide it.
		// If both env and config are missing, maybe default to localhost for dev or error.
		// Let's assume user provides it.
		// For now, let's not force error, client might handle empty URL or we can set a default.
		// But let's report error if empty.
		resp.Diagnostics.AddError("Missing API URL", "API URL must be configured via provider block or REVOSAI_API_URL")
	}

	if token == "" {
		resp.Diagnostics.AddError("Missing Token", "Token must be configured via provider block or REVOSAI_TOKEN")
	}

	if resp.Diagnostics.HasError() {
		return
	}

	c := client.NewClient(apiURL, token)

	resp.DataSourceData = c
	resp.ResourceData = c
}

func (p *RevosProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewOverlayResource,
	}
}

func (p *RevosProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}
