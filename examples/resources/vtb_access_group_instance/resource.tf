data "vtb_user_data" "shemakov" {
    query_string = "vtb70177758"
    domain       = "test.vtb.ru"
}

data "vtb_user_data" "kurushkina" {
    query_string = "vtb70165094"
    domain       = "test.vtb.ru"
}

resource "vtb_access_group_instance" "group_one" {
  name = "example-group-name"
  domain =  "test.vtb.ru"
  description = "example"
  users = [
    data.vtb_user_data.shemakov,
    data.vtb_user_data.kurushkina,
  ]
}