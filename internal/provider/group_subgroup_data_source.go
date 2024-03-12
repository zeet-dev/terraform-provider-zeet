package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/zeet-dev/cli/pkg/api"
	zeetv1 "github.com/zeet-dev/cli/pkg/sdk/v1"
	"github.com/zeet-dev/terraform-provider-zeet/internal/provider/customtypes"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &GroupSubGroupDataSource{}

func NewGroupSubGroupDataSource() datasource.DataSource {
	return &GroupSubGroupDataSource{}
}

// GroupSubGroupDataSource defines the data source implementation.
type GroupSubGroupDataSource struct {
	client *api.Client
}

// GroupSubGroupDataSourceModel describes the data source data model.
type GroupSubGroupDataSourceModel struct {
	Name    types.String          `tfsdk:"name"`
	GroupId customtypes.UUIDValue `tfsdk:"group_id"`
	TeamId  customtypes.UUIDValue `tfsdk:"team_id"`
	Id      customtypes.UUIDValue `tfsdk:"id"`
}

func (d *GroupSubGroupDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_subgroup"
}

func (d *GroupSubGroupDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "GroupSubGroup data source",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "GroupSubGroup identifier",
				Required:            true,
				CustomType:          customtypes.UUIDType{},
			},
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
			"name": schema.StringAttribute{
				MarkdownDescription: "GroupSubGroup name",
				Computed:            true,
			},
		},
	}
}

func (d *GroupSubGroupDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *GroupSubGroupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data GroupSubGroupDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	result, err := zeetv1.SubGroupQuery(ctx, d.client.ClientV1(), data.TeamId.ValueUUID(), data.GroupId.ValueUUID(), data.Id.ValueUUID())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read groupsubgroup, got error: %s", err))
		return
	}

	data.Id = customtypes.NewUUIDValue(result.Team.Groups.Nodes[0].SubGroup.Id)
	data.Name = types.StringValue(result.Team.Groups.Nodes[0].SubGroup.Name)

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "read a data source")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
