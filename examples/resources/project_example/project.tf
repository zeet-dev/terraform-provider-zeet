data "zeet_team" "team" {
  id = "some-uuid"
}

data "zeet_group" "group" {
  team_id = data.zeet_team.team.id
  id      = "some-uuid"
}

data "zeet_project" "project" {
  team_id     = data.zeet_team.team.id
  group_id    = data.zeet_group.group.id
  subgroup_id = "some-uuid"

  name         = "project-name"
  blueprint_id = "some-uuid"
}
