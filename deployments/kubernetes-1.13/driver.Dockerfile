FROM golang:1.12.5-stretch AS build

# Install build dependencies
RUN apt-get update && apt-get install -y --no-install-recommends make

# build the driver
ADD . /go/src/github.com/intel/csi-intel-rsd
WORKDIR /go/src/github.com/intel/csi-intel-rsd
RUN make
RUN pwd && cp csirsd /

# build clean container
FROM ubuntu:18.04
RUN apt-get update && apt-get install -y --no-install-recommends util-linux e2fsprogs dosfstools xfsprogs jfsutils nvme-cli && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

# move required binaries from the build container
COPY --from=build /csirsd /usr/bin/

ENTRYPOINT ["/usr/bin/csirsd"]
