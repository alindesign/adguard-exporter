FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -ldflags="-w -s" -v -o /app/bin/adguard-explorer

FROM alpine:3.20

COPY --from=builder /app/bin/adguard-explorer /app/adguard-explorer

ENV SERVER_PORT 9618

EXPOSE 9618

ENTRYPOINT ["/app/adguard-explorer"]
