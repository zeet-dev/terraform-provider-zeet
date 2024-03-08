package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/samber/lo"

	"github.com/zeet-dev/cli/pkg/api"
	zeetv1 "github.com/zeet-dev/cli/pkg/sdk/v1"
	"github.com/zeet-dev/terraform-provider-zeet/internal/provider/customtypes"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &GroupSubgroupResource{}
var _ resource.ResourceWithImportState = &GroupSubgroupResource{}

func NewGroupSubgroupResource() resource.Resource {
	return &GroupSubgroupResource{}
}

// GroupSubgroupResource defines the resource implementation.
type GroupSubgroupResource struct {
	client *api.Client
}

// GroupSubgroupResourceModel describes the resource data model.
type GroupSubgroupResourceModel struct {
	Id      customtypes.UUIDValue `tfsdk:"id"`
	GroupId customtypes.UUIDValue `tfsdk:"group_id"`
	TeamId  customtypes.UUIDValue `tfsdk:"team_id"`
	Name    types.String          `tfsdk:"name"`
}

func (r *GroupSubgroupResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_subgroup"
}

func (r *GroupSubgroupResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Group Subgroup resource",

		Attributes: map[string]schema.Attribute{
			"team_id": schema.StringAttribute{
				MarkdownDescription: "Team identifier",
				Required:            true,
				CustomType:          customtypes.UUIDType{},
			},
			"group_id": schema.StringAttribute{
				MarkdownDescription: "Group identifier",
				Required:            true,
				CustomType:          customtypes.UUIDType{},
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Subgroup identifier",
				CustomType:          customtypes.UUIDType{},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Subgroup name",
				Required:            true,
			},
		},
	}
}

func (r *GroupSubgroupResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*api.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *api.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *GroupSubgroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data GroupSubgroupResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	result, err := zeetv1.CreateSubGroupMutation(ctx, r.client.ClientV1(), zeetv1.CreateSubGroupInput{
		GroupId: data.GroupId.ValueUUID(),
		Name:    data.Name.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create subgroup, got error: %s", err))
		return
	}

	data.Id = customtypes.NewUUIDValue(result.CreateSubGroup.Id)

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "created a resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *GroupSubgroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data GroupSubgroupResourceModel
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	result, err := zeetv1.SubGroupQuery(ctx, r.client.ClientV1(), data.TeamId.ValueUUID(), data.GroupId.ValueUUID(), data.Id.ValueUUID())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read subgroup, got error: %s", err))
		return
	}

	data.Name = types.StringValue(result.Team.Groups.Nodes[0].SubGroup.Name)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *GroupSubgroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data GroupSubgroupResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	result, err := zeetv1.UpdateSubGroupMutation(ctx, r.client.ClientV1(), data.Id.ValueUUID(), zeetv1.UpdateSubGroupInput{
		Name: lo.ToPtr(data.Name.ValueString()),
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update subgroup, got error: %s", err))
		return
	}

	data.Name = types.StringValue(result.UpdateSubGroup.Name)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *GroupSubgroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data GroupSubgroupResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	_, err := zeetv1.DeleteSubGroupMutation(ctx, r.client.ClientV1(), data.Id.ValueUUID())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete subgroup, got error: %s", err))
		return
	}
}

func (r *GroupSubgroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
