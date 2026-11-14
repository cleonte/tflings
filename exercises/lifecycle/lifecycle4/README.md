# lifecycle4: replace_triggered_by

`replace_triggered_by` replaces a resource whenever another resource (or one of
its attributes) changes, even though nothing in the first one changed.

tflings builds your configuration with configuration version `1`, then plans it
with `2`, and expects the app to be replaced.
