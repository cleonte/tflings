# iteration2: for_each

`for_each` makes one copy of a resource for every element of a map (or set).
Unlike `count`, each copy is named by its key, so removing one entry does not
renumber the others. Inside the block `each.key` and `each.value` are the entry.
