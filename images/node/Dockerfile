# TODO(bentheelder): place these in a registry
ARG BASE_IMAGE="kind-base"
FROM ${BASE_IMAGE}

# copy all artifacts provided by the build tooling into the image
COPY ./files/ /kind/

RUN du -h /kind

# install all the debs if there are any
RUN [ ! -d /kind/debs/ ] || \
    dpkg -i /kind/debs/*.deb && \
    rm -rf \
        /kind/debs/*.deb \
        /var/cache/debconf/* \
        /var/lib/apt/lists/* \
        /var/log/*

RUN du -h /kind
