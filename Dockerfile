FROM golang:1.21-alpine AS build_base

RUN apk add --no-cache git

WORKDIR /tmp/build

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN go build -v -o cronus cmd/main.go


FROM alpine:3.19
RUN apk add ca-certificates

COPY --from=build_base /tmp/build/cronus /app/cronus

EXPOSE 8080

CMD ["/app/cronus"]