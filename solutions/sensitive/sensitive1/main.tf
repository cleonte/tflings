variable "api_key" {
  type      = string
  default   = "abc123-not-a-real-key"
  sensitive = true
}

resource "terraform_data" "client" {
  input = var.api_key
}
