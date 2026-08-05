FROM golang:1.25.8-alpine AS build

ARG VERSION=dev

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal
COPY core ./core
COPY runtime ./runtime
COPY limits ./limits
COPY observe ./observe
COPY facts ./facts
COPY httpadapter ./httpadapter
COPY testkit ./testkit

RUN go build -ldflags "-X main.version=${VERSION}" -o /out/cervorules-policygen ./cmd/cervorules-policygen
RUN go build -ldflags "-X main.version=${VERSION}" -o /out/cervorules-vocabgen ./cmd/cervorules-vocabgen

FROM alpine:3.20

ARG VERSION=dev

LABEL org.opencontainers.image.title="cervorules-tools"
LABEL org.opencontainers.image.description="CervoRules v3 native policy and vocabulary generators plus schemas"
LABEL org.opencontainers.image.version="${VERSION}"

COPY --from=build /out/cervorules-policygen /usr/local/bin/cervorules-policygen
COPY --from=build /out/cervorules-vocabgen /usr/local/bin/cervorules-vocabgen
COPY schemas /opt/cervorules/schemas
COPY README.md CHANGELOG.md /opt/cervorules/

RUN addgroup -S cervorules && adduser -S -G cervorules cervorules
USER cervorules

ENTRYPOINT ["/bin/sh", "-lc"]
