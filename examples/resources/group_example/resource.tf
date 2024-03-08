data "zeet_team" "team" {
  id = "some-uuid"
}

resource "zeet_group" "group" {
  team_id = data.zeet_team.team.id
  name    = "some-value"
}
