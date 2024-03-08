package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/samber/lo"

	"github.com/zeet-dev/cli/pkg/api"
	zeetv0 "github.com/zeet-dev/cli/pkg/sdk/v0"
	zeetv1 "github.com/zeet-dev/cli/pkg/sdk/v1"
	"github.com/zeet-dev/terraform-provider-zeet/internal/provider/customtypes"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ProjectResource{}
var _ resource.ResourceWithImportState = &ProjectResource{}

func NewProjectResource() resource.Resource {
	return &ProjectResource{}
}

// ProjectResource defines the resource implementation.
type ProjectResource struct {
	client *api.Client
}

// ProjectResourceModel describes the resource data model.
type ProjectResourceModel struct {
	TeamId     customtypes.UUIDValue `tfsdk:"team_id"`
	GroupId    customtypes.UUIDValue `tfsdk:"group_id"`
	SubGroupId customtypes.UUIDValue `tfsdk:"subgroup_id"`
	Id         customtypes.UUIDValue `tfsdk:"id"`

	Name        types.String          `tfsdk:"name"`
	Enabled     types.Bool            `tfsdk:"enabled"`
	BlueprintId customtypes.UUIDValue `tfsdk:"blueprint_id"`

	// for IAC based projects
	Deploys  []ProjectDeployModel  `tfsdk:"deploys"`
	Workflow *ProjectWorkflowModel `tfsdk:"workflow"`

	// for Container based projects
	Container *ProjectContainerModel `tfsdk:"container"`
}

type ProjectDeployModel struct {
	DefaultWorkflowSteps []types.String       `tfsdk:"default_workflow_steps"`
	RequirePlanApproval  types.Bool           `tfsdk:"require_plan_approval"`
	Variables            jsontypes.Normalized `tfsdk:"variables"`
	Kubernetes           jsontypes.Normalized `tfsdk:"kubernetes"`
	Helm                 jsontypes.Normalized `tfsdk:"helm"`
	Terraform            jsontypes.Normalized `tfsdk:"terraform"`
}

type ProjectWorkflowModel struct {
	Steps jsontypes.Normalized `tfsdk:"steps"`
}

type ProjectContainerModel struct {
	Source     ProjectContainerSourceModel    `tfsdk:"source"`
	Branch     *ProjectContainerBranchModel   `tfsdk:"branch"`
	Workflow   *ProjectContainerWorkflowModel `tfsdk:"workflow"`
	Build      jsontypes.Normalized           `tfsdk:"build"`
	Kubernetes jsontypes.Normalized           `tfsdk:"kubernetes"`
}

type ProjectContainerSourceModel struct {
	Git               jsontypes.Normalized `tfsdk:"git"`
	ContainerRegistry jsontypes.Normalized `tfsdk:"container_registry"`
}

type ProjectContainerBranchModel struct {
	ProductionBranch types.String `tfsdk:"production_branch"`
	AutoDeployBranch types.Bool   `tfsdk:"auto_deploy_branch"`
	AutoStopBranch   types.Bool   `tfsdk:"auto_stop_branch"`
	BranchIgnore     types.String `tfsdk:"branch_ignore"`
	BranchStopIgnore types.String `tfsdk:"branch_stop_ignore"`
}

type ProjectContainerWorkflowModel struct {
	AutoRetry            types.Bool            `tfsdk:"auto_retry"`
	AutoRollback         types.Bool            `tfsdk:"auto_rollback"`
	ManualDeploy         types.Bool            `tfsdk:"manual_deploy"`
	PipelineClusterId    customtypes.UUIDValue `tfsdk:"pipeline_cluster_id"`
	DeployTimeoutSeconds types.Int64           `tfsdk:"deploy_timeout_seconds"`
}

func (r *ProjectResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

func (r *ProjectResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Project resource",
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
			"subgroup_id": schema.StringAttribute{
				MarkdownDescription: "Subgroup identifier",
				Required:            true,
				CustomType:          customtypes.UUIDType{},
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Project identifier",
				CustomType:          customtypes.UUIDType{},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Project name",
				Required:            true,
			},
			"blueprint_id": schema.StringAttribute{
				MarkdownDescription: "Blueprint identifier",
				Required:            true,
				CustomType:          customtypes.UUIDType{},
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Indicates if the project is enabled or not (draft mode)",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"deploys": schema.ListNestedAttribute{
				MarkdownDescription: "Deployment configurations",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"default_workflow_steps": schema.ListAttribute{
							MarkdownDescription: "Default workflow steps for deployment",
							Required:            true,
							ElementType:         types.StringType,
						},
						"require_plan_approval": schema.BoolAttribute{
							MarkdownDescription: "Indicates if deployment plan approval is required",
							Optional:            true,
						},
						"variables": schema.StringAttribute{
							MarkdownDescription: "Blueprint variables, GraphQL type [`[BlueprintVariableInput]`](https://docs.zeet.co/graphql/inputs/blueprint-variable-input/)",
							Optional:            true,
							CustomType:          jsontypes.NormalizedType{},
						},
						"kubernetes": schema.StringAttribute{
							MarkdownDescription: "Kubernetes deployment configuration",
							Optional:            true,
							CustomType:          jsontypes.NormalizedType{},
						},
						"helm": schema.StringAttribute{
							MarkdownDescription: "Helm deployment configuration",
							Optional:            true,
							CustomType:          jsontypes.NormalizedType{},
						},
						"terraform": schema.StringAttribute{
							MarkdownDescription: "Terraform deployment configuration",
							Optional:            true,
							CustomType:          jsontypes.NormalizedType{},
						},
					}},
			},
			"workflow": schema.SingleNestedAttribute{
				MarkdownDescription: "Workflow configuration",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"steps": schema.StringAttribute{
						MarkdownDescription: "Workflow steps",
						Required:            true,
						CustomType:          jsontypes.NormalizedType{},
					},
				},
			},
			"container": schema.SingleNestedAttribute{
				MarkdownDescription: "Container configuration",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"source": schema.SingleNestedAttribute{
						MarkdownDescription: "Source configuration for the container",
						Required:            true,
						Attributes: map[string]schema.Attribute{
							"git": schema.StringAttribute{
								MarkdownDescription: "Git configuration for container deployment",
								Optional:            true,
								CustomType:          jsontypes.NormalizedType{},
							},
							"container_registry": schema.StringAttribute{
								MarkdownDescription: "Container registry configuration for container deployment",
								Optional:            true,
								CustomType:          jsontypes.NormalizedType{},
							},
						},
					},
					"branch": schema.StringAttribute{
						MarkdownDescription: "Branch configuration for container deployment",
						Optional:            true,
						CustomType:          jsontypes.NormalizedType{},
					},
					"workflow": schema.StringAttribute{
						MarkdownDescription: "Workflow configuration for container deployment",
						Optional:            true,
						CustomType:          jsontypes.NormalizedType{},
					},
					"build": schema.StringAttribute{
						MarkdownDescription: "Build configuration for the container",
						Optional:            true,
						CustomType:          jsontypes.NormalizedType{},
					},
					"kubernetes": schema.StringAttribute{
						MarkdownDescription: "Kubernetes configuration for the container",
						Required:            true,
						CustomType:          jsontypes.NormalizedType{},
					},
				},
			},
		},
	}
}

func (r *ProjectResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ProjectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ProjectResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if data.Container != nil {
		// Create container project
		result, err := zeetv0.CreateResourceAlphaMutation(ctx, r.client.Client(), zeetv0.CreateResourceAlphaInput{
			UserID:        data.TeamId.ValueUUID(),
			ProjectID:     lo.ToPtr(data.GroupId.ValueUUID()),
			EnvironmentID: lo.ToPtr(data.SubGroupId.ValueUUID()),

			Name:        data.Name.ValueString(),
			BlueprintID: lo.ToPtr(data.BlueprintId.ValueUUID()),
		})
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create project, got error: %s", err))
			return
		}

		team, err := zeetv0.UserQuery(ctx, r.client.Client(), data.TeamId.ValueUUID())
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read team, got error: %s", err))
			return
		}

		getResult, err := zeetv0.RepoForProjectEnvironmentQuery(ctx, r.client.Client(),
			fmt.Sprintf("%s/%s", team.User.Name, result.CreateResourceAlpha.Project.Name),
			result.CreateResourceAlpha.ProjectEnvironment.Name,
			result.CreateResourceAlpha.Name)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read project, got error: %s", err))
			return
		}

		data.Id = customtypes.NewUUIDValue(getResult.Project.Environment.Repo.Id)
	} else if data.Workflow != nil && len(data.Deploys) > 0 {
		// Create workflow project
		result, err := zeetv1.CreateProjectMutation(ctx, r.client.ClientV1(), zeetv1.CreateProjectInput{
			TeamId:     data.TeamId.ValueUUID(),
			GroupId:    lo.ToPtr(data.GroupId.ValueUUID()),
			SubGroupId: lo.ToPtr(data.SubGroupId.ValueUUID()),

			Name:        data.Name.ValueString(),
			BlueprintId: lo.ToPtr(data.BlueprintId.ValueUUID()),
		})
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create project, got error: %s", err))
			return
		}

		data.Id = customtypes.NewUUIDValue(result.CreateProject.Id)
	} else {
		// Not valid
		resp.Diagnostics.AddError("Invalid Configuration", "Project must have either a container or workflow configuration")
		return
	}

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "created a resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ProjectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ProjectResourceModel
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	result, err := zeetv1.ProjectDetailQuery(ctx, r.client.ClientV1(), data.TeamId.ValueUUID(), data.Id.ValueUUID())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read project, got error: %s", err))
		return
	}

	data.Name = types.StringValue(result.Team.Project.Name)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ProjectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ProjectResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	result, err := zeetv1.UpdateProjectMutation(ctx, r.client.ClientV1(), data.Id.ValueUUID(), zeetv1.UpdateProjectInput{
		Name: lo.ToPtr(data.Name.ValueString()),
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update project, got error: %s", err))
		return
	}

	data.Name = types.StringValue(result.UpdateProject.Name)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ProjectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ProjectResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	_, err := zeetv1.DeleteProjectMutation(ctx, r.client.ClientV1(), data.Id.ValueUUID(), lo.ToPtr(false))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete project, got error: %s", err))
		return
	}
}

func (r *ProjectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
