FROM golang:1.25.4-alpine3.22 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY wrapper.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -o scraper wrapper.go

FROM chromedp/headless-shell:143.0.7445.3
RUN apt-get update && apt-get install -y curl && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=builder /app/scraper .
EXPOSE 4001

ENTRYPOINT []
CMD ["./scraper"]
