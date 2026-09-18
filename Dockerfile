FROM golang:1.23.4-alpine3.21 AS builder

RUN apk add --no-cache git tzdata build-base

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o /out/user-service .

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S app && adduser -S app -G app

WORKDIR /app

COPY --from=builder /out/user-service /app/user-service

USER app

EXPOSE 8001

ENTRYPOINT [ "/app/user-service" ]
