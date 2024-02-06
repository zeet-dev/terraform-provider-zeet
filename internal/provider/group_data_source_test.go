package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGroupDataSource(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"team": map[string]any{
					"groups": map[string]any{
						"nodes": []map[string]any{
							{
								"id":   "ddf9093e-cc11-46a5-82c7-fc99fc44ef93",
								"name": "test",
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
					resource.TestCheckResourceAttr("data.zeet_group.test", "team_id", "99c11487-1683-4e10-9620-94d9a78a0b67"),
					resource.TestCheckResourceAttr("data.zeet_group.test", "id", "ddf9093e-cc11-46a5-82c7-fc99fc44ef93"),
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
