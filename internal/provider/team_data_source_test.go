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

func TestAccTeamDataSource(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"data": zeetv1.TeamResponse{
				Team: &zeetv1.TeamTeam{
					Id:   testTeamId,
					Name: "test",
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
				Config: fmt.Sprintf(testAccTeamDataSourceConfig, server.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.zeet_team.test", "id", testTeamId.String()),
					resource.TestCheckResourceAttr("data.zeet_team.test", "name", "test"),
				),
			},
		},
	})
}

const testAccTeamDataSourceConfig = `
provider "zeet" {
  api_url = "%s"
}

data "zeet_team" "test" {
  id = "99c11487-1683-4e10-9620-94d9a78a0b67"
}
`
