# Packs

This repo provides buildpack cloud builders ("Packs") that conform to the Pack Spec.

The Pack Spec is more general than the buildpack APIs implemented in the buildpack Lifecycle.
The Pack Spec is consistent for all Packs, regardless of their buildpack API version.

External buildpack tools should implement the Pack Spec. They should not implement individual buildpack APIs.

## Quick Start: Cloud Foundry Packs

Build:
```bash
docker run --rm -v "$(pwd)/app:/workspace" -v "$(pwd)/out:/out" packs/cf:build
```

Run:
```bash
docker run --rm -P -v "$(pwd)/out:/workspace" packs/cf:run
```

Export:
```bash
docker run --rm -v "$(pwd)/out:/workspace" -v /var/run/docker.sock:/var/run/docker.sock packs/cf:export my-image
```
