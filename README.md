# ArangoDB Exporter for Prometheus

This exporter exposes the statistics provided by a specific ArangoDB instance
in a format compatible with prometheus.

## Usage

To use the ArangoDB Exporter, run the following:

```bash
arangodb-exporter \
    --arangodb.endpoint=http://<your-database-host>:8529 \
    --arangodb.jwtsecret=<your-jwt-secret> \
    --ssl.keyfile=<your-optional-ssl-keyfile>
```

This results in an ArangoDB Exporter exposing all statistics of
the ArangoDB server (running at `http://<your-database-host>:8529`)
at `http://<your-host-ip>:9101/metrics`.

## Configuring Prometheus

There are several ways to configure Prometheus to fetch metrics from the ArangoDB Exporter.

Below you're find a sample Prometheus configuration file that can be used to fetch
metrics from an ArangoDB exporter listening on localhost port 9101 (without TLS).

```yaml
global:
  scrape_interval:     15s
scrape_configs:
- job_name: arangodb
  static_configs:
  - targets: ['localhost:9101']
```

For more info on configuring Prometheus go to [its configuration documentation](https://prometheus.io/docs/prometheus/latest/configuration/configuration).

## Building

To build this project, you need Go 1.8 or higher and Docker installed.
Then run:

```bash
DOCKERNAMESPACE=<your docker hub account name> make
```