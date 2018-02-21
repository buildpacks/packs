# Packs

This repo provides Buildpack Cloud Builders ("Packs") that conform to the Pack Spec.

The Pack Spec is more general than the Buildpack APIs implemented in the Buildpack Lifecycle.
The Pack Spec is consistent for all Packs, regardless of ther Buildpack API version.

External buildpack tools (such as [pack](https://github.com/buildpack/pack) and [aerosol](https://github.com/buildpack/aerosol))
should implement the Pack Spec and not implement individual Buildpack API specs.

## Quick Start: Cloud Foundry Packs

Build:
```bash
docker run -it -v $(pwd)/app:/workspace -v $(pwd)/out:/out packs/cf:build
```

Run:
```bash
docker run  --rm -it -P -v $(pwd)/out:/workspace packs/cflinuxfs2:run
```

Export:
```bash
docker run -it --rm -v $(pwd)/out:/workspace -v /var/run/docker.sock:/var/run/docker.sock  packs/cflinuxfs2:export my-image
```
