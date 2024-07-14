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
	_ datasource.DataSource              = &projectDataSource{}
	_ datasource.DataSourceWithConfigure = &projectDataSource{}
)

// NewProjectDataSource is a helper function to simplify the provider implementation.
func NewProjectDataSource() datasource.DataSource {
	return &projectDataSource{}
}

// projectDataSource is the data source implementation.
type projectDataSource struct {
	client bitwarden.BitwardenClientInterface
}

// projectDataSourceModel maps the data source schema data.
type projectDataSourceModel struct {
	Projects []projectModel `tfsdk:"projects"`
	ID       types.String   `tfsdk:"id"`
}

type projectModel struct {
	CreationDate   types.String `tfsdk:"creation_date"`
	Id             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	OrganizationId types.String `tfsdk:"organization_id"`
	RevisionDate   types.String `tfsdk:"revision_date"`
}

func (p projectDataSource) Metadata(ctx context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_projects"
}

func (p projectDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		Description: "Fetches the list of projects.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "projects identities",
				Computed:    true,
			},
			"projects": schema.ListNestedAttribute{
				Description: "List of projects.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"CreationDate": schema.StringAttribute{
							Description: "Creation date of the project",
							Computed:    true,
							Optional:    true,
						},
						"Name": schema.StringAttribute{
							Description: "Name of the project",
							Computed:    true,
						},
						"ID": schema.StringAttribute{
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
							Optional:    true,
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (p *projectDataSource) Configure(_ context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
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

func (p projectDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var info projectDataSourceModel

	request.Config.Get(ctx, &info)

	for _, projectInfo := range info.Projects {
		project, err := p.client.Projects().Get(projectInfo.Id.ValueString())

		if err != nil {
			response.Diagnostics.AddError(
				"Unable to list projects under organization id",
				"Validate that the organization id is not empty and is valid.",
			)

			return
		}
		secretModel := projectModel{
			CreationDate:   types.StringValue(project.CreationDate),
			Name:           types.StringValue(project.Name),
			Id:             types.StringValue(project.ID),
			OrganizationId: types.StringValue(project.OrganizationID),
			RevisionDate:   types.StringValue(project.RevisionDate),
		}
		info.Projects = append(info.Projects, secretModel)
	}

	diags := response.State.Set(ctx, &info)

	response.Diagnostics.Append(diags...)
}
