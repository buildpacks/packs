# Buildpack v3 reference implementation

Execution of these images differs from the standard packs API.

## Usage

Create your launch dir:

```
$ mkdir launch
$ cp -R /path/to/your/app launch/app
```

Create a volume for the cache from step to step:

```
$ docker volume create --name packs_cache
```

Create a volume to pass TOML files from step to step:

```
$ docker volume create --name packs_ws
```

Detect:

```
$ docker run --rm -v "$(pwd)/launch:/launch" -v "packs_ws:/workspace" packs/v3:detect
```

Build:

```
$ docker run --rm -v "$(pwd)/launch:/launch" -v "packs_cache:/cache" -v "packs_ws:/workspace" packs/v3:build
```

Run:

```
$ docker run --rm -P -v "$(pwd)/launch:/launch" -v "packs_ws:/workspace" \
  -e PACK_STACK_NAME="heroku/heroku" -e PACK_STACK_TAG="18" \
  packs/v3:run
```

Export:

```
$ docker run --rm -v "$(pwd)/launch:/launch" packs/v3:export
```

## Building the images

To build the packs/v3 images yourself, you'll need Stephen's™️ YAML to JSON converter:

```
$ go get github.com/sclevine/yj
$ cd $GOPATH/sclevine/yj
$ dep ensure
$ go install
```

Then build the images by running:

```
$ bin/build <stack>
```

Where `<stack>` is any image compatible with the v3 lifecycle. For example, `heroku/heroku:18-build`.