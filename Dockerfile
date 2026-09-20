FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata && \
    cp /usr/share/zoneinfo/Africa/Nairobi /etc/localtime && \
    echo "Africa/Nairobi" > /etc/timezone

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/payment-svc ./cmd/payment-svc && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/notification-svc ./cmd/notification-svc && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/fingerprint-svc ./cmd/fingerprint-svc && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/report-svc ./cmd/report-svc

FROM alpine:3.20 AS runtime

RUN apk add --no-cache ca-certificates tzdata wget && \
    cp /usr/share/zoneinfo/Africa/Nairobi /etc/localtime && \
    echo "Africa/Nairobi" > /etc/timezone

WORKDIR /app

COPY --from=builder /out/api /app/api
COPY --from=builder /out/payment-svc /app/payment-svc
COPY --from=builder /out/notification-svc /app/notification-svc
COPY --from=builder /out/fingerprint-svc /app/fingerprint-svc
COPY --from=builder /out/report-svc /app/report-svc

RUN mkdir -p /app/media

EXPOSE 8000 8002 8003 8004 8005

ENTRYPOINT ["/bin/sh"]
