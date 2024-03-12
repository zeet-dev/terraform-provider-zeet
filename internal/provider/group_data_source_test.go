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

func TestAccGroupDataSource(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"data": zeetv1.GroupResponse{
				Team: &zeetv1.GroupTeam{
					Groups: zeetv1.GroupTeamGroupsGroupConnection{
						Nodes: []zeetv1.GroupTeamGroupsGroupConnectionNodesGroup{
							{
								Id:   testGroupId,
								Name: "test",
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
				Config: fmt.Sprintf(testAccGroupDataSourceConfig, server.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.zeet_group.test", "team_id", testTeamId.String()),
					resource.TestCheckResourceAttr("data.zeet_group.test", "id", testGroupId.String()),
				),
			},
		},
	})
}

const testAccGroupDataSourceConfig = `
provider "zeet" {
  api_url = "%s"
}

data "zeet_group" "test" {
  team_id = "99c11487-1683-4e10-9620-94d9a78a0b67"
  id = "ddf9093e-cc11-46a5-82c7-fc99fc44ef93"
}
`
