FROM golang:alpine AS builder

WORKDIR /app
COPY . .

ARG GO_TAGS=""
RUN go mod download
# Optimize build by removing debug information and disable cross compilation
RUN CGO_ENABLED=0 GOOS=linux go build -tags="${GO_TAGS}" -ldflags="-w -s" -o /go/bin/fintech ./cmd/server/main.go

FROM alpine:3.21 AS certs
RUN apk add --no-cache ca-certificates

FROM scratch

COPY --from=builder /go/bin/fintech /go/bin/fintech
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
ENV SSL_CERT_FILE=/etc/ssl/certs/ca-certificates.crt

CMD ["/go/bin/fintech"]
