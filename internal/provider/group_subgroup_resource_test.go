package provider_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	zeetv1 "github.com/zeet-dev/cli/pkg/sdk/v1"
)

func TestAccGroupSubgroupResource(t *testing.T) {
	var created bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		reqs := string(req)
		if strings.Contains(reqs, "mutation createSubGroup") && strings.Contains(reqs, "one") {
			json.NewEncoder(w).Encode(map[string]any{
				"data": zeetv1.CreateSubGroupResponse{
					CreateSubGroup: zeetv1.CreateSubGroupCreateSubGroup{
						Id:   testSubGroupId,
						Name: "one",
					},
				},
			})
		} else if strings.Contains(reqs, "mutation updateSubGroup") && strings.Contains(reqs, "two") {
			json.NewEncoder(w).Encode(map[string]any{
				"data": zeetv1.UpdateSubGroupResponse{
					UpdateSubGroup: zeetv1.UpdateSubGroupUpdateSubGroup{
						Id:   testSubGroupId,
						Name: "two",
					},
				},
			})
		} else if strings.Contains(reqs, "query subGroup") {
			if !created {
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
				created = true
			} else {
				json.NewEncoder(w).Encode(map[string]any{
					"data": zeetv1.SubGroupResponse{
						Team: &zeetv1.SubGroupTeam{
							Groups: zeetv1.SubGroupTeamGroupsGroupConnection{
								Nodes: []zeetv1.SubGroupTeamGroupsGroupConnectionNodesGroup{
									{
										Id: testGroupId,
										SubGroup: zeetv1.SubGroupTeamGroupsGroupConnectionNodesGroupSubGroup{
											Id:   testSubGroupId,
											Name: "two",
										},
									},
								},
							},
						},
					},
				})
			}
		} else if strings.Contains(reqs, "mutation deleteSubGroup") {
			json.NewEncoder(w).Encode(map[string]any{
				"data": zeetv1.DeleteSubGroupResponse{
					DeleteSubGroup: true,
				},
			})
		} else {
			t.Fatal("unexpected request")
		}
	}))

	defer server.Close()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccGroupSubgroupResourceConfig(server.URL, "one"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zeet_group_subgroup.test", "name", "one"),
					resource.TestCheckResourceAttr("zeet_group_subgroup.test", "group_id", testGroupId.String()),
					resource.TestCheckResourceAttr("zeet_group_subgroup.test", "team_id", testTeamId.String()),
					resource.TestCheckResourceAttr("zeet_group_subgroup.test", "id", testSubGroupId.String()),
				),
			},
			// Update and Read testing
			{
				Config: testAccGroupSubgroupResourceConfig(server.URL, "two"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zeet_group_subgroup.test", "name", "two"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccGroupSubgroupResourceConfig(server string, name string) string {
	return fmt.Sprintf(`
provider "zeet" {
  api_url = %[1]q
}

resource "zeet_group_subgroup" "test" {
  team_id = "99c11487-1683-4e10-9620-94d9a78a0b67"
  group_id = "ddf9093e-cc11-46a5-82c7-fc99fc44ef93"

  name = %[2]q
}
`, server, name)
}
