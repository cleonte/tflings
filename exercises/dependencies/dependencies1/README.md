# dependencies1: explicit dependencies

Terraform works out the order from references: a resource that uses another
one's attribute is created after it. Sometimes there is a dependency that no
attribute shows, such as an app that needs its database to exist first. For
those, the `depends_on` meta-argument lists what to wait for.

Make `terraform_data.app` wait for `terraform_data.database`.
