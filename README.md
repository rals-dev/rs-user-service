<h3>User Service</h3>

<h3>Description</h3>

<p>This repository will be used to manage user and auth</p>

<h3>Directory Structure</h3>

```
user-service
    L cmd                            → Contains the main entry point or initial configuration of the application
    L common                         → Stores common functions used throughout the application
    L config                         → Contains application configurations such as environment variables and other settings
    L constants                      → Stores global constant values used across the application
    L controllers                    → Manages control logic for handling HTTP requests
    L database                       → Contains files related to database management
        L seeders                    → Scripts for populating initial (seed) data into the database
    L domain                         → The application's domain module containing core domain elements
        L dto                        → Data Transfer Objects, used to define the structure of transferred data
        L models                     → Object models representing the application's or database's data structure
    L middlewares                    → Contains middleware for processing requests/responses before or after reaching the controller
    L repositories                   → Contains data access logic for interacting with the database
    L routes                         → Contains API route definitions
    L services                       → Stores the application's core business logic
```

## How to setup

```
- Clone this repository
- go mod tidy
- copy .env.example to .env (if you want to run with consul)
- copy .config.json.example to .config.json
```

### Configuration notes

- `signatureKey` and `jwtSecretKey` (config.json, or the equivalent Consul keys) are
  required and must be at least 16 characters; the app now fails fast at startup
  if either is missing or too short instead of silently signing/verifying with
  an empty secret.
- `allowedOrigins` (config.json) is the CORS allow-list of browser origins
  permitted to call this API. There is no wildcard fallback: origins not in the
  list receive no CORS headers, so browsers block the response. Configure this
  explicitly per environment.
- Consul-based config uses `CONSUL_HTTP_URL`, `CONSUL_HTTP_PATH`,
  `CONSUL_HTTP_TOKEN` and `CONSUL_WATCH_INTERVAL_SECONDS` from `.env` /
  environment variables (previously `CONSUL_HTTP_PATH` was read under a
  mismatched name and never actually applied).
- `TIMEZONE` (env var, see `.env.example`) controls the process-wide time
  location; it defaults to `Asia/Jakarta` if unset.
- Admin seeding is environment-driven and has no hardcoded default credential.
  Set `ADMIN_USERNAME`, `ADMIN_PASSWORD` (min 8 chars), `ADMIN_EMAIL` and
  `ADMIN_PHONE_NUMBER` to seed a development admin user; leave any of them
  unset to skip seeding entirely (this is the required state in production).
- `GET /api/v1/auth/:uuid` and `PUT /api/v1/auth/:uuid` are only permitted for
  the JWT owner (matching UUID) or a caller with the `admin` role; anyone else
  gets `403 Forbidden`.
- Running via Docker without Consul: since the runtime image no longer bakes
  in `config.json`, mount it explicitly, e.g. add
  `- ./config.json:/app/config.json:ro` under `volumes:` in `docker-compose.yml`.

### Follow-ups requiring an isolated/build environment

The following could not be executed in this session (no build/test/DB access)
and should be run in an isolated environment before merging:
- `go mod tidy` (not run; no new dependencies were added, but this should still
  be run to keep go.mod/go.sum tidy).
- `go build ./...` and `go vet ./...` to confirm everything compiles.
- Running the new unit tests (`go test ./...`).
- Applying the `AutoMigrate`-driven unique constraints (`idx_users_uuid`,
  `idx_users_username`, `idx_users_email`, `idx_roles_code`) against a
  database that may already contain duplicate rows - resolve duplicates first
  or the migration will fail.
- Validating the updated `Jenkinsfile` (particularly the `sshUserPrivateKey`
  credential binding and the still-commented `image:` line in
  `docker-compose.yml`, which the "Update docker-compose.yml" stage assumes is
  present and uncommented) against a real Jenkins environment.

## How to run

```bash
make watch-prepare (only for the first time or when you add new dependency)
make watch
```

## How to run with docker

```bash
docker-compose up -d --build --force-recreate
```

## How to build
```bash
make build
```

