FROM golang:1.21-alpine AS build

RUN apk add --no-cache git ca-certificates
WORKDIR /tmp/build
COPY go.mod go.sum .
RUN --mount=type=cache,target=/go/pkg/mod/ \
    go mod download -x
COPY . .
RUN go build -v -o cronus cmd/main.go

FROM scratch

WORKDIR /app/
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs
COPY --from=build /tmp/build/cronus /app/cronus
COPY assets /app/assets
EXPOSE 8080

CMD ["/app/cronus"]