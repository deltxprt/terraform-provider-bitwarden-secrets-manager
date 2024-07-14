// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	bitwarden "github.com/bitwarden/sdk-go"
	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ProjectResource{}
var _ resource.ResourceWithImportState = &ProjectResource{}

func NewProjectResource() resource.Resource {
	return &ProjectResource{}
}

// ProjectResource defines the resource implementation.
type ProjectResource struct {
	client bitwarden.BitwardenClientInterface
}

// ProjectResourceModel describes the resource data model.
type ProjectResourceModel struct {
	Projects []projectItemModel `tfsdk:"projects"`
	Id       types.String       `tfsdk:"id"`
}

type projectItemModel struct {
	Name           types.String `tfsdk:"name"`
	ProjectId      types.String `tfsdk:"project_id"`
	OrganizationId types.String `tfsdk:"organization_id"`
}

func (r *ProjectResource) Metadata(ctx context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_project"
}

func (r *ProjectResource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Project Resource",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "name of the project",
				Optional:            true,
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "id of the project in bitwarden secrets manager",
				Computed:            true,
			},
			"organization_id": schema.StringAttribute{
				MarkdownDescription: "id of the organization associated with the project",
				Computed:            true,
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Example identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *ProjectResource) Configure(ctx context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if request.ProviderData == nil {
		return
	}

	client, ok := request.ProviderData.(bitwarden.BitwardenClientInterface)

	if !ok {
		response.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected bitwarden.BitwardenClientInterface, got: %T. Please report this issue to the provider developers.", request.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *ProjectResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data *ProjectResourceModel

	// Read Terraform plan data into the model
	response.Diagnostics.Append(request.Plan.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	// httpResp, err := r.client.Do(httpReq)
	// if err != nil {
	//     response.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create example, got error: %s", err))
	//     return
	// }

	resourceId, err := uuid.GenerateUUID()

	if err != nil {
		response.Diagnostics.AddAttributeError(
			path.Root("resource_ID"),
			"Unable to generate resource id",
			"The projects couldn't be created, due to and id generation issue",
		)
	}
	data.Id = types.StringValue(resourceId)

	var projectsCreation []*bitwarden.ProjectResponse
	for _, project := range data.Projects {
		projectCreation, err := r.client.Projects().Create(project.Name.ValueString(), project.OrganizationId.ValueString())
		if err != nil {
			response.Diagnostics.AddError(
				"Error creating project",
				"Could not create project, unexpected error: "+err.Error(),
			)
			return
		}
		projectsCreation = append(projectsCreation, projectCreation)
	}

	for projectIndex, projectItem := range projectsCreation {
		data.Projects[projectIndex] = projectItemModel{
			Name:           types.StringValue(projectItem.Name),
			OrganizationId: types.StringValue(projectItem.OrganizationID),
			ProjectId:      types.StringValue(projectItem.ID),
		}
	}

	// For the purposes of this example code, hardcoding a response value to
	// save into the Terraform state.

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "created a resource")

	// Save data into Terraform state
	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}

func (r *ProjectResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var data *ProjectResourceModel

	// Read Terraform prior state data into the model
	response.Diagnostics.Append(request.State.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	// httpResp, err := r.client.Do(httpReq)
	// if err != nil {
	//     response.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read example, got error: %s", err))
	//     return
	// }

	var projects []*bitwarden.ProjectResponse
	for _, project := range data.Projects {
		project, err := r.client.Projects().Get(project.ProjectId.ValueString())
		if err != nil {
			response.Diagnostics.AddError(
				"Error creating project",
				"Could not find project, unexpected error: "+err.Error(),
			)
			return
		}
		projects = append(projects, project)
	}

	for projectIndex, projectItem := range projects {
		data.Projects[projectIndex] = projectItemModel{
			Name:           types.StringValue(projectItem.Name),
			OrganizationId: types.StringValue(projectItem.OrganizationID),
			ProjectId:      types.StringValue(projectItem.ID),
		}
	}

	// Save updated data into Terraform state
	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}

func (r *ProjectResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var data *ProjectResourceModel

	// Read Terraform plan data into the model
	response.Diagnostics.Append(request.Plan.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	// httpResp, err := r.client.Do(httpReq)
	// if err != nil {
	//     response.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update example, got error: %s", err))
	//     return
	// }

	var projects []*bitwarden.ProjectResponse
	for _, project := range data.Projects {
		project, err := r.client.Projects().Update(
			project.ProjectId.ValueString(),
			project.OrganizationId.ValueString(),
			project.Name.ValueString(),
		)
		if err != nil {
			response.Diagnostics.AddError(
				"Error creating project",
				"Could not update project, unexpected error: "+err.Error(),
			)
			return
		}
		projects = append(projects, project)
	}

	for projectIndex, projectItem := range projects {
		data.Projects[projectIndex] = projectItemModel{
			Name:           types.StringValue(projectItem.Name),
			OrganizationId: types.StringValue(projectItem.OrganizationID),
			ProjectId:      types.StringValue(projectItem.ID),
		}
	}

	// Save updated data into Terraform state
	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}
}

func (r *ProjectResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var data *ProjectResourceModel

	// Read Terraform prior state data into the model
	response.Diagnostics.Append(request.State.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	// httpResp, err := r.client.Do(httpReq)
	// if err != nil {
	//     response.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete example, got error: %s", err))
	//     return
	// }

	var projectsToDelete []string

	for _, project := range data.Projects {
		projectsToDelete = append(projectsToDelete, project.ProjectId.ValueString())
	}

	_, err := r.client.Projects().Delete(projectsToDelete)
	if err != nil {
		response.Diagnostics.AddError(
			"Error creating project",
			"Could not delete project, unexpected error: "+err.Error(),
		)
		return
	}

}

func (r *ProjectResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), request, response)
}
