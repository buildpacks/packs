# Buildpack v3 reference implementation

Execution of these images differs from the standard packs API.

Optionally, you may want to create a Docker volume to share files across runs:

```
$ docker volume create --name packs
```

Detect:

```
$ docker run --rm -v "$(pwd):/launch/app" -v "packs:/workspace" packs/v3:detect
```

Build:

```
$ docker run --rm -v "$(pwd):/launch/app" -v "packs:/workspace" packs/v3:build
```

## Building the images

To build the packs/v3 images yourself, you'll need Stephen's™️ YAML to JSON converter:

```
$ go get sclevine/yj
$ cd $GOPATH/sclevine/yj
$ dep ensure
$ go install
```

Then build the images by running:

```
$ bin/build <stack>
```

Where `<stack>` is any image compatible with the v3 lifecycle. For example, `heroku/heroku:18-build`.