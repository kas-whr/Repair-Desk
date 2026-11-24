FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal

ARG VERSION=dev
# Builds every binary in cmd/: api and prober
RUN CGO_ENABLED=0 go build -trimpath \
    -ldflags="-s -w -X github.com/kas-whr/Repair-Desk/internal/buildinfo.Version=${VERSION}" \
    -o /out/ ./cmd/...

FROM alpine:3.20
# Numeric UID: Kubernetes runAsNonRoot can only verify numeric users
RUN adduser -D -H -u 10001 app
USER 10001
COPY --from=build /out/ /usr/local/bin/
EXPOSE 8080
ENTRYPOINT ["api"]
