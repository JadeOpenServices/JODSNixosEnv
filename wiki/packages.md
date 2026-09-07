
# Packages

GjallarOS keeps repository-owned package definitions separate from the modules
that configure those packages.

Main paths:

    pkgs/
    lib/

## pkgs

    pkgs/

This contains software that GjallarOS packages, wraps, patches or otherwise
provides itself.

A package definition answers:

    "How is this software built or exposed?"

It should not contain unrelated desktop or system policy.

## lib

    lib/

Shared Nix helpers and reusable repository logic belong here when they are not
specific to one module.

## Packages versus modules

The normal split is:

    pkgs/
        build or provide software

    system/
        configure machine-wide software

    user/
        configure user-facing software

For example, a custom executable may be built in `pkgs/` and then exposed or
configured by a module under `system/` or `user/`.

## Overlays

Package overlays are used when GjallarOS needs to adjust package definitions or
package metadata.

They should stay in the package layer instead of being embedded into unrelated
application modules.
