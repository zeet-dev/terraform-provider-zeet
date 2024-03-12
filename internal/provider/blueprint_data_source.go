package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/zeet-dev/cli/pkg/api"
	zeetv0 "github.com/zeet-dev/cli/pkg/sdk/v0"
	zeetv1 "github.com/zeet-dev/cli/pkg/sdk/v1"
	"github.com/zeet-dev/terraform-provider-zeet/internal/provider/customtypes"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &BlueprintDataSource{}

func NewBlueprintDataSource() datasource.DataSource {
	return &BlueprintDataSource{}
}

// BlueprintDataSource defines the data source implementation.
type BlueprintDataSource struct {
	client *api.Client
}

// BlueprintDataSourceModel describes the data source data model.
type BlueprintDataSourceModel struct {
	Id            customtypes.UUIDValue        `tfsdk:"id"`
	Slug          types.String                 `tfsdk:"slug"`
	IsOfficial    types.Bool                   `tfsdk:"is_official"`
	Type          types.String                 `tfsdk:"type"`
	Configuration *BlueprintConfigurationModel `tfsdk:"configuration"`
}

type BlueprintConfigurationModel struct {
	Slug                types.String         `tfsdk:"slug"`
	DisplayName         types.String         `tfsdk:"display_name"`
	Description         types.String         `tfsdk:"description"`
	Tags                types.List           `tfsdk:"tags"`
	Enabled             types.Bool           `tfsdk:"enabled"`
	Published           types.Bool           `tfsdk:"published"`
	DriverConfiguration jsontypes.Normalized `tfsdk:"driver_configuration"`
	RichInputSchema     jsontypes.Normalized `tfsdk:"rich_input_schema"`
	Variables           jsontypes.Normalized `tfsdk:"variables"`
}

func (d *BlueprintDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_blueprint"
}

func (d *BlueprintDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Blueprint data source",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Blueprint identifier, either id or slug must be set, can be used for official and custom blueprints",
				Optional:            true,
				CustomType:          customtypes.UUIDType{},
			},
			"slug": schema.StringAttribute{
				MarkdownDescription: "Blueprint slug, either id or slug must be set, can only be used for official blueprints",
				Optional:            true,
			},
			"is_official": schema.BoolAttribute{
				MarkdownDescription: "Blueprint is official",
				Computed:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Blueprint type",
				Computed:            true,
			},
			"configuration": schema.SingleNestedAttribute{
				MarkdownDescription: "Blueprint configuration",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"slug": schema.StringAttribute{
						MarkdownDescription: "Blueprint slug",
						Computed:            true,
					},
					"display_name": schema.StringAttribute{
						MarkdownDescription: "Blueprint display name",
						Computed:            true,
					},
					"description": schema.StringAttribute{
						MarkdownDescription: "Blueprint description",
						Computed:            true,
					},
					"tags": schema.ListAttribute{
						MarkdownDescription: "List of tags associated with the blueprint",
						Optional:            true,
						ElementType:         types.StringType,
					},
					"enabled": schema.BoolAttribute{
						MarkdownDescription: "Indicates if the blueprint is enabled",
						Computed:            true,
					},
					"published": schema.BoolAttribute{
						MarkdownDescription: "Indicates if the blueprint is published",
						Computed:            true,
					},
					"driver_configuration": schema.StringAttribute{
						MarkdownDescription: "Blueprint driver configuration in [JSON format](https://docs.zeet.co/graphql/objects/blueprint-configuration/#code-style-fontweight-normal-blueprintconfigurationbdriverconfigurationbcodeblueprintdriverconfiguration-)",
						Computed:            true,
						CustomType:          jsontypes.NormalizedType{},
					},
					"rich_input_schema": schema.StringAttribute{
						MarkdownDescription: "Blueprint rich input schema in [JSON format](https://anchor.zeet.co/static/schemas/blueprint-rich-input-schema.schema.json)",
						Computed:            true,
						CustomType:          jsontypes.NormalizedType{},
					},
					"variables": schema.StringAttribute{
						MarkdownDescription: "Blueprint variables in (JSON format)[https://docs.zeet.co/graphql/objects/blueprint-configuration/#code-style-fontweight-normal-blueprintconfigurationbvariablesbcodeblueprintvariablespec--]",
						Computed:            true,
						CustomType:          jsontypes.NormalizedType{},
					},
				},
			},
		},
	}
}

func (d *BlueprintDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*api.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *api.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client
}

func (d *BlueprintDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data BlueprintDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if data.Id.IsNull() {
		if data.Slug.IsNull() {
			resp.Diagnostics.AddError("Invalid Configuration", "Either id or slug must be set")
			return
		}
		// query official blueprint by slug
		result, err := zeetv0.MarketplaceBlueprintQuery(ctx, d.client.Client(), "zeet", data.Slug.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read blueprint, got error: %s", err))
			return
		}
		data.Id = customtypes.NewUUIDValue(result.BlueprintsMarketplace.Blueprint.Id)
	}

	result, err := zeetv1.BlueprintByIdQuery(ctx, d.client.ClientV1(), data.Id.ValueUUID())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read blueprint, got error: %s", err))
		return
	}

	data.Id = customtypes.NewUUIDValue(result.Blueprint.Id)
	data.Slug = types.StringValue(result.Blueprint.Configuration.Slug)
	data.IsOfficial = types.BoolValue(*result.Blueprint.IsOfficial)
	data.Type = types.StringValue(string(result.Blueprint.Type))
	data.Configuration = &BlueprintConfigurationModel{
		Slug:        types.StringValue(result.Blueprint.Configuration.Slug),
		DisplayName: types.StringValue(result.Blueprint.Configuration.DisplayName),
		Published:   types.BoolValue(result.Blueprint.Configuration.Published),
		// DriverConfiguration: jsontypes.NewNormalizedValue(result.Blueprint.Configuration),
		// Variables: jsontypes.NewNormalizedValue(result.Blueprint.Configuration.Variables),
	}

	if result.Blueprint.Configuration.Description != nil {
		data.Configuration.Description = types.StringValue(*result.Blueprint.Configuration.Description)
	}

	if result.Blueprint.Enabled != nil {
		data.Configuration.Enabled = types.BoolValue(*result.Blueprint.Enabled)
	}

	tags, diag := types.ListValueFrom(ctx, types.StringType, result.Blueprint.Configuration.Tags)
	if diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}
	data.Configuration.Tags = tags

	if result.Blueprint.Configuration.RichInputSchema != nil {
		data.Configuration.RichInputSchema = jsontypes.NewNormalizedValue(*result.Blueprint.Configuration.RichInputSchema)
	}

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "read a data source")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
