# url-shortener

A URL shortener in Go, built end to end as a learning project: code, tooling, hosting, and scaling, all at zero cost.

## Target architecture

![System design for the URL shortener: client → API gateway → load balancer, splitting into a URL service (short code generation + database) and a redirect service (cache first, database on a miss, analytics via a message queue)](system-design-url-shortener.png)

This is the full-scale design the project is working towards. It is being built in stages, starting small.

## Current status: v1

v1 is a single Go server with no external dependencies:

- **Shorten:** `POST /api/shorten` with `{"url": "https://example.com"}` returns `{"code", "short_url"}`. The URL is validated (http/https only, no credentials, no localhost or private IPs, max 2048 characters), hashed with SHA-256, and encoded as a 7-character Base62 code. The same URL always gets the same code.
- **Redirect:** `GET /{code}` responds with a `302` to the original URL, or a generic `404`.
- **Page:** `GET /` serves a minimal HTML form.
- **Storage:** in memory only, so links are lost when the server restarts.

Not built yet: database, cache, reputation checks for malicious URLs, rate limiting, analytics.

## Run locally

```sh
go run .
```

Then open http://localhost:8080.

Optional settings:

| Variable   | Default                   | Purpose                              |
|------------|---------------------------|--------------------------------------|
| `PORT`     | `8080`                    | Port the server listens on           |
| `BASE_URL` | `http://localhost:$PORT`  | Public address used in short links   |

## Test

```sh
go test -race ./...
```
