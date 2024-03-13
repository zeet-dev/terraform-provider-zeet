package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
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
	Id                   customtypes.UUIDValue `tfsdk:"id"`
	DefaultWorkflowSteps []types.String        `tfsdk:"default_workflow_steps"`
	RequirePlanApproval  types.Bool            `tfsdk:"require_plan_approval"`
	Variables            jsontypes.Normalized  `tfsdk:"variables"`
	Kubernetes           jsontypes.Normalized  `tfsdk:"kubernetes"`
	Helm                 jsontypes.Normalized  `tfsdk:"helm"`
	Terraform            jsontypes.Normalized  `tfsdk:"terraform"`
}

type ProjectWorkflowModel struct {
	Id    customtypes.UUIDValue `tfsdk:"id"`
	Steps jsontypes.Normalized  `tfsdk:"steps"`
}

type ProjectContainerModel struct {
	RepoId     customtypes.UUIDValue          `tfsdk:"repo_id"`
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

func (p *ProjectResourceModel) IsContainer() bool {
	return p.Container != nil
}

func (p *ProjectResourceModel) IsWorkflow() bool {
	return p.Workflow != nil && len(p.Deploys) > 0

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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"group_id": schema.StringAttribute{
				MarkdownDescription: "Group identifier",
				Required:            true,
				CustomType:          customtypes.UUIDType{},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(), // TODO: allow moving projects between groups
				},
			},
			"subgroup_id": schema.StringAttribute{
				MarkdownDescription: "Subgroup identifier",
				Required:            true,
				CustomType:          customtypes.UUIDType{},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(), // TODO: allow moving projects between groups
				},
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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Indicates if the project is enabled or not (paused or draft state)",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"deploys": schema.ListNestedAttribute{
				MarkdownDescription: "Deployment configurations",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "Deployment identifier",
							Computed:            true,
							CustomType:          customtypes.UUIDType{},
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"default_workflow_steps": schema.ListAttribute{
							MarkdownDescription: "Default workflow steps for deployment list of [step types](https://docs.zeet.co/graphql/enums/blueprint-driver-workflow-step-action/)",
							Required:            true,
							ElementType:         types.StringType,
						},
						"require_plan_approval": schema.BoolAttribute{
							MarkdownDescription: "Indicates if the approval step is required in all workflow runs",
							Optional:            true,
							Computed:            true,
							Default:             booldefault.StaticBool(false),
						},
						"variables": schema.StringAttribute{
							MarkdownDescription: "Blueprint variables, GraphQL type [`[BlueprintVariableInput]`](https://docs.zeet.co/graphql/inputs/blueprint-variable-input/)",
							Optional:            true,
							CustomType:          jsontypes.NormalizedType{},
						},
						"kubernetes": schema.StringAttribute{
							MarkdownDescription: "Kubernetes deployment configuration in [JSON format](https://docs.zeet.co/graphql/inputs/deployment-configuration-kubernetes-input/)",
							Optional:            true,
							CustomType:          jsontypes.NormalizedType{},
						},
						"helm": schema.StringAttribute{
							MarkdownDescription: "Helm deployment configuration in [JSON format](https://docs.zeet.co/graphql/inputs/deployment-configuration-kubernetes-helm-input/)",
							Optional:            true,
							CustomType:          jsontypes.NormalizedType{},
						},
						"terraform": schema.StringAttribute{
							MarkdownDescription: "Terraform deployment configuration in [JSON format](https://docs.zeet.co/graphql/inputs/deployment-configuration-terraform-input/)",
							Optional:            true,
							CustomType:          jsontypes.NormalizedType{},
						},
					}},
			},
			"workflow": schema.SingleNestedAttribute{
				MarkdownDescription: "Workflow configuration",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						MarkdownDescription: "Workflow identifier",
						Computed:            true,
						CustomType:          customtypes.UUIDType{},
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"steps": schema.StringAttribute{
						MarkdownDescription: "Workflow steps in [JSON format](https://docs.zeet.co/graphql/objects/workflow-step-definition/)",
						Required:            true,
						CustomType:          jsontypes.NormalizedType{},
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
				},
			},
			"container": schema.SingleNestedAttribute{
				MarkdownDescription: "Container configuration",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"repo_id": schema.StringAttribute{
						MarkdownDescription: "Repo identifier",
						Computed:            true,
						CustomType:          customtypes.UUIDType{},
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"source": schema.SingleNestedAttribute{
						MarkdownDescription: "Source configuration for the container",
						Required:            true,
						Attributes: map[string]schema.Attribute{
							"git": schema.StringAttribute{
								MarkdownDescription: "Git configuration for container deployment in [JSON format](https://docs.zeet.co/0.1.0/graphql/inputs/git-source-input/)",
								Optional:            true,
								CustomType:          jsontypes.NormalizedType{},
							},
							"container_registry": schema.StringAttribute{
								MarkdownDescription: "Container registry configuration for container deployment in [JSON format](https://docs.zeet.co/0.1.0/graphql/inputs/container-registry-source-input/)",
								Optional:            true,
								CustomType:          jsontypes.NormalizedType{},
							},
						},
					},
					"branch": schema.SingleNestedAttribute{
						MarkdownDescription: "Branch configuration for container deployment",
						Optional:            true,
						Computed:            true,
						Attributes: map[string]schema.Attribute{
							"production_branch": schema.StringAttribute{
								MarkdownDescription: "Production branch for container deployment",
								Optional:            true,
								Computed:            true,
								Default:             stringdefault.StaticString("production"),
							},
							"auto_deploy_branch": schema.BoolAttribute{
								MarkdownDescription: "Indicates if auto deploy branch is enabled\nUpdate is not supported at the moment",
								Optional:            true,
							},
							"auto_stop_branch": schema.BoolAttribute{
								MarkdownDescription: "Indicates if auto stop branch is enabled\nUpdate is not supported at the moment",
								Optional:            true,
							},
							"branch_ignore": schema.StringAttribute{
								MarkdownDescription: "Branch ignore for container deployment\nUpdate is not supported at the moment",
								Optional:            true,
							},
							"branch_stop_ignore": schema.StringAttribute{
								MarkdownDescription: "Branch stop ignore for container deployment\nUpdate is not supported at the moment",
								Optional:            true,
							},
						},
						Default: objectdefault.StaticValue(
							types.ObjectValueMust(
								map[string]attr.Type{
									"production_branch": types.StringType,
								},
								map[string]attr.Value{
									"production_branch": types.StringValue("production"),
								},
							),
						),
					},
					"workflow": schema.SingleNestedAttribute{
						MarkdownDescription: "Workflow configuration for container deployment",
						Optional:            true,
						Attributes: map[string]schema.Attribute{
							"auto_retry": schema.BoolAttribute{
								MarkdownDescription: "Indicates if auto retry is enabled",
								Optional:            true,
								Computed:            true,
								Default:             booldefault.StaticBool(false),
							},
							"auto_rollback": schema.BoolAttribute{
								MarkdownDescription: "Indicates if auto rollback is enabled",
								Optional:            true,
								Computed:            true,
								Default:             booldefault.StaticBool(false),
							},
							"manual_deploy": schema.BoolAttribute{
								MarkdownDescription: "Indicates if manual deploy is enabled",
								Optional:            true,
								Computed:            true,
								Default:             booldefault.StaticBool(false),
							},
							"pipeline_cluster_id": schema.StringAttribute{
								MarkdownDescription: "Pipeline cluster identifier",
								Optional:            true,
								CustomType:          customtypes.UUIDType{},
							},
							"deploy_timeout_seconds": schema.Int64Attribute{
								MarkdownDescription: "Deploy timeout in seconds",
								Optional:            true,
							},
						},
					},
					"build": schema.StringAttribute{
						MarkdownDescription: "Build configuration for the container deployment in [JSON format](https://docs.zeet.co/0.1.0/graphql/inputs/resource-build-input/)\nUpdate is not supported at the moment",
						Optional:            true,
						CustomType:          jsontypes.NormalizedType{},
					},
					"kubernetes": schema.StringAttribute{
						MarkdownDescription: "Kubernetes configuration for the container deployment in [JSON format](https://docs.zeet.co/0.1.0/graphql/inputs/resource-kubernetes-input/)\nUpdate is not supported at the moment",
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

	if data.IsContainer() {
		createInput := zeetv0.CreateResourceAlphaInput{
			UserID:        data.TeamId.ValueUUID(),
			ProjectID:     lo.ToPtr(data.GroupId.ValueUUID()),
			EnvironmentID: lo.ToPtr(data.SubGroupId.ValueUUID()),

			Name:        data.Name.ValueString(),
			BlueprintID: lo.ToPtr(data.BlueprintId.ValueUUID()),

			Enabled: lo.ToPtr(data.Enabled.ValueBool()),
		}
		if !data.Container.Source.Git.IsNull() {
			createInput.Source = &zeetv0.SourceInput{
				Git: &zeetv0.GitSourceInput{},
			}
			if err := json.Unmarshal([]byte(data.Container.Source.Git.ValueString()), &createInput.Source.Git); err != nil {
				resp.Diagnostics.AddError("Invalid Configuration", fmt.Sprintf("Unable to unmarshal git source, got error: %s", err))
				return
			}
		} else if !data.Container.Source.ContainerRegistry.IsNull() {
			createInput.Source = &zeetv0.SourceInput{
				ContainerRegistry: &zeetv0.ContainerRegistrySourceInput{},
			}
			if err := json.Unmarshal([]byte(data.Container.Source.ContainerRegistry.ValueString()), &createInput.Source.ContainerRegistry); err != nil {
				resp.Diagnostics.AddError("Invalid Configuration", fmt.Sprintf("Unable to unmarshal container registry source, got error: %s", err))
				return
			}
		} else {
			resp.Diagnostics.AddError("Invalid Configuration", "Container source must be either git or container registry")
			return
		}

		if data.Container.Branch != nil {
			createInput.Branch = &zeetv0.ResourceBranchInput{}

			if !data.Container.Branch.ProductionBranch.IsNull() {
				createInput.Branch.ProductionBranch = lo.ToPtr(data.Container.Branch.ProductionBranch.ValueString())
			}
			if !data.Container.Branch.AutoDeployBranch.IsNull() {
				createInput.Branch.AutoDeployBranch = lo.ToPtr(data.Container.Branch.AutoDeployBranch.ValueBool())
			}
			if !data.Container.Branch.AutoStopBranch.IsNull() {
				createInput.Branch.AutoStopBranch = lo.ToPtr(data.Container.Branch.AutoStopBranch.ValueBool())
			}
			if !data.Container.Branch.BranchIgnore.IsNull() {
				createInput.Branch.BranchIgnore = lo.ToPtr(data.Container.Branch.BranchIgnore.ValueString())
			}
			if !data.Container.Branch.BranchStopIgnore.IsNull() {
				createInput.Branch.BranchStopIgnore = lo.ToPtr(data.Container.Branch.BranchStopIgnore.ValueString())
			}
		}

		if data.Container.Workflow != nil {
			createInput.Workflow = &zeetv0.ResourceWorkflowInput{}
			if !data.Container.Workflow.AutoRetry.IsNull() {
				createInput.Workflow.AutoRetry = lo.ToPtr(data.Container.Workflow.AutoRetry.ValueBool())
			}
			if !data.Container.Workflow.AutoRollback.IsNull() {
				createInput.Workflow.AutoRollback = lo.ToPtr(data.Container.Workflow.AutoRollback.ValueBool())
			}
			if !data.Container.Workflow.ManualDeploy.IsNull() {
				createInput.Workflow.ManualDeploy = lo.ToPtr(data.Container.Workflow.ManualDeploy.ValueBool())
			}
			if !data.Container.Workflow.PipelineClusterId.IsNull() {
				createInput.Workflow.PipelineClusterID = lo.ToPtr(data.Container.Workflow.PipelineClusterId.ValueUUID())
			}
			if !data.Container.Workflow.DeployTimeoutSeconds.IsNull() {
				createInput.Workflow.DeployTimeoutSeconds = lo.ToPtr(int(data.Container.Workflow.DeployTimeoutSeconds.ValueInt64()))
			}
		}

		if !data.Container.Build.IsNull() {
			createInput.Build = &zeetv0.ResourceBuildInput{}
			if err := json.Unmarshal([]byte(data.Container.Build.ValueString()), &createInput.Build); err != nil {
				resp.Diagnostics.AddError("Invalid Configuration", fmt.Sprintf("Unable to unmarshal build, got error: %s", err))
				return
			}
		}

		if !data.Container.Kubernetes.IsNull() {
			createInput.Kubernetes = &zeetv0.ResourceKubernetesInput{}
			if err := json.Unmarshal([]byte(data.Container.Kubernetes.ValueString()), &createInput.Kubernetes); err != nil {
				resp.Diagnostics.AddError("Invalid Configuration", fmt.Sprintf("Unable to unmarshal kubernetes, got error: %s", err))
				return
			}
		}

		// Create container project
		result, err := zeetv0.CreateResourceAlphaMutation(ctx, r.client.Client(), createInput)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create project, got error: %s", err))
			return
		}

		// pull the project id again because it's only available in v1.Project
		pv3Result, err := zeetv0.ProjectV3Query(ctx, r.client.Client(), data.TeamId.ValueUUID().String(), result.CreateResourceAlpha.Project.Name, result.CreateResourceAlpha.ProjectEnvironment.Name,
			result.CreateResourceAlpha.Name)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read project, got error: %s", err))
			return
		}

		if len(pv3Result.User.ProjectV3Adapters.Nodes) != 1 {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read project, got error: %s", "project not found"))
			return
		}

		data.Id = customtypes.NewUUIDValue(pv3Result.User.ProjectV3Adapters.Nodes[0].Id)
		data.Container.RepoId = customtypes.NewUUIDValue(uuid.MustParse(pv3Result.User.ProjectV3Adapters.Nodes[0].Repo.Id))
	} else if data.IsWorkflow() {
		// Create workflow project
		createInput := zeetv1.CreateProjectInput{
			TeamId:     data.TeamId.ValueUUID(),
			GroupId:    lo.ToPtr(data.GroupId.ValueUUID()),
			SubGroupId: lo.ToPtr(data.SubGroupId.ValueUUID()),

			Name:        data.Name.ValueString(),
			BlueprintId: lo.ToPtr(data.BlueprintId.ValueUUID()),

			Enabled: lo.ToPtr(data.Enabled.ValueBool()),
		}

		if !data.Workflow.Steps.IsNull() {
			createInput.Workflow = &zeetv1.WorkflowDefinitionInput{
				Steps: []zeetv1.WorkflowStepDefinitionInput{},
			}
			if err := json.Unmarshal([]byte(data.Workflow.Steps.ValueString()), &createInput.Workflow.Steps); err != nil {
				resp.Diagnostics.AddError("Invalid Configuration", fmt.Sprintf("Unable to unmarshal workflow steps, got error: %s", err))
				return
			}
		} else {
			resp.Diagnostics.AddError("Invalid Configuration", "Workflow steps must be defined")
			return
		}

		for _, deploy := range data.Deploys {
			input := zeetv1.DeploymentConfigurationInput{
				DefaultWorkflowSteps: lo.Map(deploy.DefaultWorkflowSteps, func(s types.String, _ int) zeetv1.BlueprintDriverWorkflowStepAction {
					return zeetv1.BlueprintDriverWorkflowStepAction(s.ValueString())
				}),
			}

			if !deploy.RequirePlanApproval.IsNull() {
				input.RequirePlanApproval = lo.ToPtr(deploy.RequirePlanApproval.ValueBool())
			}

			if !deploy.Helm.IsNull() {
				input.Helm = &zeetv1.DeploymentConfigurationKubernetesHelmInput{}
				if err := json.Unmarshal([]byte(deploy.Helm.ValueString()), &input.Helm); err != nil {
					resp.Diagnostics.AddError("Invalid Configuration", fmt.Sprintf("Unable to unmarshal helm, got error: %s", err))
					return
				}
			}

			if !deploy.Kubernetes.IsNull() {
				input.Kubernetes = &zeetv1.DeploymentConfigurationKubernetesInput{}
				if err := json.Unmarshal([]byte(deploy.Kubernetes.ValueString()), &input.Kubernetes); err != nil {
					resp.Diagnostics.AddError("Invalid Configuration", fmt.Sprintf("Unable to unmarshal kubernetes, got error: %s", err))
					return
				}
			}

			if !deploy.Terraform.IsNull() {
				input.Terraform = &zeetv1.DeploymentConfigurationTerraformInput{}
				if err := json.Unmarshal([]byte(deploy.Terraform.ValueString()), &input.Terraform); err != nil {
					resp.Diagnostics.AddError("Invalid Configuration", fmt.Sprintf("Unable to unmarshal terraform, got error: %s", err))
					return
				}
			}

			if !deploy.Variables.IsNull() {
				input.Variables = []zeetv1.BlueprintVariableInput{}
				if err := json.Unmarshal([]byte(deploy.Variables.ValueString()), &input.Variables); err != nil {
					resp.Diagnostics.AddError("Invalid Configuration", fmt.Sprintf("Unable to unmarshal variables, got error: %s", err))
					return
				}
			}

			createInput.Deploys = append(createInput.Deploys, input)
		}

		createResult, err := zeetv1.CreateProjectMutation(ctx, r.client.ClientV1(), createInput)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create project, got error: %s", err))
			return
		}

		data.Id = customtypes.NewUUIDValue(createResult.CreateProject.Id)

		readResult, err := zeetv1.ProjectDetailQuery(ctx, r.client.ClientV1(), data.TeamId.ValueUUID(), data.Id.ValueUUID())
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read project, got error: %s", err))
			return
		}

		data.Name = types.StringValue(readResult.Team.Project.Name)
		data.Workflow.Id = customtypes.NewUUIDValue(readResult.Team.Project.Workflow.Id)
		for i, deploy := range readResult.Team.Project.Deploys.Nodes {
			data.Deploys[i].Id = customtypes.NewUUIDValue(deploy.Id)
		}
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

	// read logic
	if data.IsContainer() {
		getResult, err := zeetv0.UserRepoQuery(ctx, r.client.Client(), data.Container.RepoId.ValueUUID().String())
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read project, got error: %s", err))
			return
		}

		// basic data
		data.GroupId = customtypes.NewUUIDValue(getResult.CurrentUser.Repo.Project.Id)
		data.SubGroupId = customtypes.NewUUIDValue(getResult.CurrentUser.Repo.ProjectEnvironment.Id)
		data.Name = types.StringValue(getResult.CurrentUser.Repo.Name)
		prevContainer := data.Container
		data.Container = &ProjectContainerModel{
			RepoId: customtypes.NewUUIDValue(uuid.MustParse(getResult.CurrentUser.Repo.Id)),
		}

		// branch
		data.Container.Branch = &ProjectContainerBranchModel{}
		if getResult.CurrentUser.Repo.ProductionBranch != nil {
			data.Container.Branch.ProductionBranch = types.StringValue(*getResult.CurrentUser.Repo.ProductionBranch)
		}
		if getResult.CurrentUser.Repo.GithubIntegration != nil {
			data.Container.Branch.AutoDeployBranch = types.BoolValue(getResult.CurrentUser.Repo.GithubIntegration.AutoDeployBranch)
			data.Container.Branch.AutoStopBranch = types.BoolValue(getResult.CurrentUser.Repo.GithubIntegration.AutoStopBranch)
			data.Container.Branch.BranchIgnore = types.StringValue(getResult.CurrentUser.Repo.GithubIntegration.BranchIgnore)
			data.Container.Branch.BranchStopIgnore = types.StringValue(getResult.CurrentUser.Repo.GithubIntegration.BranchStopIgnore)
		} else if getResult.CurrentUser.Repo.GitlabIntegration != nil {
			data.Container.Branch.AutoDeployBranch = types.BoolValue(getResult.CurrentUser.Repo.GitlabIntegration.AutoDeployBranch)
			data.Container.Branch.AutoStopBranch = types.BoolValue(getResult.CurrentUser.Repo.GitlabIntegration.AutoStopBranch)
			data.Container.Branch.BranchIgnore = types.StringValue(getResult.CurrentUser.Repo.GitlabIntegration.BranchIgnore)
			data.Container.Branch.BranchStopIgnore = types.StringValue(getResult.CurrentUser.Repo.GitlabIntegration.BranchStopIgnore)
		}

		// workflow
		data.Container.Workflow = &ProjectContainerWorkflowModel{}
		if getResult.CurrentUser.Repo.AutoRetry != nil {
			data.Container.Workflow.AutoRetry = types.BoolValue(*getResult.CurrentUser.Repo.AutoRetry)
		}
		if getResult.CurrentUser.Repo.AutoRollback != nil {
			data.Container.Workflow.AutoRollback = types.BoolValue(*getResult.CurrentUser.Repo.AutoRollback)
		}
		if getResult.CurrentUser.Repo.ManualDeploy != nil {
			data.Container.Workflow.ManualDeploy = types.BoolValue(*getResult.CurrentUser.Repo.ManualDeploy)
		}
		if getResult.CurrentUser.Repo.PipelineCluster != nil && getResult.CurrentUser.Repo.PipelineCluster.Id != uuid.Nil {
			data.Container.Workflow.PipelineClusterId = customtypes.NewUUIDValue(getResult.CurrentUser.Repo.PipelineCluster.Id)
		}
		if getResult.CurrentUser.Repo.DeployTimeoutSeconds != nil {
			data.Container.Workflow.DeployTimeoutSeconds = types.Int64Value(int64(*getResult.CurrentUser.Repo.DeployTimeoutSeconds))
		}

		// source & build
		data.Container.Source = prevContainer.Source
		data.Container.Build = prevContainer.Build
		switch getResult.CurrentUser.Repo.Source.Type {
		case zeetv0.RepoSourceTypeDocker:
			splits := strings.Split(getResult.CurrentUser.Repo.Source.Id, ":")
			val := zeetv0.ContainerRegistrySourceInput{
				Repository: splits[0],
				Tag:        &splits[1],
			}
			valJson, err := json.Marshal(val)
			if err != nil {
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read project, got error: %s", err))
				return
			}
			data.Container.Source.ContainerRegistry = jsontypes.NewNormalizedValue(string(valJson))
		case zeetv0.RepoSourceTypeGit, zeetv0.RepoSourceTypeGithub, zeetv0.RepoSourceTypeGitlab, zeetv0.RepoSourceTypeBitbucket:
			val := zeetv0.GitSourceInput{
				Repository: getResult.CurrentUser.Repo.Source.Id,
			}
			valJson, err := json.Marshal(val)
			if err != nil {
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read project, got error: %s", err))
				return
			}
			data.Container.Source.Git = jsontypes.NewNormalizedValue(string(valJson))

			// build
			buildInput := zeetv0.ResourceBuildInput{
				Build: &zeetv0.ProjectBuildInput{
					BuildType:        &getResult.CurrentUser.Repo.BuildMethod.Type,
					DockerfilePath:   getResult.CurrentUser.Repo.BuildMethod.DockerfilePath,
					WorkingDirectory: getResult.CurrentUser.Repo.BuildMethod.WorkingDirectory,
					BuildCommand:     getResult.CurrentUser.Repo.BuildMethod.BuildCommand,
					RunCommand:       getResult.CurrentUser.Repo.BuildMethod.RunCommand,
					StaticPath:       getResult.CurrentUser.Repo.BuildMethod.StaticPath,
					NodejsVersion:    getResult.CurrentUser.Repo.BuildMethod.NodejsVersion,
					PythonVersion:    getResult.CurrentUser.Repo.BuildMethod.PythonVersion,
					GolangVersion:    getResult.CurrentUser.Repo.BuildMethod.GolangVersion,
				},
				GitSubmodules: getResult.CurrentUser.Repo.GitSubmodules,
				// TODO: add other misc build fields
			}
			valJson, err = json.Marshal(buildInput)
			if err != nil {
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read project, got error: %s", err))
				return
			}
			data.Container.Build = jsontypes.NewNormalizedValue(string(valJson))
		default:
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read project, got error: %s", "source type not supported"))
			return
		}

		// kubernetes
		data.Container.Kubernetes = prevContainer.Kubernetes
		if getResult.CurrentUser.Repo.DeployTarget != nil && *getResult.CurrentUser.Repo.DeployTarget == zeetv0.DeployTargetKubernetes {
			input := zeetv0.ResourceKubernetesInput{
				DeployTarget: &zeetv0.ProjectDeployInput{
					DeployTarget: *getResult.CurrentUser.Repo.DeployTarget,
					ClusterID:    &getResult.CurrentUser.Repo.Cluster.Id,
				},
				Namespace: getResult.CurrentUser.Repo.Namespace,
				App: &zeetv0.ResourceKubernetesAppInput{
					DeployService: getResult.CurrentUser.Repo.DeployService,
					Ports: lo.Map(getResult.CurrentUser.Repo.Ports, func(p zeetv0.RepoNetworkPortsPort, _ int) zeetv0.PortInput {
						pi := zeetv0.PortInput{
							Port:     p.Port,
							Protocol: zeetv0.PortProtocol(p.Protocol),
							Public:   p.Public,
							Https:    p.Https,
						}
						if p.Grpc {
							pi.Grpc = &p.Grpc
						}
						return pi
					}),
				},
			}

			if getResult.CurrentUser.Repo.UseHumanReadableKubernetesName != nil && !*getResult.CurrentUser.Repo.UseHumanReadableKubernetesName {
				// default to true
				input.App.UseHumanReadableName = getResult.CurrentUser.Repo.UseHumanReadableKubernetesName
			}

			// resource
			if getResult.CurrentUser.Repo.Cpu != nil && getResult.CurrentUser.Repo.Memory != nil &&
				*getResult.CurrentUser.Repo.Cpu != "" && *getResult.CurrentUser.Repo.Memory != "" {
				input.App.Resources = &zeetv0.ContainerResourcesSpecInput{
					Cpu:              lo.Must(strconv.ParseFloat(*getResult.CurrentUser.Repo.Cpu, 64)),
					Memory:           lo.Must(strconv.ParseFloat(*getResult.CurrentUser.Repo.Memory, 64)),
					EphemeralStorage: getResult.CurrentUser.Repo.EphemeralStorage,
					Spot:             lo.ToPtr(*getResult.CurrentUser.Repo.Dedicated),
					// Accelerator: &zeetv0.ContainerResourcesAcceleratorSpecInput{},
				}
			}
			valJson, err := json.Marshal(input)
			if err != nil {
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read project, got error: %s", err))
				return
			}
			data.Container.Kubernetes = jsontypes.NewNormalizedValue(string(valJson))
		} else {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read project, got error: %s", "deploy target not kubernetes"))
			return
		}
	} else if data.IsWorkflow() {
		readResult, err := zeetv1.ProjectDetailQuery(ctx, r.client.ClientV1(), data.TeamId.ValueUUID(), data.Id.ValueUUID())
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read project, got error: %s", err))
			return
		}

		data.Name = types.StringValue(readResult.Team.Project.Name)
		data.Enabled = types.BoolValue(readResult.Team.Project.Status == zeetv1.ProjectStatusPaused)

		// workflow
		data.Workflow.Id = customtypes.NewUUIDValue(readResult.Team.Project.Workflow.Id)

		// deploys
		for i, deploy := range readResult.Team.Project.Deploys.Nodes {
			data.Deploys[i].Id = customtypes.NewUUIDValue(deploy.Id)
			data.Deploys[i].DefaultWorkflowSteps = lo.Map(deploy.Configuration.DefaultWorkflowSteps, func(s zeetv1.BlueprintDriverWorkflowStepAction, _ int) types.String {
				return types.StringValue(string(s))
			})
			if deploy.Configuration.RequirePlanApproval != nil {
				data.Deploys[i].RequirePlanApproval = types.BoolValue(*deploy.Configuration.RequirePlanApproval)
			}
			if deploy.Configuration.Variables != nil && len(deploy.Configuration.Variables) > 0 {
				valJson, err := json.Marshal(deploy.Configuration.Variables)
				if err != nil {
					resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read project, got error: %s", err))
					return
				}
				input := []zeetv1.BlueprintVariableInput{}
				if err := json.Unmarshal(valJson, &input); err != nil {
					resp.Diagnostics.AddError("Invalid Configuration", fmt.Sprintf("Unable to unmarshal variables, got error: %s", err))
					return
				}
				for j := range input {
					input[j].SpecId = nil
					if deploy.Configuration.Variables[j].ValueString != nil {
						input[j].Type = lo.ToPtr(zeetv1.BlueprintVariableTypeString)
						input[j].Value = *deploy.Configuration.Variables[j].ValueString
					} else if deploy.Configuration.Variables[j].ValueInt != nil {
						input[j].Type = lo.ToPtr(zeetv1.BlueprintVariableTypeInteger)
						input[j].Value = strconv.FormatInt(int64(*deploy.Configuration.Variables[j].ValueInt), 10)
					} else if deploy.Configuration.Variables[j].ValueFloat != nil {
						input[j].Type = lo.ToPtr(zeetv1.BlueprintVariableTypeFloat)
						input[j].Value = strconv.FormatFloat(*deploy.Configuration.Variables[j].ValueFloat, 'f', -1, 64)
					} else if deploy.Configuration.Variables[j].ValueBoolean != nil {
						input[j].Type = lo.ToPtr(zeetv1.BlueprintVariableTypeBoolean)
						input[j].Value = strconv.FormatBool(*deploy.Configuration.Variables[j].ValueBoolean)
					} else if deploy.Configuration.Variables[j].ValueJson != nil {
						input[j].Type = lo.ToPtr(zeetv1.BlueprintVariableTypeJson)
						input[j].Value = *deploy.Configuration.Variables[j].ValueJson
					}
				}
				inputJson, err := json.Marshal(input)
				if err != nil {
					resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read project, got error: %s", err))
					return
				}
				data.Deploys[i].Variables = jsontypes.NewNormalizedValue(string(inputJson))
			}
			if deploy.Configuration.Kubernetes != nil {
				// TODO: explicit transformation
				valJson, err := json.Marshal(deploy.Configuration.Kubernetes)
				if err != nil {
					resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read project, got error: %s", err))
					return
				}
				data.Deploys[i].Kubernetes = jsontypes.NewNormalizedValue(string(valJson))
			}
			if deploy.Configuration.Helm != nil {
				// TODO: explicit transformation
				valJson, err := json.Marshal(deploy.Configuration.Helm)
				if err != nil {
					resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read project, got error: %s", err))
					return
				}
				data.Deploys[i].Helm = jsontypes.NewNormalizedValue(string(valJson))
			}
			if deploy.Configuration.Terraform != nil {
				valJson, err := json.Marshal(deploy.Configuration.Terraform)
				if err != nil {
					resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read project, got error: %s", err))
					return
				}
				input := zeetv1.DeploymentConfigurationTerraformInput{}
				if err := json.Unmarshal(valJson, &input); err != nil {
					resp.Diagnostics.AddError("Invalid Configuration", fmt.Sprintf("Unable to unmarshal terraform, got error: %s", err))
					return
				}
				// manually fix the input
				if deploy.Configuration.Terraform.Target.Provider.AwsAccount != nil {
					input.Target.Provider.AwsAccountId = lo.ToPtr(deploy.Configuration.Terraform.Target.Provider.AwsAccount.Id)
				} else if deploy.Configuration.Terraform.Target.Provider.GcpAccount != nil {
					input.Target.Provider.GcpAccountId = lo.ToPtr(deploy.Configuration.Terraform.Target.Provider.GcpAccount.Id)
				} else {
					resp.Diagnostics.AddError("Invalid Configuration", "Terraform provider must be either AWS or GCP")
					return
				}

				if deploy.Configuration.Terraform.Blueprint.OutputConfiguration != nil &&
					lo.IsEmpty(*deploy.Configuration.Terraform.Blueprint.OutputConfiguration) {
					input.Blueprint.OutputConfiguration = nil
				}

				if deploy.Configuration.Terraform.Blueprint.Source.TerraformModule != nil && deploy.Configuration.Terraform.Blueprint.Source.TerraformModule.Integration != nil &&
					lo.IsEmpty(*deploy.Configuration.Terraform.Blueprint.Source.TerraformModule.Integration) {
					input.Blueprint.Source.TerraformModule.Integration = nil
				}

				inputJson, err := json.Marshal(input)
				if err != nil {
					resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read project, got error: %s", err))
					return
				}
				data.Deploys[i].Terraform = jsontypes.NewNormalizedValue(string(inputJson))
			}
		}
	} else {
		// Not valid
		resp.Diagnostics.AddError("Invalid Configuration", "Project must have either a container or workflow configuration")
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ProjectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state ProjectResourceModel
	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// update logic
	if state.IsContainer() && plan.IsContainer() {
		if !plan.Name.Equal(state.Name) {
			_, err := zeetv0.UpdateProjectSettingsMutation(ctx, r.client.Client(), zeetv0.UpdateProjectInput{
				Id:   state.Id.ValueUUID().String(),
				Name: lo.ToPtr(state.Name.ValueString()),
			})
			if err != nil {
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update project, got error: %s", err))
				return
			}
		}
		// Update Logic

		updateInput := zeetv0.UpdateResourceAlphaInput{}

		// source
		if !plan.Container.Source.Git.IsNull() && !state.Container.Source.Git.IsNull() && !plan.Container.Source.Git.Equal(state.Container.Source.Git) {
			updateInput.Source = &zeetv0.SourceInput{
				Git: &zeetv0.GitSourceInput{},
			}
			if err := json.Unmarshal([]byte(plan.Container.Source.Git.ValueString()), &updateInput.Source.Git); err != nil {
				resp.Diagnostics.AddError("Invalid Configuration", fmt.Sprintf("Unable to unmarshal git source, got error: %s", err))
				return
			}
		} else if !plan.Container.Source.ContainerRegistry.IsNull() && !state.Container.Source.ContainerRegistry.IsNull() && !plan.Container.Source.ContainerRegistry.Equal(state.Container.Source.ContainerRegistry) {
			updateInput.Source = &zeetv0.SourceInput{
				ContainerRegistry: &zeetv0.ContainerRegistrySourceInput{},
			}
			if err := json.Unmarshal([]byte(plan.Container.Source.ContainerRegistry.ValueString()), &updateInput.Source.ContainerRegistry); err != nil {
				resp.Diagnostics.AddError("Invalid Configuration", fmt.Sprintf("Unable to unmarshal container registry source, got error: %s", err))
				return
			}
		} else {
			resp.Diagnostics.AddError("Invalid Configuration", "Container source must be either git or container registry, and the type must not change")
			return
		}

		if updateInput.Source != nil {
			_, err := zeetv0.UpdateResourceAlphaMutation(ctx, r.client.Client(), state.Container.RepoId.ValueUUID(), updateInput)
			if err != nil {
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update project, got error: %s", err))
				return
			}
		}

		updateRepoInput := zeetv0.UpdateProjectInput{
			Id: state.Container.RepoId.ValueUUID().String(),
		}
		// TODO: branch

		// workflow
		updateRepoInput.AutoRetry = lo.ToPtr(state.Container.Workflow.AutoRetry.ValueBool())
		updateRepoInput.AutoRollback = lo.ToPtr(state.Container.Workflow.AutoRollback.ValueBool())
		updateRepoInput.ManualDeploy = lo.ToPtr(state.Container.Workflow.ManualDeploy.ValueBool())
		updateRepoInput.PipelineClusterID = lo.ToPtr(state.Container.Workflow.PipelineClusterId.ValueUUID())
		updateRepoInput.DeployTimeoutSeconds = lo.ToPtr(int(state.Container.Workflow.DeployTimeoutSeconds.ValueInt64()))

		// TODO: build

		// TODO: kubernetes

		// Toggle deployment last
		if !plan.Enabled.Equal(state.Enabled) {
			if plan.Enabled.ValueBool() {
				_, err := zeetv0.EnableProjectMutation(ctx, r.client.Client(), state.Container.RepoId.ValueUUID().String())
				if err != nil {
					resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to resume project, got error: %s", err))
					return
				}
			} else {
				_, err := zeetv0.DisableProjectMutation(ctx, r.client.Client(), state.Container.RepoId.ValueUUID().String())
				if err != nil {
					resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to pause project, got error: %s", err))
					return
				}
			}
		}
	} else if state.IsWorkflow() && plan.IsWorkflow() {
		if !plan.Name.Equal(state.Name) {
			_, err := zeetv1.UpdateProjectMutation(ctx, r.client.ClientV1(), state.Id.ValueUUID(), zeetv1.UpdateProjectInput{
				Name: lo.ToPtr(state.Name.ValueString()),
			})
			if err != nil {
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update project, got error: %s", err))
				return
			}
		}

		// Update Logic
		for i, deploy := range plan.Deploys {
			input := &zeetv1.DeploymentConfigurationInput{
				DefaultWorkflowSteps: lo.Map(deploy.DefaultWorkflowSteps, func(s types.String, _ int) zeetv1.BlueprintDriverWorkflowStepAction {
					return zeetv1.BlueprintDriverWorkflowStepAction(s.ValueString())
				}),
			}

			if !deploy.RequirePlanApproval.IsNull() {
				input.RequirePlanApproval = lo.ToPtr(deploy.RequirePlanApproval.ValueBool())
			}

			if !deploy.Helm.IsNull() {
				input.Helm = &zeetv1.DeploymentConfigurationKubernetesHelmInput{}
				if err := json.Unmarshal([]byte(deploy.Helm.ValueString()), &input.Helm); err != nil {
					resp.Diagnostics.AddError("Invalid Configuration", fmt.Sprintf("Unable to unmarshal helm, got error: %s", err))
					return
				}
			}

			if !deploy.Kubernetes.IsNull() {
				input.Kubernetes = &zeetv1.DeploymentConfigurationKubernetesInput{}
				if err := json.Unmarshal([]byte(deploy.Kubernetes.ValueString()), &input.Kubernetes); err != nil {
					resp.Diagnostics.AddError("Invalid Configuration", fmt.Sprintf("Unable to unmarshal kubernetes, got error: %s", err))
					return
				}
			}

			if !deploy.Terraform.IsNull() {
				input.Terraform = &zeetv1.DeploymentConfigurationTerraformInput{}
				if err := json.Unmarshal([]byte(deploy.Terraform.ValueString()), &input.Terraform); err != nil {
					resp.Diagnostics.AddError("Invalid Configuration", fmt.Sprintf("Unable to unmarshal terraform, got error: %s", err))
					return
				}
			}

			if !deploy.Variables.IsNull() {
				input.Variables = []zeetv1.BlueprintVariableInput{}
				if err := json.Unmarshal([]byte(deploy.Variables.ValueString()), &input.Variables); err != nil {
					resp.Diagnostics.AddError("Invalid Configuration", fmt.Sprintf("Unable to unmarshal variables, got error: %s", err))
					return
				}
			}

			if _, err := zeetv1.UpdateDeployMutation(ctx, r.client.ClientV1(), state.Deploys[i].Id.ValueUUID(), zeetv1.UpdateDeployInput{
				Configuration: input,
			}); err != nil {
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update deploy, got error: %s", err))
				return
			}
		}

		if !plan.Workflow.Steps.Equal(state.Workflow.Steps) {
			input := &zeetv1.WorkflowDefinitionInput{
				Steps: []zeetv1.WorkflowStepDefinitionInput{},
			}
			if err := json.Unmarshal([]byte(plan.Workflow.Steps.ValueString()), &input.Steps); err != nil {
				resp.Diagnostics.AddError("Invalid Configuration", fmt.Sprintf("Unable to unmarshal workflow steps, got error: %s", err))
				return
			}
			_, err := zeetv1.UpdateWorkflowMutation(ctx, r.client.ClientV1(), state.Workflow.Id.ValueUUID(), zeetv1.UpdateWorkflowInput{
				Definition: input,
			})
			if err != nil {
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update project, got error: %s", err))
				return
			}
		}

		// Toggle deployment last
		if !plan.Enabled.Equal(state.Enabled) {
			if plan.Enabled.ValueBool() {
				_, err := zeetv1.SubmitWorkflowRunMutation(ctx, r.client.ClientV1(), state.Workflow.Id.ValueUUID(), nil)
				if err != nil {
					resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to resume project, got error: %s", err))
					return
				}
			} else {
				_, err := zeetv1.DeleteProjectResourcesMutation(ctx, r.client.ClientV1(), state.Id.ValueUUID())
				if err != nil {
					resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to pause project, got error: %s", err))
					return
				}
			}
		}
	} else {
		// Not valid
		resp.Diagnostics.AddError("Invalid Configuration", "Project must have either a container or workflow configuration, and it must not change")
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ProjectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ProjectResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if data.IsContainer() {
		_, err := zeetv0.DeleteProjectMutation(ctx, r.client.Client(), data.Container.RepoId.ValueUUID().String())
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete project, got error: %s", err))
			return
		}
	} else if data.IsWorkflow() {
		_, err := zeetv1.DeleteProjectMutation(ctx, r.client.ClientV1(), data.Id.ValueUUID(), lo.ToPtr(false))
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete project, got error: %s", err))
			return
		}
	} else {
		// Not valid
		resp.Diagnostics.AddError("Invalid Configuration", "Project must have either a container or workflow configuration")
		return
	}
}

func (r *ProjectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
