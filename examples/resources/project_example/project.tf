data "zeet_team" "team" {
  id = "some-uuid"
}

data "zeet_group" "group" {
  team_id = data.zeet_team.team.id
  id      = "some-uuid"
}

resource "zeet_project" "container_project" {
  team_id     = data.zeet_team.team.id
  group_id    = data.zeet_group.group.id
  subgroup_id = "some-uuid"

  name         = "project-name"
  blueprint_id = "some-uuid"

  container = {
    source = {
      git = jsonencode({
        repository : "https://github.com/zeet-demo/node-express-demo.git"
      })
    }
    build = jsonencode({
      build : {
        buildType : "NODE",
        buildCommand : "npm --production=false install",
        nodejsVersion : "18",
        runCommand : "npm start",
        workingDirectory : "./"
      }
    })
    kubernetes = jsonencode({
      deployTarget : {
        deployTarget : "KUBERNETES",
        clusterID : "some-uuid",
      },
      app : {
        deployService : true,
        volumes : [],
        envs : [],
        ports : [
          {
            port : "3000",
            protocol : "tcp",
            public : true,
            https : true
          }
        ],
        resources : {
          cpu : 1,
          memory : 1,
          spot : false
        }
      }
    })
  }

  enabled = true
}

resource "zeet_project" "helm_project" {
  team_id     = data.zeet_team.team.id
  group_id    = data.zeet_group.group.id
  subgroup_id = "some-uuid"

  name         = "project-name"
  blueprint_id = "some-uuid"

  deploys = [{
    default_workflow_steps = ["DRIVER_PLAN", "DRIVER_APPROVE", "DRIVER_APPLY"]
    helm = jsonencode({
      blueprint : {
        source : {
          helmRepository : {
            repositoryUrl : "https://grafana.github.io/helm-charts",
            chart : "grafana"
          }
        }
      },
      target : {
        clusterId : "some-uuid",
        namespace : "grafana",
        releaseName : "grafana"
      }
    })
  }]

  workflow = {
    steps : jsonencode([{ action : "ORCHESTRATION_DEPLOY" }])
  }

  enabled = true
}
