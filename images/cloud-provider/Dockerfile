ARG DOCKER_VERSION=20.10.23

FROM golang:1.20
WORKDIR /go/src
# make deps fetching cacheable
COPY go.mod go.sum ./
RUN go mod download
# build
COPY . .
RUN CGO_ENABLED=0 go build -o ./cloud-provider-kind ./cmd/ccm

# Obtain docker binary
ARG TARGETPLATFORM
ARG DOCKER_VERSION
RUN DOCKER_ARCH=$(case ${TARGETPLATFORM:-linux/amd64} in \
    "linux/amd64")   echo "x86_64"  ;; \
    "linux/arm/v7")  echo "armhf"   ;; \
    "linux/arm64")   echo "aarch64" ;; \
    *)               echo ""        ;; esac) \
  && echo "DOCKER_ARCH=$DOCKER_ARCH" \
  && mkdir -p /opt/docker \
  && set -x; wget -q "https://download.docker.com/linux/static/stable/${DOCKER_ARCH}/docker-${DOCKER_VERSION}.tgz" -qO "/tmp/docker.tgz" \
  && tar --extract --file /tmp/docker.tgz --strip-components 1

# build real cloud-provider-kind image
FROM gcr.io/distroless/static-debian11
COPY --from=0 --chown=root:root /go/src/cloud-provider-kind /bin/cloud-provider-kind
COPY --from=0 --chown=root:root /go/src/docker /usr/bin/docker
ENTRYPOINT ["/bin/cloud-provider-kind"]
