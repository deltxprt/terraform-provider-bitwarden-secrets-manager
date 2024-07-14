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
var _ resource.Resource = &SecretResource{}
var _ resource.ResourceWithImportState = &SecretResource{}

func NewSecretResource() resource.Resource {
	return &SecretResource{}
}

// SecretResource defines the resource implementation.
type SecretResource struct {
	client bitwarden.BitwardenClientInterface
}

// SecretResourceModel describes the resource data model.
type SecretResourceModel struct {
	Secrets []secretItemModel `tfsdk:"secrets"`
	Id      types.String      `tfsdk:"id"`
}

type secretItemModel struct {
	Key            types.String `tfsdk:"key"`
	Value          types.String `tfsdk:"value"`
	Note           types.String `tfsdk:"note"`
	SecretId       types.String `tfsdk:"secret_id"`
	ProjectId      types.String `tfsdk:"project_id"`
	OrganizationId types.String `tfsdk:"organization_id"`
}

func (r *SecretResource) Metadata(ctx context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_secret"
}

func (r *SecretResource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Secret Resource",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "name of the project",
				Optional:            true,
			},
			"secret_id": schema.StringAttribute{
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

func (r *SecretResource) Configure(ctx context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
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

func (r *SecretResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data *SecretResourceModel

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

	var secretsCreation []*bitwarden.SecretResponse
	for _, secret := range data.Secrets {
		SecretCreation, err := r.client.Secrets().Create(
			secret.Key.ValueString(),
			secret.Value.ValueString(),
			secret.Note.ValueString(),
			secret.OrganizationId.ValueString(),
			[]string{secret.ProjectId.ValueString()},
		)
		if err != nil {
			response.Diagnostics.AddError(
				"Error creating secret",
				"Could not create secret, unexpected error: "+err.Error(),
			)
			return
		}
		secretsCreation = append(secretsCreation, SecretCreation)
	}

	for projectIndex, projectItem := range secretsCreation {
		data.Secrets[projectIndex] = secretItemModel{
			Key:            types.StringValue(projectItem.Key),
			Value:          types.StringValue(projectItem.Value),
			Note:           types.StringValue(projectItem.Note),
			OrganizationId: types.StringValue(projectItem.OrganizationID),
			SecretId:       types.StringValue(projectItem.ID),
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

func (r *SecretResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var data *SecretResourceModel

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

	var secrets []*bitwarden.SecretResponse
	for _, secret := range data.Secrets {
		secret, err := r.client.Secrets().Get(secret.SecretId.ValueString())
		if err != nil {
			response.Diagnostics.AddError(
				"Error creating secret",
				"Could not find secret, unexpected error: "+err.Error(),
			)
			return
		}
		secrets = append(secrets, secret)
	}

	for projectIndex, projectItem := range secrets {
		data.Secrets[projectIndex] = secretItemModel{
			Key:            types.StringValue(projectItem.Key),
			Value:          types.StringValue(projectItem.Value),
			Note:           types.StringValue(projectItem.Note),
			OrganizationId: types.StringValue(projectItem.OrganizationID),
			SecretId:       types.StringValue(projectItem.ID),
		}
	}

	// Save updated data into Terraform state
	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}

func (r *SecretResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var data *SecretResourceModel

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

	var secrets []*bitwarden.SecretResponse
	for _, secret := range data.Secrets {
		secret, err := r.client.Secrets().Update(
			secret.SecretId.ValueString(),
			secret.Key.ValueString(),
			secret.Value.ValueString(),
			secret.Note.ValueString(),
			secret.OrganizationId.ValueString(),
			[]string{secret.ProjectId.ValueString()},
		)
		if err != nil {
			response.Diagnostics.AddError(
				"Error creating secret",
				"Could not update secret, unexpected error: "+err.Error(),
			)
			return
		}
		secrets = append(secrets, secret)
	}

	for projectIndex, projectItem := range secrets {
		data.Secrets[projectIndex] = secretItemModel{
			SecretId:       types.StringValue(projectItem.ID),
			Key:            types.StringValue(projectItem.Key),
			Value:          types.StringValue(projectItem.Value),
			Note:           types.StringValue(projectItem.Note),
			OrganizationId: types.StringValue(projectItem.OrganizationID),
			ProjectId:      types.StringValue(*projectItem.ProjectID),
		}
	}

	// Save updated data into Terraform state
	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}
}

func (r *SecretResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var data *SecretResourceModel

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

	var secretsToDelete []string

	for _, secret := range data.Secrets {
		secretsToDelete = append(secretsToDelete, secret.SecretId.ValueString())
	}

	_, err := r.client.Secrets().Delete(secretsToDelete)
	if err != nil {
		response.Diagnostics.AddError(
			"Error creating secret",
			"Could not delete secret, unexpected error: "+err.Error(),
		)
		return
	}

}

func (r *SecretResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), request, response)
}
