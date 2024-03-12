package provider_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/samber/lo"

	zeetv1 "github.com/zeet-dev/cli/pkg/sdk/v1"
)

func TestAccBlueprintDataSource(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"data": zeetv1.BlueprintResponse{
				Team: &zeetv1.BlueprintTeam{
					Id: testTeamId,
					Blueprint: &zeetv1.BlueprintTeamBlueprint{
						BlueprintDetail: zeetv1.BlueprintDetail{
							Id:         testBlueprintId,
							Type:       zeetv1.BlueprintTypeTerraform,
							IsOfficial: lo.ToPtr(false),
							Enabled:    lo.ToPtr(true),
							Configuration: zeetv1.BlueprintDetailConfigurationBlueprintConfiguration{
								BlueprintConfigurationDetail: zeetv1.BlueprintConfigurationDetail{
									Slug:            "route53-delegation",
									DisplayName:     "route53 delegation",
									Published:       true,
									Description:     lo.ToPtr("route53 delegation blueprint"),
									Tags:            []string{"route53", "delegation"},
									RichInputSchema: lo.ToPtr(`{"test": "value"}`),
								},
							},
						},
					},
				},
			},
		})
	}))
	defer server.Close()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: fmt.Sprintf(testAccBlueprintDataSourceConfig, server.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.zeet_blueprint.test", "team_id", testTeamId.String()),
					resource.TestCheckResourceAttr("data.zeet_blueprint.test", "id", testBlueprintId.String()),
				),
			},
		},
	})
}

const testAccBlueprintDataSourceConfig = `
provider "zeet" {
  api_url = "%s"
}

data "zeet_blueprint" "test" {
  team_id = "99c11487-1683-4e10-9620-94d9a78a0b67"
  id = "2e9aa322-3a41-4930-9f3c-2987836d3b70"
}
`
