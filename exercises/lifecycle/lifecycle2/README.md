# lifecycle2: prevent_destroy

`prevent_destroy = true` makes Terraform refuse any plan that would destroy the
resource, including a replacement. It is a safety net for things such as
databases: whoever wants to remove it has to remove the setting first.

tflings builds your configuration, then plans it with a change that would
replace the database, and expects Terraform to refuse.
