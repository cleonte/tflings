# iteration7: count or for_each?

`count` numbers its instances: `server[0]`, `server[1]`, `server[2]`. Take an item out of
the **middle or the start** of the list and every instance after it moves down one
place, so Terraform changes (or, for some resources, replaces) all of them.

`for_each` names the instances after the items: `server["web"]`, `server["api"]`.
Removing `web` removes only `server["web"]`.

This configuration uses `count`. Change it to `for_each`, so that taking one server out
of `var.servers` plans exactly one change: the removal of that server.
