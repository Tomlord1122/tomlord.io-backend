# AGENT.md — tomlord.io-backend (Go API)

Quick reference for AI coding agents working in this directory. For human-oriented
docs see `README.md` and the root `AGENTS.md`.

## TL;DR

- **Stack:** Go 1.24 · Gin · pgx/v5 · sqlc · Viper · Gorilla WebSocket
- **Auth:** Goth (Google OAuth) → JWT (cookie + bearer)
- **DB:** PostgreSQL (Supabase in prod, Docker locally)
- **Deploy:** Fly.io (256 MB RAM, shared-cpu-1x, always 1 machine)
- **Cache:** in-process TTL (`internal/cache/memory.go`), singleton

## Commands

```bash
make setup      # docker-up + wait-for-db + migrateup (first-time setup)
make run        # go run cmd/api/main.go
make watch      # air live reload
make build      # compile to ./main
make test       # go test ./... -v
make sqlc       # regenerate internal/db_sqlc/ from sqlc/queries/*.sql
make migrateup  # apply all migrations (golang-migrate)
make migratedown
```

After any change always run:

```bash
go build ./...     # smoke test
go vet ./...       # quick lint
```

## Module map

```
cmd/
  api/main.go                 # entrypoint: viper config → server.NewServer → ListenAndServe
  deploy-jwt/                 # one-off helper for Fly secret rotation

internal/
  auth/                       # Goth + user creation; AuthService
  cache/memory.go             # singleton in-process TTL cache (Get/Set/Delete/DeletePrefix)
  config/                     # viper bootstrap (.env in dev, env vars in prod)
  database/                   # pgxpool factory + Health()
  db_sqlc/                    # GENERATED — never edit by hand, run `make sqlc`
  middleware/                 # AuthMiddleware (RequireAuth / RequireSuperUser / Optional…)
  originpolicy/               # CORS allow-list logic
  server/
    server.go                 # Server struct + service wiring (NewServer)
    routes.go                 # all gin route registrations
    preview_handlers.go       # /api/preview handler
    (other *_handlers.go files for blog/page/message/auth/ws)
  services/
    blog.go                   # BlogService — wraps sqlc + cache
    page.go                   # PageService (CMS)
    message.go                # MessageService (comments + thumbs)
    preview.go                # PreviewService (OpenGraph fetcher, SSRF-protected)
  websocket/                  # Hub + per-room broadcast

sqlc/
  migrations/                 # golang-migrate up/down pairs (DO add new ones, never edit existing)
  queries/                    # SQL queries → sqlc generates Go from these
sqlc.yaml                     # sqlc config
```

## Hard rules

1. **Never edit `internal/db_sqlc/`.** It is regenerated. Change the SQL in
   `sqlc/queries/*.sql` and run `make sqlc`.
2. **Always go through a service.** Handlers in `internal/server/` should call
   `s.<service>.<Method>(ctx, ...)` and translate to HTTP responses. No raw
   queries in handlers.
3. **All new HTTP routes** are registered in `internal/server/routes.go`. Group
   under existing `r.Group("/api")` etc. Apply auth middleware explicitly.
4. **Cache invalidation is the caller's job.** When mutating data, call
   `b.cache.Delete(...)` and/or `b.cache.DeletePrefix(...)` like `BlogService`
   does (see lines 140–145, 290–292 of `services/blog.go`).
5. **Schema changes require a new migration pair.** Add
   `sqlc/migrations/NNN_description.up.sql` + `.down.sql`. Never edit a
   migration that has shipped to prod.
6. **Wire new services in `server.go`'s `NewServer`** and add a field to the
   `Server` struct. Don't construct services inside handlers.

## Adding a new endpoint (template)

```go
// 1. Add SQL → sqlc/queries/foo.sql, then `make sqlc`.
// 2. Service method:
//    internal/services/foo.go → func (f *FooService) DoThing(ctx, req) (*Result, error)
// 3. Handler:
//    internal/server/foo_handlers.go → func (s *Server) doThingHandler(c *gin.Context)
// 4. Wire in server.go (NewServer): fooService := services.NewFooService(dbService)
//    + add `fooService *services.FooService` field on Server.
// 5. Register in routes.go:
//    api.POST("/things", s.authMiddleware.RequireAuth(), s.doThingHandler)
```

## Link-preview feature (recent addition)

Endpoint: `GET /api/preview?url=<absolute-url>` → `services.LinkPreview` JSON.

- `services/preview.go` fetches the URL, parses OpenGraph + `<title>` +
  `<link rel="icon">` using `golang.org/x/net/html`.
- Cached in the singleton `MemoryCache` for 1 hour (key = `preview:url:` + sha256).
- **SSRF defenses (do not weaken):**
  - `validatePreviewURL` blocks non-http/https schemes, `localhost`, IP literals
    in private ranges, and resolves hostnames through `net.LookupIP` to catch
    DNS-based bypasses.
  - `http.Client.CheckRedirect` re-runs `validatePreviewURL` on every redirect
    and caps redirects at 5.
  - 10 s request timeout, 2 MB body cap (`io.LimitReader`).
- The frontend calls this endpoint server-side (SvelteKit SSR) so most reads
  hit the cache. CORS already permits the Vercel frontend.

## Cache contract

`internal/cache/memory.go` is a singleton. **Don't instantiate** — call
`cache.GetInstance()`. Keys use a service-prefix convention so
`DeletePrefix("blogs:list:")` can mass-invalidate. Keep using prefixes when you
add new cached values.

| Prefix              | TTL    | Owner          |
| ------------------- | ------ | -------------- |
| `blog:slug:`        | 10 min | BlogService    |
| `blogs:list:`       | 5 min  | BlogService    |
| `page:name:`        | 10 min | PageService    |
| `preview:url:`      | 1 hour | PreviewService |

## Auth quick reference

- `s.authMiddleware.RequireAuth()`        — any logged-in user
- `s.authMiddleware.RequireSuperUser()`   — admin only (writes to blogs / pages)
- `s.authMiddleware.OptionalAuth()`       — populates user if cookie present, else continues
- `s.authMiddleware.RequireSuperUserOrOwner()` — used for message delete

JWT lives in an HTTP-only cookie *and* is also returned via callback redirect
query param so the SPA can store it. The middleware accepts both.

## Database access pattern

```go
queries := s.dbService.GetQueries()         // *db.Queries (sqlc)
row, err := queries.GetBlogBySlug(ctx, slug)
```

`pgxpool` is configured with `MinConns=2`. Long-running ops should not hold a
connection — use the request context.

## Environment variables (see `.env.example`)

```
DATABASE_URL                # full pg URL (overrides BLUEPRINT_DB_*)
JWT_SECRET                  # >= 32 chars
SESSION_SECRET              # for gorilla/sessions
GOOGLE_CLIENT_ID
GOOGLE_CLIENT_SECRET
FRONTEND_URL                # used for OAuth redirect target
ALLOWED_ORIGINS             # comma-separated CORS allow-list
SYNC_TOKEN                  # protects /api/sync-blogs
```

## Don't

- Don't import a service into another service. If services need to share logic,
  extract a helper into a new sub-package.
- Don't add new external HTTP fetchers without SSRF protection. Reuse or extend
  `services.PreviewService.client` with `CheckRedirect`.
- Don't bypass `cache.GetInstance()` — multiple instances would silently
  duplicate state.
- Don't push secrets to git. `.env` is gitignored; production secrets go via
  `fly secrets set`.
- Don't remove the `/api/sync-blogs` endpoint without confirming the migration
  is no longer needed.
