package provider_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	zeetv1 "github.com/zeet-dev/cli/pkg/sdk/v1"
)

func TestAccGroupSubGroupDataSource(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"data": zeetv1.SubGroupResponse{
				Team: &zeetv1.SubGroupTeam{
					Groups: zeetv1.SubGroupTeamGroupsGroupConnection{
						Nodes: []zeetv1.SubGroupTeamGroupsGroupConnectionNodesGroup{
							{
								Id: testGroupId,
								SubGroup: zeetv1.SubGroupTeamGroupsGroupConnectionNodesGroupSubGroup{
									Id:   testSubGroupId,
									Name: "one",
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
				Config: fmt.Sprintf(testAccGroupSubGroupDataSourceConfig, server.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.zeet_group_subgroup.test", "team_id", testTeamId.String()),
					resource.TestCheckResourceAttr("data.zeet_group_subgroup.test", "group_id", testGroupId.String()),
					resource.TestCheckResourceAttr("data.zeet_group_subgroup.test", "id", testSubGroupId.String()),
					resource.TestCheckResourceAttr("data.zeet_group_subgroup.test", "name", "one"),
				),
			},
		},
	})
}

const testAccGroupSubGroupDataSourceConfig = `
provider "zeet" {
  api_url = "%s"
}

data "zeet_group_subgroup" "test" {
  team_id = "99c11487-1683-4e10-9620-94d9a78a0b67"
  group_id = "ddf9093e-cc11-46a5-82c7-fc99fc44ef93"
  id = "ddf9093e-cc11-46a5-82c7-fc99fc44ef93"
}
`
