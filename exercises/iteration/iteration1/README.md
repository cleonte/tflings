# iteration1: count

`count` makes several copies of a resource. Inside the block, `count.index`
says which copy you are (0, 1, 2, ...). Make one `terraform_data.server` for
each name in the list, holding that name. tflings plans again with a different
list, so the number must follow the list.
