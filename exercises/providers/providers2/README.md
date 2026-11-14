# providers2: provider version constraints

`required_providers` names every provider and where it comes from (`source`),
and `version` limits which releases may be used. `~> 6.0` means "6.0 or newer,
but below 7.0": new features and fixes, no breaking major version.

The `.terraform.lock.hcl` file then records the exact version that `init`
picked, so the whole team gets the same one.

The constraint in this exercise matches no release that exists. Fix it. This
plans against Floci (`tflings aws up`); nothing is created.
