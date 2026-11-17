# I AM NOT DONE

variable "name" {
  type    = string
  default = "tflings"
}

# TODO: terraform init cannot install this module. Read the error: the folder it
# names is not in this exercise. Fix the source.
module "greeting" {
  source = "./modules/greetings"
  name   = var.name
}

output "greeting" {
  value = module.greeting.message
}
