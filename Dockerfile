# dynamic config
ARG             BUILD_DATE
ARG             VCS_REF
ARG             VERSION

# build
FROM            golang:1.17.7-alpine as builder
RUN             apk add --no-cache git gcc musl-dev make
ENV             GO111MODULE=on
WORKDIR         /go/src/moul.io/bot
COPY            go.* ./
RUN             go mod download
COPY            . ./
RUN             make install

# minimalist runtime
FROM alpine:3.16.3
LABEL           org.label-schema.build-date=$BUILD_DATE \
                org.label-schema.name="bot" \
                org.label-schema.description="" \
                org.label-schema.url="https://moul.io/bot/" \
                org.label-schema.vcs-ref=$VCS_REF \
                org.label-schema.vcs-url="https://github.com/moul/bot" \
                org.label-schema.vendor="Manfred Touron" \
                org.label-schema.version=$VERSION \
                org.label-schema.schema-version="1.0" \
                org.label-schema.cmd="docker run -i -t --rm moul/bot" \
                org.label-schema.help="docker exec -it $CONTAINER bot --help"
COPY            --from=builder /go/bin/moul-bot /bin/
ENTRYPOINT      ["/bin/moul-bot"]
#CMD             []
