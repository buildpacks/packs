package img_test

import (
	"errors"
	"math/rand"
	"reflect"
	"strings"
	"testing"

	"github.com/buildpack/packs"
	"github.com/buildpack/packs/img"
	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/sclevine/spec"
)

func Test(t *testing.T) {
	spec.Run(t, "Actions", func(t *testing.T, when spec.G, it spec.S) {
		var (
			baseImage  v1.Image
			baseTag    string
			baseStore  img.Store
			imageTag   string
			imageStore img.Store
		)

		it.Before(func() {
			baseTag = randStringRunes(16)
			_, err := packs.Run("docker", "build", "-t", baseTag, "-f", "testdata/base.Dockerfile", "testdata")
			assertNil(t, err)
			baseStore, err = img.NewDaemon(baseTag)
			assertNil(t, err)
			baseImage, err = baseStore.Image()
			assertNil(t, err)

			imageTag = randStringRunes(16)
			imageStore, err = img.NewDaemon(imageTag)
			assertNil(t, err)
		})

		it.After(func() {
			_, err := packs.Run("docker", "rmi", "-f", baseTag, imageTag)
			assertNil(t, err)
		})

		when(".Append", func() {
			it("appends a tarball on top of a base image", func() {
				image, err := img.Append(baseImage, "testdata/some-appended-layer.tgz")
				assertNil(t, err)
				err = imageStore.Write(image)
				assertNil(t, err)
				output, err := packs.Run("docker", "run", "--rm", imageTag, "ls", "/layers/")
				layers := strings.Fields(output)
				if !reflect.DeepEqual(layers, []string{
					"some-appended-layer.txt",
					"some-layer-1.txt",
					"some-layer-2.txt",
					"some-layer-3.txt",
				}) {
					t.Fatalf(`Unexpected file contents "%s"`, layers)
				}
			})

			when("layer doesn't exist", func() {
				it("errors", func() {
					_, err := img.Append(baseImage, "testdata/some-missing-layer.tgz")
					if err == nil {
						t.Fatal("Expected an error")
					} else if !strings.Contains(err.Error(), "get layer from file:") {
						t.Fatalf(`Expected error to contain: "get layer from file:": got "%s"`, err)
					}
				})
			})

			when("append fails", func() {
				it.Before(func() {
					_, err := packs.Run("docker", "rmi", "-f", baseTag)
					assertNil(t, err)
				})
				it("errors", func() {
					_, err := img.Append(baseImage, "testdata/some-appended-layer.tgz")
					if err == nil {
						t.Fatal("Expected an error")
					} else if !strings.Contains(err.Error(), "append layer:") {
						t.Fatalf(`Expected error to container: "append layer": got "%s"`, err)
					}
				})
			})
		}, spec.Parallel())

		when(".Rebase", func() {
			var newBaseImage v1.Image
			var newBaseTag string
			var upperImage v1.Image
			var upperTag string

			it.Before(func() {
				newBaseTag = randStringRunes(16)
				_, err := packs.Run("docker", "build", "-t", newBaseTag, "-f", "testdata/newbase.Dockerfile", "testdata")
				assertNil(t, err)
				newBaseStore, err := img.NewDaemon(newBaseTag)
				assertNil(t, err)
				newBaseImage, err = newBaseStore.Image()
				assertNil(t, err)

				upperTag = randStringRunes(16)
				_, err = packs.Run("docker", "build", "--build-arg=base="+baseTag, "-t", upperTag, "-f", "testdata/upper.Dockerfile", "testdata")
				assertNil(t, err)
				upperStore, err := img.NewDaemon(upperTag)
				assertNil(t, err)
				upperImage, err = upperStore.Image()
				assertNil(t, err)
			})

			it("rebases image using labels", func() {
				image, err := img.Rebase(upperImage, newBaseImage, func(labels map[string]string) (v1.Image, error) {
					return baseImage, nil
				})
				assertNil(t, err)

				if err := imageStore.Write(image); err != nil {
					t.Fatal(err)
				}
				output, err := packs.Run("docker", "run", "--rm", imageTag, "ls", "/layers/")
				layers := strings.Fields(output)
				if !reflect.DeepEqual(layers, []string{
					"some-new-base-layer.txt",
					"upper-layer-1.txt",
					"upper-layer-2.txt",
				}) {
					t.Fatalf(`Unexpected file contents "%s"`, layers)
				}
			})

			when("old base finder func errors", func() {
				it("errors", func() {
					_, err := img.Rebase(upperImage, newBaseImage, func(labels map[string]string) (v1.Image, error) {
						return nil, errors.New("old base finder error")
					})
					if err == nil {
						t.Fatal("Expected an error")
					} else if err.Error() != "find old base: old base finder error" {
						t.Fatalf(`Expected error to eqal: "find old base: old base finder error": got "%s"`, err)
					}
				})
			})

			when("rebase fails", func() {
				it("errors", func() {
					_, err := img.Rebase(upperImage, newBaseImage, func(labels map[string]string) (v1.Image, error) {
						return newBaseImage, nil
					})
					if err == nil {
						t.Fatal("Expected an error")
					} else if !strings.HasPrefix(err.Error(), "rebase image:") {
						t.Fatalf(`Expected error to have prefix: "rebase image:": got "%s"`, err)
					}
				})
			})
		})

		when(".Label", func() {
			it("adds labels to image", func() {
				image, err := img.Label(baseImage, "label1", "val1")
				assertNil(t, err)
				image, err = img.Label(image, "label2", "val2")
				assertNil(t, err)

				if err := imageStore.Write(image); err != nil {
					t.Fatal(err)
				}

				output, err := packs.Run("docker", "inspect", "--format={{.Config.Labels.label1}}", imageTag)
				assertNil(t, err)
				if output != "val1" {
					t.Errorf(`expected label1 to equal "val1", got "%s"`, output)
				}

				output, err = packs.Run("docker", "inspect", "--format={{.Config.Labels.label2}}", imageTag)
				assertNil(t, err)
				if output != "val2" {
					t.Errorf(`expected label2 to equal "val2", got "%s"`, output)
				}
			})
		})
	}, spec.Parallel())
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz")

func randStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func assertNil(t *testing.T, err error) {
	if err != nil {
		t.Fatal(err)
	}
}
