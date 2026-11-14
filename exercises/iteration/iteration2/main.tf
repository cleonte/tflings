# I AM NOT DONE

variable "ports" {
  type = map(number)
  default = {
    http  = 80
    https = 443
  }
}

# TODO: one resource for every entry of the map, named by its key and holding its port.
resource "terraform_data" "port" {
  input = 80
}
