# Packs

This repo provides Buildpack Cloud Builders ("Packs") that conform to the Pack Spec.

The Pack Spec is more general than the Buildpack APIs implemented in the Buildpack Lifecycle.
The Pack Spec is consistent for all Packs, regardless of ther Buildpack API version.

External buildpack tools (such as [pack](https://github.com/buildpack/pack) and [aerosol](https://github.com/buildpack/aerosol))
should implement the Pack Spec and not implement individual buildpack API specs.
