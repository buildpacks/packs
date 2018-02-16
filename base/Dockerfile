FROM cloudfoundry/cflinuxfs2
MAINTAINER Stephen Levine <stephen.levine@gmail.com>

ENV \
  GO_VERSION=1.9.4 \
  DIEGO_VERSION=1.32.0

RUN \
  curl -L "https://storage.googleapis.com/golang/go${GO_VERSION}.linux-amd64.tar.gz" | tar -C /usr/local -xz && \
  git -C /tmp clone --single-branch https://github.com/cloudfoundry/diego-release && \
  cd /tmp/diego-release && \
  git checkout "v${DIEGO_VERSION}" && \
  git submodule update --init --recursive \
    src/code.cloudfoundry.org/archiver \
    src/code.cloudfoundry.org/buildpackapplifecycle \
    src/code.cloudfoundry.org/bytefmt \
    src/code.cloudfoundry.org/cacheddownloader \
    src/code.cloudfoundry.org/goshims \
    src/code.cloudfoundry.org/lager \
    src/code.cloudfoundry.org/systemcerts \
    src/github.com/cloudfoundry-incubator/credhub-cli \
    src/gopkg.in/yaml.v2 && \
  export PATH=/usr/local/go/bin:$PATH && \
  export GOPATH=/tmp/diego-release && \
  CGO_ENABLED=0 go build -a -installsuffix static -o /tmp/lifecycle/builder code.cloudfoundry.org/buildpackapplifecycle/builder && \
  CGO_ENABLED=0 go build -a -installsuffix static -o /tmp/lifecycle/launcher code.cloudfoundry.org/buildpackapplifecycle/launcher && \
  CGO_ENABLED=0 go build -a -installsuffix static -o /tmp/lifecycle/shell code.cloudfoundry.org/buildpackapplifecycle/shell/shell && \
  rm -rf /tmp/diego-release /usr/local/go

USER vcap

ENV \
  CF_INSTANCE_ADDR= \
  CF_INSTANCE_PORT= \
  CF_INSTANCE_PORTS=[] \
  CF_INSTANCE_IP=0.0.0.0 \
  CF_STACK=cflinuxfs2 \
  HOME=/home/vcap \
  PATH=/usr/local/bin:/usr/bin:/bin \
  LANG=en_US.UTF-8 \
  MEMORY_LIMIT=1024m \
  VCAP_SERVICES={}

ENV VCAP_APPLICATION '{ \
    "limits": {"fds": 16384, "mem": 1024, "disk": 1024}, \
    "application_name": "local", "name": "local", "space_name": "local-space", \
    "application_uris": ["localhost"], "uris": ["localhost"], \
    "application_id": "01d31c12-d066-495e-aca2-8d3403165360", \
    "application_version": "2b860df9-a0a1-474c-b02f-5985f53ea0bb", \
    "version": "2b860df9-a0a1-474c-b02f-5985f53ea0bb", \
    "space_id": "18300c1c-1aa4-4ae7-81e6-ae59c6cdbaf1" \
  }'

RUN mkdir /tmp/app /tmp/cache

ENTRYPOINT /tmp/lifecycle/builder

