# url-shortener

Turn a long URL into a short link like `http://localhost:8080/XWqS8CG` that redirects back to the original.

Written in Go with only the standard library. This is a learning project: the goal is to build a shortener end to end (code, tooling, hosting and scaling) at zero cost, one stage at a time.

## Quick start

Requires Go 1.22 or newer.

```sh
git clone https://github.com/manankapoor23/url-shortener.git
cd url-shortener
go run .
```

Open http://localhost:8080, paste a URL and click **Shorten**.

Or use the API directly:

```sh
# Create a short link
curl -X POST localhost:8080/api/shorten -d '{"url": "https://go.dev/doc/"}'
# → {"code":"XWqS8CG","short_url":"http://localhost:8080/XWqS8CG"}

# Follow it
curl -i localhost:8080/XWqS8CG
# → HTTP/1.1 302 Found
#   Location: https://go.dev/doc/
```

## How it works

**Creating a link**

1. **Validate** the URL. It is rejected if it:
   - isn't a full `http://` or `https://` URL (this blocks `javascript:` and `data:` links)
   - is longer than 2048 characters
   - contains a username or password (`https://paypal.com@evil.com` actually goes to `evil.com`)
   - points back to this shortener
   - points to `localhost` or a private network address such as `192.168.x.x`

   Whitespace is trimmed and the host is lowercased first, so `HTTPS://Example.com` and `https://example.com` count as the same URL.
2. **Hash** the URL with SHA-256 and turn the result into **7 Base62 characters** (`a-z`, `A-Z`, `0-9`). That gives about 3.5 trillion possible codes. Since the hash is deterministic, the same URL always gets the same code.
3. **Store** the code → URL pair in memory. If the code is already taken by a different URL (very rare), it hashes again with an attempt number, up to 5 tries.

**Following a link**

- A known code responds with **`302 Found`** and the original URL. A temporary redirect means browsers ask the server on every click instead of caching it, so links can be disabled or counted later.
- Any other code gets the same plain **`404`**, so the response never reveals which codes exist.

**Concurrency:** Go handles each request in its own goroutine. The map is guarded by a read/write mutex: many redirects can read at once, while creating a link takes an exclusive lock only around the check-and-store step.

## API

| Method | Path           | Body                     | Response                                               |
|--------|----------------|--------------------------|--------------------------------------------------------|
| `GET`  | `/`            | none                     | HTML page with the shorten form                        |
| `POST` | `/api/shorten` | `{"url": "https://..."}` | `200` `{"code", "short_url"}` · `400` `{"error"}` if invalid · `413` if body > 4 KB |
| `GET`  | `/{code}`      | none                     | `302` to the original URL · `404` if unknown           |

## Configuration

| Variable   | Default                  | Purpose                                                   |
|------------|--------------------------|-----------------------------------------------------------|
| `PORT`     | `8080`                   | Port the server listens on                                |
| `BASE_URL` | `http://localhost:$PORT` | Public address used to build short links (e.g. your domain) |

## Project layout

| File                | What it does                                    |
|---------------------|-------------------------------------------------|
| `main.go`           | Reads config and starts the HTTP server         |
| `handlers.go`       | Routes and HTTP handlers                        |
| `validate.go`       | URL validation rules                            |
| `shortcode.go`      | SHA-256 → Base62 code generation                |
| `store.go`          | In-memory, mutex-guarded code → URL map         |
| `static/index.html` | The web page (embedded into the binary)         |
| `*_test.go`         | Tests for each of the above                     |

## Tests

```sh
go test -race ./...
```

`-race` turns on Go's race detector, which checks the concurrent-access code for data races.

## Limitations (v1)

- **Links disappear on restart.** Everything is held in memory; there is no database yet.
- **No check for malicious sites.** Validation only checks the URL's format, not whether the site is harmful.
- **No rate limiting.** Anyone who can reach the server can create unlimited links.

## Roadmap

The diagram below is the full-scale design this project is working towards. v1 covers only the short-code generation and redirect parts, in a single server.

![Target architecture: client → API gateway → load balancer, splitting into a URL service (short code generation + database) and a redirect service (cache first, database on a miss, analytics via a message queue)](system-design-url-shortener.png)

Next steps, roughly in order:

- [x] v1: shorten + redirect, input validation, in-memory storage
- [ ] Check URLs against a malicious-site list (fail closed if the check is unavailable)
- [ ] Deploy to free hosting
- [ ] Persistent database
- [ ] Cache for redirects
- [ ] Rate limiting
- [ ] Click analytics
- [ ] Better UI
