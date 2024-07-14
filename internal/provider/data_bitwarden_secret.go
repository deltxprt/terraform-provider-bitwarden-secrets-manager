// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	bitwarden "github.com/bitwarden/sdk-go"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &secretDataSource{}
	_ datasource.DataSourceWithConfigure = &secretDataSource{}
)

// NewSecretDataSource is a helper function to simplify the provider implementation.
func NewSecretDataSource() datasource.DataSource {
	return &secretDataSource{}
}

// secretDataSource is the data source implementation.
type secretDataSource struct {
	client bitwarden.BitwardenClientInterface
}

// secretDataSourceModel maps the data source schema data.
type secretDataSourceModel struct {
	Secrets []secretModel `tfsdk:"secrets"`
	ID      types.String  `tfsdk:"id"`
}

type secretModel struct {
	Id             types.String `tfsdk:"id"`
	Key            types.String `tfsdk:"key"`
	Value          types.String `tfsdk:"value"`
	Note           types.String `tfsdk:"note"`
	OrganizationId types.String `tfsdk:"organization_id"`
	ProjectId      types.String `tfsdk:"project_id"`
}

func (p secretDataSource) Metadata(ctx context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_secrets"
}

func (p secretDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		Description: "Fetches the list of projects.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "projects identities",
				Computed:    true,
			},
			"secrets": schema.ListNestedAttribute{
				Description: "List of projects.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"Key": schema.StringAttribute{
							Description: "Key/Name of the secret",
							Computed:    true,
						},
						"value": schema.StringAttribute{
							Description: "value of the secret",
							Computed:    true,
							Sensitive:   true,
						},
						"note": schema.StringAttribute{
							Description: "note for the secret",
							Computed:    true,
							Sensitive:   true,
						},
						"ID": schema.StringAttribute{
							Description: "Id of the secret",
							Computed:    true,
						},
						"ProjectId": schema.StringAttribute{
							Description: "Id of the project",
							Computed:    true,
						},
						"OrganizationID": schema.StringAttribute{
							Description: "organization ID associated with the project",
							Computed:    true,
						},
						"RevisionDate": schema.StringAttribute{
							Description: "Last date the project was updated/revised",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (p *secretDataSource) Configure(_ context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
	if request.ProviderData == nil {
		return
	}

	client, ok := request.ProviderData.(bitwarden.BitwardenClientInterface)
	if !ok {
		response.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *bitwarden.BitwardenClientInterface, got: %T. Please report this issue to the provider developers.", request.ProviderData),
		)

		return
	}

	p.client = client
}

func (p secretDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var info secretDataSourceModel

	request.Config.Get(ctx, &info)

	for _, secretInfo := range info.Secrets {
		secret, err := p.client.Secrets().Get(secretInfo.Id.ValueString())

		if err != nil {
			response.Diagnostics.AddError(
				"Unable to list secrets under organization id",
				"Validate that the organization id is not empty and is valid.",
			)

			return
		}
		secretModel := secretModel{
			Key:            types.StringValue(secret.Key),
			Value:          types.StringValue(secret.Value),
			Note:           types.StringValue(secret.Note),
			ProjectId:      types.StringValue(*secret.ProjectID),
			OrganizationId: types.StringValue(secret.OrganizationID),
			Id:             types.StringValue(secret.ID),
		}
		info.Secrets = append(info.Secrets, secretModel)
	}
	diags := response.State.Set(ctx, &info)

	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
}
