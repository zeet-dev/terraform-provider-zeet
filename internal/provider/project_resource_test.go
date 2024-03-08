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

func TestAccProjectResourceHelm(t *testing.T) {
	var created bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		reqs := string(req)
		if strings.Contains(reqs, "mutation createProject") && strings.Contains(reqs, "one") {
			json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"createProject": map[string]any{
						"id":   "69a5f7df-048d-4fc3-885d-178cdcb9b180",
						"name": "one",
					},
				},
			})
		} else if strings.Contains(reqs, "mutation updateProject") && strings.Contains(reqs, "two") {
			json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"updateProject": map[string]any{
						"id":   "69a5f7df-048d-4fc3-885d-178cdcb9b180",
						"name": "two",
					},
				},
			})
		} else if strings.Contains(reqs, "query project") {
			if !created {
				json.NewEncoder(w).Encode(map[string]any{
					"data": map[string]any{
						"team": map[string]any{
							"project": map[string]any{
								"id":   "ddf9093e-cc11-46a5-82c7-fc99fc44ef93",
								"name": "one",
							},
						},
					},
				})
				created = true
			} else {
				json.NewEncoder(w).Encode(map[string]any{
					"data": map[string]any{
						"team": map[string]any{
							"project": map[string]any{
								"id":   "ddf9093e-cc11-46a5-82c7-fc99fc44ef93",
								"name": "one",
							},
						},
					},
				})
			}
		} else if strings.Contains(reqs, "mutation deleteProject") {
			json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"deleteProject": true,
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
				Config: testAccProjectResourceConfigWithHelmDeployment(server.URL, "one", "5a0e108d-6df6-456d-aa3a-a89e78b57cf6"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zeet_project.test_helm", "name", "one"),
					resource.TestCheckResourceAttr("zeet_project.test_helm", "team_id", "99c11487-1683-4e10-9620-94d9a78a0b67"),
					resource.TestCheckResourceAttr("zeet_project.test_helm", "id", "69a5f7df-048d-4fc3-885d-178cdcb9b180"),
				),
			},
			// TODO: Update and Read testing
			// {
			// 	Config: testAccProjectResourceConfigWithHelmDeployment(server.URL, "two", "5a0e108d-6df6-456d-aa3a-a89e78b57cf6"),
			// 	Check: resource.ComposeAggregateTestCheckFunc(
			// 		resource.TestCheckResourceAttr("zeet_project.test", "name", "two"),
			// 	),
			// },
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccProjectResourceConfigWithHelmDeployment(server string, name string, clusterID string) string {
	return fmt.Sprintf(`
provider "zeet" {
  api_url = %[1]q
}

resource "zeet_project" "test_helm" {
  team_id = "99c11487-1683-4e10-9620-94d9a78a0b67"
  group_id = "ddf9093e-cc11-46a5-82c7-fc99fc44ef93"
  subgroup_id = "149ad8a9-cb35-477b-bbac-39a39f146074"

  name = %[2]q
  blueprint_id = "5a0e108d-6df6-456d-aa3a-a89e78b57cf6"

  deploys = [{
	default_workflow_steps = ["DRIVER_PLAN", "DRIVER_APPROVE", "DRIVER_APPLY"]
	helm = jsonencode({
	  blueprint: {
		source: {
		  helmRepository: {
			repositoryUrl: "https://grafana.github.io/helm-charts",
			chart: "grafana"
		  }
		}
	  },
	  target: {
		clusterId: %[3]q,
		namespace: "grafana",
		releaseName: "grafana"
	  }
	})
  }]

  workflow = {
    steps: jsonencode([{ action: "ORCHESTRATION_DEPLOY" }])
  }

  enabled = true
}
`, server, name, clusterID)
}

func TestAccProjectResourceContainer(t *testing.T) {
	var created bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		reqs := string(req)
		if strings.Contains(reqs, "mutation createResourceAlpha") && strings.Contains(reqs, "one") {
			json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"createResourceAlpha": map[string]any{
						"name": "one",
						"project": map[string]any{
							"name": "p",
						},
						"projectEnvironment": map[string]any{
							"name": "e",
						},
					},
				},
			})
		} else if strings.Contains(reqs, "mutation updateProject") && strings.Contains(reqs, "two") {
			json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"updateProject": map[string]any{
						"id":   "69a5f7df-048d-4fc3-885d-178cdcb9b180",
						"name": "two",
					},
				},
			})
		} else if strings.Contains(reqs, "query user") {
			json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"user": map[string]any{
						"id":    "99c11487-1683-4e10-9620-94d9a78a0b67",
						"login": "test",
					},
				},
			})
		} else if strings.Contains(reqs, "query repoForProjectEnvironment") {
			if !created {
				json.NewEncoder(w).Encode(map[string]any{
					"data": map[string]any{
						"project": map[string]any{
							"name": "p",
							"environment": map[string]any{
								"name": "e",
								"repo": map[string]any{
									"id":   "69a5f7df-048d-4fc3-885d-178cdcb9b180",
									"name": "one",
								},
							},
						},
					},
				})
				created = true
			} else {
				json.NewEncoder(w).Encode(map[string]any{
					"data": map[string]any{
						"project": map[string]any{
							"environment": map[string]any{
								"repo": map[string]any{
									"id":   "69a5f7df-048d-4fc3-885d-178cdcb9b180",
									"name": "two",
								},
							},
						},
					},
				})
			}
		} else if strings.Contains(reqs, "mutation deleteProject") {
			json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"deleteProject": true,
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
				Config: testAccProjectResourceConfigWithContainerDeployment(server.URL, "one", "5a0e108d-6df6-456d-aa3a-a89e78b57cf6"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zeet_project.test_container", "name", "one"),
					resource.TestCheckResourceAttr("zeet_project.test_container", "team_id", "99c11487-1683-4e10-9620-94d9a78a0b67"),
					resource.TestCheckResourceAttr("zeet_project.test_container", "id", "69a5f7df-048d-4fc3-885d-178cdcb9b180"),
				),
			},
			// TODO: Update and Read testing
			// {
			// 	Config: testAccProjectResourceConfigWithContainerDeployment(server.URL, "two", "5a0e108d-6df6-456d-aa3a-a89e78b57cf6"),
			// 	Check: resource.ComposeAggregateTestCheckFunc(
			// 		resource.TestCheckResourceAttr("zeet_project.test", "name", "two"),
			// 	),
			// },
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccProjectResourceConfigWithContainerDeployment(server string, name string, clusterID string) string {
	return fmt.Sprintf(`
provider "zeet" {
  api_url = %[1]q
}

resource "zeet_project" "test_container" {
  team_id = "99c11487-1683-4e10-9620-94d9a78a0b67"
  group_id = "ddf9093e-cc11-46a5-82c7-fc99fc44ef93"
  subgroup_id = "149ad8a9-cb35-477b-bbac-39a39f146074"

  name = %[2]q
  blueprint_id = "5a0e108d-6df6-456d-aa3a-a89e78b57cf6"

  container = {
    source = {
      git = jsonencode({
        repository: "https://github.com/zeet-demo/node-express-demo.git"
      })
    }
    build = jsonencode({
      build: {
        buildType: "NODE",
        buildCommand: "npm --production=false install",
        nodejsVersion: "18",
        runCommand: "npm start",
        workingDirectory: "./"
      }
    })
    kubernetes = jsonencode({
      deployTarget: {
        deployTarget: "KUBERNETES",
        clusterID: %[3]q
      },
      app: {
        deployService: true,
        volumes: [],
        envs: [],
        ports: [
          {
            port: "3000",
            protocol: "tcp",
            public: true,
            https: true
          }
        ],
        resources: {
          cpu: 1,
          memory: 1,
          spot: false
        }
      }
    })
  }

  enabled = true
}
`, server, name, clusterID)
}
