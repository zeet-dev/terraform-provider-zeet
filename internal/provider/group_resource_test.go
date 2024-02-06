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

func TestAccGroupResource(t *testing.T) {
	var created bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		reqs := string(req)
		if strings.Contains(reqs, "mutation createGroup") && strings.Contains(reqs, "one") {
			json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"createGroup": map[string]any{
						"id":   "69a5f7df-048d-4fc3-885d-178cdcb9b180",
						"name": "one",
					},
				},
			})
		} else if strings.Contains(reqs, "mutation updateGroup") && strings.Contains(reqs, "two") {
			json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"updateGroup": map[string]any{
						"id":   "69a5f7df-048d-4fc3-885d-178cdcb9b180",
						"name": "two",
					},
				},
			})
		} else if strings.Contains(reqs, "query group") {
			if !created {
				json.NewEncoder(w).Encode(map[string]any{
					"data": map[string]any{
						"team": map[string]any{
							"groups": map[string]any{
								"nodes": []map[string]any{
									{
										"id":   "ddf9093e-cc11-46a5-82c7-fc99fc44ef93",
										"name": "one",
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
										"id":   "ddf9093e-cc11-46a5-82c7-fc99fc44ef93",
										"name": "two",
									},
								},
							},
						},
					},
				})
			}
		} else if strings.Contains(reqs, "mutation deleteGroup") {
			json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"deleteGroup": true,
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
				Config: testAccGroupResourceConfig(server.URL, "one"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zeet_group.test", "name", "one"),
					resource.TestCheckResourceAttr("zeet_group.test", "team_id", "99c11487-1683-4e10-9620-94d9a78a0b67"),
					resource.TestCheckResourceAttr("zeet_group.test", "id", "69a5f7df-048d-4fc3-885d-178cdcb9b180"),
				),
			},
			// Update and Read testing
			{
				Config: testAccGroupResourceConfig(server.URL, "two"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zeet_group.test", "name", "two"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccGroupResourceConfig(server string, name string) string {
	return fmt.Sprintf(`
provider "zeet" {
  api_url = %[1]q
}

resource "zeet_group" "test" {
  team_id = "99c11487-1683-4e10-9620-94d9a78a0b67"
  name = %[2]q
}
`, server, name)
}
