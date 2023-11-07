FROM nvidia/cuda:11.8.0-devel-ubuntu22.04

ARG TARGETARCH
ARG GOFLAGS="'-ldflags=-w -s'"

WORKDIR /go/src/github.com/jmorganca/ollama
RUN apt-get update --fix-missing  && apt-get install -y git build-essential cmake
ADD https://dl.google.com/go/go1.21.3.linux-$TARGETARCH.tar.gz /tmp/go1.21.3.tar.gz
RUN mkdir -p /usr/local && tar xz -C /usr/local </tmp/go1.21.3.tar.gz

COPY . .
ENV GOARCH=$TARGETARCH
ENV GOFLAGS=$GOFLAGS
RUN /usr/local/go/bin/go generate ./... \
    && /usr/local/go/bin/go build -o lollama ./lambda

FROM ubuntu:22.04
RUN apt-get update --fix-missing && apt-get install -y ca-certificates
COPY --from=0 /go/src/github.com/jmorganca/ollama/lollama /bin/lollama
RUN chmod +x /bin/lollama
ENTRYPOINT ["/bin/lollama"]
