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

	zeetv0 "github.com/zeet-dev/cli/pkg/sdk/v0"
	zeetv1 "github.com/zeet-dev/cli/pkg/sdk/v1"
)

func TestAccProjectResourceHelm(t *testing.T) {
	readCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		reqs := string(req)
		if strings.Contains(reqs, "mutation createProject") && strings.Contains(reqs, "one") {
			json.NewEncoder(w).Encode(map[string]any{
				"data": zeetv1.CreateProjectResponse{
					CreateProject: zeetv1.CreateProjectCreateProject{
						Id:   testProjectId,
						Name: "one",
						Workflow: &zeetv1.CreateProjectCreateProjectWorkflow{
							Id: testWorkflowId,
						},
					},
				},
			})
		} else if strings.Contains(reqs, "mutation updateProject") && strings.Contains(reqs, "two") {
			json.NewEncoder(w).Encode(map[string]any{
				"data": zeetv1.UpdateProjectResponse{
					UpdateProject: zeetv1.UpdateProjectUpdateProject{
						Id:   testProjectId,
						Name: "two",
					},
				},
			})
		} else if strings.Contains(reqs, "query projectDetail") {
			if readCalls == 0 || readCalls == 1 {
				json.NewEncoder(w).Encode(map[string]any{
					"data": zeetv1.ProjectDetailResponse{
						Team: &zeetv1.ProjectDetailTeam{
							Project: &zeetv1.ProjectDetailTeamProject{
								ProjectDetail: zeetv1.ProjectDetail{
									ProjectInfo: zeetv1.ProjectInfo{
										Id:   testProjectId,
										Name: "one",
										Workflow: &zeetv1.ProjectInfoWorkflow{
											Id: testWorkflowId,
										},
									},
									Deploys: zeetv1.ProjectDetailDeploysDeployConnection{
										Nodes: []zeetv1.ProjectDetailDeploysDeployConnectionNodesDeploy{
											{
												DeployConfigurationDetail: zeetv1.DeployConfigurationDetail{
													Id: testDeployId,
												},
											},
										},
									},
								},
							},
						},
					},
				})
				readCalls++
			} else {
				json.NewEncoder(w).Encode(map[string]any{
					"data": zeetv1.ProjectDetailResponse{
						Team: &zeetv1.ProjectDetailTeam{
							Project: &zeetv1.ProjectDetailTeamProject{
								ProjectDetail: zeetv1.ProjectDetail{
									ProjectInfo: zeetv1.ProjectInfo{
										Id:   testProjectId,
										Name: "two",
										Workflow: &zeetv1.ProjectInfoWorkflow{
											Id: testWorkflowId,
										},
									},
									Deploys: zeetv1.ProjectDetailDeploysDeployConnection{
										Nodes: []zeetv1.ProjectDetailDeploysDeployConnectionNodesDeploy{
											{
												DeployConfigurationDetail: zeetv1.DeployConfigurationDetail{
													Id: testDeployId,
												},
											},
										},
									},
								},
							},
						},
					},
				})
			}
		} else if strings.Contains(reqs, "mutation deleteProject") {
			json.NewEncoder(w).Encode(map[string]any{
				"data": zeetv1.DeleteProjectResponse{
					DeleteProject: true,
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
				Config: testAccProjectResourceConfigWithHelmDeployment(server.URL, "one", testClusterId.String()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zeet_project.test_helm", "name", "one"),
					resource.TestCheckResourceAttr("zeet_project.test_helm", "team_id", testTeamId.String()),
					resource.TestCheckResourceAttr("zeet_project.test_helm", "id", testProjectId.String()),
				),
			},
			// Update and Read testing
			{
				Config: testAccProjectResourceConfigWithHelmDeployment(server.URL, "two", testClusterId.String()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zeet_project.test_helm", "name", "two"),
				),
			},
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
				"data": zeetv0.CreateResourceAlphaResponse{
					CreateResourceAlpha: zeetv0.CreateResourceAlphaCreateResourceAlphaRepo{
						Name: "one",
						Project: &zeetv0.CreateResourceAlphaCreateResourceAlphaRepoProject{
							Name: "p",
						},
						ProjectEnvironment: &zeetv0.CreateResourceAlphaCreateResourceAlphaRepoProjectEnvironment{
							Name: "e",
						},
					},
				},
			})
		} else if strings.Contains(reqs, "mutation updateProjectSettings") && strings.Contains(reqs, "two") {
			json.NewEncoder(w).Encode(map[string]any{
				"data": zeetv0.UpdateProjectSettingsResponse{
					UpdateProject: zeetv0.UpdateProjectSettingsUpdateProjectRepo{
						Id: testRepoId,
						RepoDetail: zeetv0.RepoDetail{
							RepoCommon: zeetv0.RepoCommon{
								Name: "two",
							},
						},
					},
				},
			})
		} else if strings.Contains(reqs, "query user") {
			json.NewEncoder(w).Encode(map[string]any{
				"data": zeetv0.UserResponse{
					User: zeetv0.UserUser{
						Id: testTeamId,
						UserDetail: zeetv0.UserDetail{
							UserCommon: zeetv0.UserCommon{
								UserPublicCommon: zeetv0.UserPublicCommon{
									Login: "test",
								},
							},
						},
					},
				},
			})
		} else if strings.Contains(reqs, "query projectV3") {
			json.NewEncoder(w).Encode(map[string]any{
				"data": zeetv0.ProjectV3Response{
					User: zeetv0.ProjectV3User{
						ProjectV3Adapters: &zeetv0.ProjectV3UserProjectV3AdaptersProjectV3AdapterConnection{
							Nodes: []zeetv0.ProjectV3UserProjectV3AdaptersProjectV3AdapterConnectionNodesProjectV3Adapter{
								{
									ProjectV3AdapterDetail: zeetv0.ProjectV3AdapterDetail{
										Id: testProjectId,
									},
								},
							},
						},
					},
				},
			})
		} else if strings.Contains(reqs, "query repoForProjectEnvironment") {
			if !created {
				json.NewEncoder(w).Encode(map[string]any{
					"data": zeetv0.RepoForProjectEnvironmentResponse{
						Project: &zeetv0.RepoForProjectEnvironmentProject{
							Id: testGroupId,
							Environment: zeetv0.RepoForProjectEnvironmentProjectEnvironment{
								Id: testSubGroupId,
								Repo: &zeetv0.RepoForProjectEnvironmentProjectEnvironmentRepo{
									RepoDetail: zeetv0.RepoDetail{
										RepoCommon: zeetv0.RepoCommon{
											Id:   testRepoId,
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
					"data": zeetv0.RepoForProjectEnvironmentResponse{
						Project: &zeetv0.RepoForProjectEnvironmentProject{
							Id: testGroupId,
							Environment: zeetv0.RepoForProjectEnvironmentProjectEnvironment{
								Id: testSubGroupId,
								Repo: &zeetv0.RepoForProjectEnvironmentProjectEnvironmentRepo{
									RepoDetail: zeetv0.RepoDetail{
										RepoCommon: zeetv0.RepoCommon{
											Id:   testRepoId,
											Name: "two",
										},
									},
								},
							},
						},
					},
				})
			}
		} else if strings.Contains(reqs, "mutation deleteProject") {
			json.NewEncoder(w).Encode(map[string]any{
				"data": zeetv0.DeleteProjectResponse{
					DeleteRepo: true,
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
				Config: testAccProjectResourceConfigWithContainerDeployment(server.URL, "one", testClusterId.String()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("zeet_project.test_container", "name", "one"),
					resource.TestCheckResourceAttr("zeet_project.test_container", "team_id", testTeamId.String()),
					resource.TestCheckResourceAttr("zeet_project.test_container", "container.repo_id", testRepoId.String()),
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
