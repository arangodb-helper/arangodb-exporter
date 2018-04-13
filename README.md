# ArangoDB Exporter for Prometheus

This exporter exposes the statistics provided by a specific ArangoDB instance
in a format compatible with prometheus.

## Usage

To use the ArangoDB Exporter, run the following:

```bash
arangodb_exporter \
    --arangodb.endpoint=http://<your-database-host>:8529 \
    --arangodb.jwtsecret=<your-jwt-secret>
```

This results in an ArangoDB Exporter exposing all statistics of
the ArangoDB server (running at `http://<your-database-host>:8529`)
at `http://<your-host-ip>:9101/metrics`.
