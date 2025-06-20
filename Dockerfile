FROM golang:alpine AS builder

RUN apk update && apk add --no-cache git

WORKDIR /app
COPY . .

RUN go mod download
# Optimize build by removing debug information and disable cross compilation
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /go/bin/fintech ./cmd

FROM scratch

COPY --from=builder /go/bin/fintech /go/bin/fintech

CMD ["/go/bin/fintech"]
