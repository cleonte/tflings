# types2: object, or a map of one type

Terraform has two ways to group values with names:

| Type | Every value has | Example |
|---|---|---|
| `map(string)` | the **same** type | `{ a = "x", b = "y" }` |
| `object({ name = string, ports = list(number) })` | its **own** type, per attribute | `{ name = "web", ports = [80, 443] }` |

A `list` or `set` is the same idea without names: one element type (`list(number)`),
while a `tuple([string, number])` gives each position its own type.

The variable of this exercise holds one string and one list, so `map(string)`
is wrong and `terraform plan` fails. Change the **type** (not the value) so that
the default is accepted and the outputs work: `port_count` is 2 and `label` is
`web:80`.

Docs: https://developer.hashicorp.com/terraform/language/expressions/type-constraints
