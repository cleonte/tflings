# dependencies2: implicit dependencies

Terraform builds a graph from your configuration. When one resource uses an
attribute of another (`terraform_data.database.input`), the second must exist
first, and Terraform sees that by itself: an **implicit dependency**.

`depends_on` is the explicit form, for a dependency that no value shows. Prefer
the implicit form when you can: it cannot go out of date.
