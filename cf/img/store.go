package img

import (
	"net/http"

	"github.com/google/go-containerregistry/authn"
	"github.com/google/go-containerregistry/name"
	"github.com/google/go-containerregistry/v1"
	"github.com/google/go-containerregistry/v1/daemon"
	"github.com/google/go-containerregistry/v1/remote"
)

type Store interface {
	Ref() name.Reference
	Digest() (v1.Hash, error)
	Image() (v1.Image, error)
	Write(image v1.Image) error
	Source(refs ...name.Repository)
}

func NewRegistry(ref string) (Store, error) {
	r, err := name.ParseReference(ref, name.WeakValidation)
	if err != nil {
		return nil, err
	}
	auth, err := authn.DefaultKeychain.Resolve(r.Context().Registry)
	if err != nil {
		return nil, err
	}
	return &registryStore{ref: r, auth: auth}, nil
}

type registryStore struct {
	ref    name.Reference
	auth   authn.Authenticator
	mounts []name.Repository
	cache  v1.Image
}

func (r *registryStore) Ref() name.Reference {
	return r.ref
}

func (r *registryStore) Digest() (v1.Hash, error) {
	image, err := r.Image()
	if err != nil {
		return v1.Hash{}, err
	}
	return image.Digest()
}

func (r *registryStore) Image() (v1.Image, error) {
	if r.cache != nil {
		return r.cache, nil
	}
	image, err := remote.Image(r.ref, r.auth, http.DefaultTransport)
	if err != nil {
		return nil, err
	}
	r.cache = image
	return image, nil
}

func (r *registryStore) Write(image v1.Image) error {
	if err := remote.Write(r.ref, image, r.auth, http.DefaultTransport, remote.WriteOptions{
		MountPaths: r.mounts,
	}); err != nil {
		return err
	}
	r.cache = image
	return nil
}

func (r *registryStore) Source(repos ...name.Repository) {
	for _, repo := range repos {
		if r.ref.Context().RegistryStr() == repo.RegistryStr() {
			r.mounts = append(r.mounts, repo)
		}
	}
}

func NewDaemon(tag string) (Store, error) {
	t, err := name.NewTag(tag, name.WeakValidation)
	if err != nil {
		return nil, err
	}
	return &daemonStore{tag: t}, nil
}

type daemonStore struct {
	tag name.Tag
}

func (d *daemonStore) Ref() name.Reference {
	return d.tag
}

func (d *daemonStore) Digest() (v1.Hash, error) {
	image, err := d.Image()
	if err != nil {
		return v1.Hash{}, err
	}
	return image.Digest()
}

func (d *daemonStore) Image() (v1.Image, error) {
	return daemon.Image(d.tag, nil)
}

func (d *daemonStore) Write(image v1.Image) error {
	_, err := daemon.Write(d.tag, image, daemon.WriteOptions{})
	return err
}

func (d *daemonStore) Source(refs ...name.Repository) {}
