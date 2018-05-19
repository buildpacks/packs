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
	Image() (v1.Image, error)
	Write(image v1.Image) error
	Source(refs ...name.Reference)
}

func NewRepo(ref name.Reference) (Store, error) {
	auth, err := authn.DefaultKeychain.Resolve(ref.Context().Registry)
	if err != nil {
		return nil, err
	}
	return &repoStore{ref: ref, auth: auth}, nil
}

type repoStore struct {
	ref    name.Reference
	auth   authn.Authenticator
	mounts []name.Repository
}

func (r *repoStore) Image() (v1.Image, error) {
	return remote.Image(r.ref, r.auth, http.DefaultTransport)
}

func (r *repoStore) Write(image v1.Image) error {
	return remote.Write(r.ref, image, r.auth, http.DefaultTransport, remote.WriteOptions{
		MountPaths: r.mounts,
	})
}

func (r *repoStore) Source(refs ...name.Reference) {
	for _, ref := range refs {
		if r.ref.Context().RegistryStr() == ref.Context().RegistryStr() {
			r.mounts = append(r.mounts, ref.Context())
		}
	}
}

func NewDaemon(tag name.Tag) Store {
	return &daemonStore{tag: tag}
}

type daemonStore struct {
	tag name.Tag
}

func (r *daemonStore) Image() (v1.Image, error) {
	return daemon.Image(r.tag, nil)
}

func (r *daemonStore) Write(image v1.Image) error {
	_, err := daemon.Write(r.tag, image, daemon.WriteOptions{})
	return err
}

func (r *daemonStore) Source(refs ...name.Reference) {}
