# Step 1: Modules caching
FROM golang:1.24.2-alpine3.21 AS modules

COPY go.mod go.sum /modules/

WORKDIR /modules

RUN go mod download

# Step 2: Builder
FROM golang:1.24.2-alpine3.21 AS builder

ARG TARGETARCH

RUN apk add --no-cache ca-certificates make tzdata

COPY --from=modules /go/pkg /go/pkg
COPY . /app

WORKDIR /app

RUN make generate
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /bin/app ./cmd/app

# Step 3: Final
FROM scratch

COPY --from=builder /app/config /config
COPY --from=builder /app/template /template
COPY --from=builder /bin/app /app
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

CMD ["/app"]
