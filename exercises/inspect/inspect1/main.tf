resource "terraform_data" "app" {
  input = "app"
}

# TODO: put the input of the resource that is only in the state here.
output "answer" {
  value = ""
}
