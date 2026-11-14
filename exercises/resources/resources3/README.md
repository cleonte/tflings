# resources3: tags

`tags` is a map of text. Companies tag resources to say who owns them and what
they are for. Give the VPC the tags `Environment = "dev"` and
`Owner = "tflings"`, and keep its `Name` tag.

This creates things in your fake AWS account; tflings reads the tags back from
Floci to check them.
