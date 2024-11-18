
FROM golang:1.23.3-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /server ./cmd/server

FROM scratch

COPY --from=builder /server /server

ENV ADDR='0.0.0.0:2939'

ENTRYPOINT ["/server", "-addr", "$ADDR"]