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
	"github.com/samber/lo"

	zeetv0 "github.com/zeet-dev/cli/pkg/sdk/v0"
	zeetv1 "github.com/zeet-dev/cli/pkg/sdk/v1"
)

/*
create (1st apply) = Create + Read
update (2nd apply) = Read + Update
destroy = Delete
*/
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
			data := zeetv1.ProjectDetailResponse{
				Team: &zeetv1.ProjectDetailTeam{
					Project: &zeetv1.ProjectDetailTeamProject{
						ProjectDetail: zeetv1.ProjectDetail{
							ProjectInfo: zeetv1.ProjectInfo{
								Id:     testProjectId,
								Name:   "one",
								Status: zeetv1.ProjectStatusJobRunStarting,
								Workflow: &zeetv1.ProjectInfoWorkflow{
									Id: testWorkflowId,
								},
							},
							Deploys: zeetv1.ProjectDetailDeploysDeployConnection{
								Nodes: []zeetv1.ProjectDetailDeploysDeployConnectionNodesDeploy{
									{
										DeployConfigurationDetail: zeetv1.DeployConfigurationDetail{
											Id: testDeployId,
											Configuration: &zeetv1.DeployConfigurationDetailConfigurationDeploymentConfiguration{
												DefaultWorkflowSteps: []zeetv1.BlueprintDriverWorkflowStepAction{
													zeetv1.BlueprintDriverWorkflowStepActionDriverPlan,
													zeetv1.BlueprintDriverWorkflowStepActionDriverApprove,
													zeetv1.BlueprintDriverWorkflowStepActionDriverApply,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			}
			if readCalls == 0 || readCalls == 1 {
				json.NewEncoder(w).Encode(map[string]any{
					"data": data,
				})
				readCalls++
			} else {
				data.Team.Project.ProjectDetail.ProjectInfo.Name = "two"
				json.NewEncoder(w).Encode(map[string]any{
					"data": data,
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
	  blueprint = {
		source = {
		  helmRepository = {
			repositoryUrl: "https://grafana.github.io/helm-charts",
			chart: "grafana"
		  }
		}
	  },
	  target = {
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
			// create step 1
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
		} else if strings.Contains(reqs, "query projectV3") {
			// create step 2
			json.NewEncoder(w).Encode(map[string]any{
				"data": zeetv0.ProjectV3Response{
					User: zeetv0.ProjectV3User{
						ProjectV3Adapters: &zeetv0.ProjectV3UserProjectV3AdaptersProjectV3AdapterConnection{
							Nodes: []zeetv0.ProjectV3UserProjectV3AdaptersProjectV3AdapterConnectionNodesProjectV3Adapter{
								{
									ProjectV3AdapterDetail: zeetv0.ProjectV3AdapterDetail{
										Id: testProjectId,
										Repo: &zeetv0.ProjectV3AdapterDetailRepo{
											Id: testRepoId.String(),
										},
									},
								},
							},
						},
					},
				},
			})
		} else if strings.Contains(reqs, "mutation updateProjectSettings") && strings.Contains(reqs, "two") {
			// update step 2
			json.NewEncoder(w).Encode(map[string]any{
				"data": zeetv0.UpdateProjectSettingsResponse{
					UpdateProject: zeetv0.UpdateProjectSettingsUpdateProjectRepo{
						Id: testRepoId.String(),
						RepoDetail: zeetv0.RepoDetail{
							RepoCommon: zeetv0.RepoCommon{
								Name: "two",
							},
						},
					},
				},
			})
		} else if strings.Contains(reqs, "query userRepo") {
			data := zeetv0.UserRepoResponse{
				CurrentUser: zeetv0.UserRepoCurrentUser{
					Repo: &zeetv0.UserRepoCurrentUserRepo{
						Id: testRepoId.String(),
						RepoDetail: zeetv0.RepoDetail{
							DeployService: lo.ToPtr(true),
							RepoCommon: zeetv0.RepoCommon{
								Id:   testRepoId.String(),
								Name: "one",
								Source: zeetv0.RepoCommonSourceRepoSource{
									Id:   "https://github.com/zeet-demo/node-express-demo.git",
									Type: zeetv0.RepoSourceTypeGit,
								},
							},
							RepoBuild: zeetv0.RepoBuild{
								BuildMethod: &zeetv0.RepoBuildBuildMethod{
									Type:             zeetv0.BuildTypeNode,
									BuildCommand:     lo.ToPtr("npm --production=false install"),
									RunCommand:       lo.ToPtr("npm start"),
									WorkingDirectory: lo.ToPtr("./"),
									NodejsVersion:    lo.ToPtr("18"),
								},
							},
							DeployTarget: lo.ToPtr(zeetv0.DeployTargetKubernetes),
							Cluster: &zeetv0.RepoDetailCluster{
								Id: testClusterId,
							},
							Project: &zeetv0.RepoDetailProject{
								Id: testGroupId,
							},
							ProjectEnvironment: &zeetv0.RepoDetailProjectEnvironment{
								Id: testSubGroupId,
							},
							RepoNetwork: zeetv0.RepoNetwork{
								Ports: []zeetv0.RepoNetworkPortsPort{
									{
										Port:     "3000",
										Protocol: string(zeetv0.PortProtocolTcp),
										Public:   true,
										Https:    true,
									},
								},
							},
							Cpu:       lo.ToPtr("1"),
							Memory:    lo.ToPtr("1G"),
							Dedicated: lo.ToPtr(false),
						},
					},
				},
			}
			if !created {
				// create step 2
				json.NewEncoder(w).Encode(map[string]any{
					"data": data,
				})
				created = true
			} else {
				// update step 1
				data.CurrentUser.Repo.RepoDetail.RepoCommon.Name = "two"
				json.NewEncoder(w).Encode(map[string]any{
					"data": data,
				})
			}
		} else if strings.Contains(reqs, "mutation deleteProject") {
			json.NewEncoder(w).Encode(map[string]any{
				"data": zeetv0.DeleteProjectResponse{
					DeleteRepo: true,
				},
			})
		} else {
			t.Fatal("unexpected request", reqs[:42])
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
      build = {
        buildType: "NODE",
        buildCommand: "npm --production=false install",
        nodejsVersion: "18",
        runCommand: "npm start",
        workingDirectory: "./"
      }
    })
    kubernetes = jsonencode({
      deployTarget = {
        deployTarget: "KUBERNETES",
        clusterID: %[3]q
      },
      app = {
        deployService: true,
        ports: [
          {
            port: "3000",
            protocol: "tcp",
            public: true,
            https: true
          }
        ],
        resources = {
          cpu: 1,
          memory: 1,
          spot: true
        }
      }
    })
  }

  enabled = true
}
`, server, name, clusterID)
}
