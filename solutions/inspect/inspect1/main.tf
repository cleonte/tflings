resource "terraform_data" "app" {
  input = "app"
}

output "answer" {
  value = "blue-42"
}
