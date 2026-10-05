

# letsgetlunch

http://lunch.northbriton.net

`letsgetlunch` is an ultra-minimal public scheduling app for Wednesday lunches.

The whole product idea is a public two-month lunch calendar:

- only Wednesdays are shown
- each Wednesday is either `Available` or `Reserved`
- available dates can be clicked to enter a name/email and suggested place
- reserved dates publicly show who reserved them
- there is no account system, admin flow, cancellation, or clearing yet

This is intentionally small and public. It is closer to taping a lunch signup
sheet to a wall than a full scheduling product.

## Run

```sh
go run ./cmd/lets
```

By default the app listens on `:8080` and stores data in `lets.db`.

Optional environment variables:

- `PORT` changes the listen port
- `ADDR` sets the full listen address and overrides `PORT`
- `DATABASE_URL` sets a PostgreSQL-compatible connection URL and overrides `DATABASE_PATH`
- `DATABASE_PATH` changes the SQLite database path

## Kubernetes

`k8s/deploy.yaml` configures the app to use the `lunch` database on CockroachDB.
The database must already exist; the app creates its tables on startup.

```sh
kubectl apply -f k8s/database-access.yaml
kubectl apply -f k8s/deploy.yaml
```

`k8s/database-access.yaml` allows pods labeled `app: letsgetlunch` in the `lunch`
namespace to reach CockroachDB on TCP port 26257 through its ingress policy.

Existing SQLite reservations are not automatically transferred to CockroachDB.

## Stack

- Go standard library HTTP server
- `html/template`
- GORM
- CockroachDB via PostgreSQL in Kubernetes; SQLite by default locally
