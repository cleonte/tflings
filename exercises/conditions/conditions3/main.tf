# I AM NOT DONE

variable "replicas" {
  type    = number
  default = 3
}

# TODO: an even number of replicas cannot elect a leader. The output must refuse
# to be computed when `var.replicas` is even. Add a precondition to the output.
output "replicas" {
  value = var.replicas
}
