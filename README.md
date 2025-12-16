# chromedp-server

HTTP API for headless Chrome web scraping using chromedp.

## Quick Start

**Run the Docker image:**

```bash
# Intel/AMD (x86_64)
docker run -p 4001:4001 ghcr.io/jozefrudy/chromedp-server:latest

# Apple Silicon (M1/M2/M3)
docker run --platform linux/amd64 -p 4001:4001 ghcr.io/jozefrudy/chromedp-server:latest
```

**Or build from source:**

```bash
docker build -t chromedp-server .
docker run -p 4001:4001 chromedp-server
```

## Usage

**Scrape URLs:**

```bash
curl -X POST http://localhost:4001/scrape \
  -H "Content-Type: application/json" \
  -d '{"urls": ["https://example.com"]}'
```

**Response:**

```json
{
  "results": [
    {
      "content": "<html>...</html>",
      "title": "Example Domain"
    }
  ]
}
```

**Health check:**

```bash
curl http://localhost:4001/health
```

## API

### POST /scrape

Scrapes 1-10 URLs and returns HTML content.

**Request:**
```json
{
  "urls": ["https://example.com", "https://example.org"]
}
```

**Response:**
```json
{
  "results": [
    {
      "content": "html content",
      "title": "page title",
      "error": "optional error message"
    }
  ]
}
```

### GET /health

Returns `{"status": "ok"}`.

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `TIMEOUT_MS` | 60000 | Timeout per URL in milliseconds |

**Example:**

```bash
docker run -e TIMEOUT_MS=120000 -p 4001:4001 ghcr.io/jozefrudy/chromedp-server:latest
```

## Features

- Random user agents and viewport sizes
- Simulated scrolling and human behavior
- Per-URL browser instances
- Client disconnection handling via context cancellation
- Built-in delays for challenge completion
