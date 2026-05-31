variable "email" {
  type     = string
  nullable = false
}

data "tidbcloud_member" "example" {
  email = var.email
}

output "output" {
  value = data.tidbcloud_member.example
}
