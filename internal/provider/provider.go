// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	bitwarden "github.com/bitwarden/sdk-go"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"os"
)

var _ provider.Provider = &BitwardenSecretsProvider{}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &BitwardenSecretsProvider{
			version: version,
		}
	}
}

// BitwardenSecretsProvider is the provider implementation.
type BitwardenSecretsProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

type bitwardenProviderModel struct {
	ApiUrl      types.String `tfsdk:"api_url"`
	IdentityUrl types.String `tfsdk:"identity_url"`
	AccessToken types.String `tfsdk:"access_token"`
}

func (b BitwardenSecretsProvider) Metadata(_ context.Context, _ provider.MetadataRequest, response *provider.MetadataResponse) {
	response.TypeName = "bitwarden"
	response.Version = b.version
}

func (b BitwardenSecretsProvider) Schema(_ context.Context, _ provider.SchemaRequest, response *provider.SchemaResponse) {
	response.Schema = schema.Schema{
		Description: "Interact with Bitwarden Secrets Manager.",
		Attributes: map[string]schema.Attribute{
			"api_url": schema.StringAttribute{
				Description: "URI for Bitwarden Secrets Manager API. May also be provided via BITWARDEN_API_URL environment variable.",
				Optional:    true,
			},
			"identity_url": schema.StringAttribute{
				Description: "Username for Bitwarden Secrets Manager API. May also be provided via BITWARDEN_IDENTITY_URL environment variable.",
				Optional:    true,
			},
			"access_token": schema.StringAttribute{
				Description: "Password for Bitwarden Secrets Manager API. May also be provided via BITWARDEN_ACCESS_TOKEN environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
		},
	}
}

func (b BitwardenSecretsProvider) Configure(ctx context.Context, request provider.ConfigureRequest, response *provider.ConfigureResponse) {
	tflog.Info(ctx, "Configuring Bitwarden client")

	var config bitwardenProviderModel

	diags := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	if config.ApiUrl.IsUnknown() {
		response.Diagnostics.AddAttributeError(
			path.Root("api_url"),
			"Unknown Bitwarden API url",
			"The provider cannot create the Bitwarden client as there is an unknown configuration value for the Bitwarden API url. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the BW_API_URL environment variable.",
		)
	}
	if config.IdentityUrl.IsUnknown() {
		response.Diagnostics.AddAttributeError(
			path.Root("Identity_url"),
			"Unknown Bitwarden Identity url",
			"The provider cannot create the Bitwarden client as there is an unknown configuration value for the Bitwarden Identity url. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the BW_IDENTITY_URL environment variable.",
		)
	}
	if config.AccessToken.IsUnknown() {
		response.Diagnostics.AddAttributeError(
			path.Root("access_token"),
			"Unknown Bitwarden access token",
			"The provider cannot create the Bitwarden client as there is an unknown configuration value for the Bitwarden access token. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the BW_ACCESS_TOKEN environment variable.",
		)
	}

	if response.Diagnostics.HasError() {
		return
	}

	// Default values to environment variables, but override
	// with Terraform configuration value if set.

	apiUrl := os.Getenv("BW_API_URL")
	identityUrl := os.Getenv("BW_IDENTITY_URL")
	accessToken := os.Getenv("BW_ACCESS_TOKEN")

	if !config.ApiUrl.IsNull() {
		apiUrl = config.ApiUrl.ValueString()
	} else {
		apiUrl = "https://api.bitwarden.com"
	}

	if !config.IdentityUrl.IsNull() {
		identityUrl = config.IdentityUrl.ValueString()
	} else {
		identityUrl = "https://identity.bitwarden.com/connect/token"
	}

	if !config.AccessToken.IsNull() {
		accessToken = config.AccessToken.ValueString()
	}

	if accessToken == "" {
		response.Diagnostics.AddAttributeError(
			path.Root("access_token"),
			"Missing Bitwarden access token",
			"The provider cannot create the Bitwarden client as there is a missing or empty value for the access token."+
				"Set the access token in the configuration or use the BW_ACCESS_TOKEN environment variable.",
		)
	}

	if response.Diagnostics.HasError() {
		return
	}

	if apiUrl != "https://api.bitwarden.com" || apiUrl != "https://api.bitwarden.eu" {
		// as specified in the bitwarden doc https://bitwarden.com/help/public-api/#endpoints
		apiUrl = fmt.Sprintf("%s/api", apiUrl)
	}

	// as specified in the bitwarden doc https://bitwarden.com/help/public-api/#endpoints
	if identityUrl == "https://identity.bitwarden.com" || identityUrl == "https://identity.bitwarden.eu" {
		identityUrl = fmt.Sprintf("%s/connect/token", identityUrl)
	} else {
		identityUrl = fmt.Sprintf("%s/identity/connect/token", identityUrl)
	}

	ctx = tflog.SetField(ctx, "bw_api_url", apiUrl)
	ctx = tflog.SetField(ctx, "bw_identity_url", identityUrl)
	ctx = tflog.SetField(ctx, "bw_access_token", accessToken)
	ctx = tflog.MaskFieldValuesWithFieldKeys(ctx, "bw_access_token")

	tflog.Debug(ctx, "Creating HashiCups client")
	bitwardenClient, err := bitwarden.NewBitwardenClient(&apiUrl, &identityUrl)

	if err != nil {
		response.Diagnostics.AddError(
			"Error while creating the bitwarden client",
			"validate the api and identity url are correct: "+err.Error(),
		)
	}

	err = bitwardenClient.AccessTokenLogin(accessToken, nil)

	if err != nil {
		response.Diagnostics.AddError(
			"Unable to login to Bitwarden Secrets Manager",
			"Either the access token is not valid or there is some communication issues: "+err.Error(),
		)
	}

	response.DataSourceData = bitwardenClient
	response.ResourceData = bitwardenClient

	tflog.Info(ctx, "Configured bitwarden client", map[string]any{"success": true})
}

func (b BitwardenSecretsProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewProjectDataSource,
		NewSecretDataSource,
	}
}

func (b BitwardenSecretsProvider) Resources(ctx context.Context) []func() resource.Resource {
	// Resources defines the resources implemented in the provider.
	return []func() resource.Resource{
		NewSecretResource,
		NewSecretResource,
	}
}
