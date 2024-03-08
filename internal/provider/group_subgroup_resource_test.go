package provider

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
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
				"data": map[string]any{
					"createSubGroup": map[string]any{
						"id":   "149ad8a9-cb35-477b-bbac-39a39f146074",
						"name": "one",
					},
				},
			})
		} else if strings.Contains(reqs, "mutation updateSubGroup") && strings.Contains(reqs, "two") {
			json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"updateSubGroup": map[string]any{
						"id":   "149ad8a9-cb35-477b-bbac-39a39f146074",
						"name": "two",
					},
				},
			})
		} else if strings.Contains(reqs, "query subGroup") {
			if !created {
				json.NewEncoder(w).Encode(map[string]any{
					"data": map[string]any{
						"team": map[string]any{
							"groups": map[string]any{
								"nodes": []map[string]any{
									{
										"id": "ddf9093e-cc11-46a5-82c7-fc99fc44ef93",
										"subGroup": map[string]any{
											"id":   "149ad8a9-cb35-477b-bbac-39a39f146074",
											"name": "one",
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
					"data": map[string]any{
						"team": map[string]any{
							"groups": map[string]any{
								"nodes": []map[string]any{
									{
										"id": "ddf9093e-cc11-46a5-82c7-fc99fc44ef93",
										"subGroup": map[string]any{
											"id":   "149ad8a9-cb35-477b-bbac-39a39f146074",
											"name": "two",
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
				"data": map[string]any{
					"deleteSubGroup": true,
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
					resource.TestCheckResourceAttr("zeet_group_subgroup.test", "group_id", "ddf9093e-cc11-46a5-82c7-fc99fc44ef93"),
					resource.TestCheckResourceAttr("zeet_group_subgroup.test", "team_id", "99c11487-1683-4e10-9620-94d9a78a0b67"),
					resource.TestCheckResourceAttr("zeet_group_subgroup.test", "id", "149ad8a9-cb35-477b-bbac-39a39f146074"),
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
