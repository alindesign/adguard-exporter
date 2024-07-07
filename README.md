# AdGuard Home Prometheus Exporter

This is a Prometheus exporter for [AdGuard Home](https://github.com/AdguardTeam/AdGuardHome). <br/>
<small>(forked from [henrywhitaker3/adguard-exporter](https://github.com/henrywhitaker3/adguard-exporter))</small>

Differences:
- Support for clients configuration file
- Support for username and password files

![Dashboard](grafana/dashboard.png)

## Installation

### Using Docker

You can run it using the following example and pass configuration environment variables:

```
$ docker run \
  -e 'APP_INTERVAL=15s' \ # defaults to 30s
  -e 'APP_DEBUG=true' \ # defaults to false
  -e 'CLIENTS_FILE=/data/clients.yaml' \ # defaults to '/etc/adguard-exporter/clients.yaml'
  -p 9618:9618 \
  ghcr.io/alindesign/adguard-exporter:latest
```

### using Docker Compose

```yaml
services:
  adguard-exporter:
    image: ghcr.io/evgeni/adguard-exporter:latest
    environment:
      APP_INTERVAL: 15s
      APP_DEBUG: true
      CLIENTS_FILE: /data/clients.yaml
    ports:
      - 9618:9618
    volumes:
      - ./clients.yaml:/data/clients.yaml

```

### Env Vars

| Variable       | Description                                                    | Required | Default |
|----------------|----------------------------------------------------------------|----------|---------|
| `APP_INTERVAL` | The interval that the exporter scrapes metrics from the server | `True`   |         |
| `APP_DEBUG`    | Turns on the go profiler                                       | `True`   |         |
| `SERVER_PORT`  | Server binding port                                            | `True`   |         |
| `SERVER_HOST`  | Server binding host                                            | `False`  | `30s`   |
| `CLIENTS_FILE` | Clients configuration                                          | `False`  | `false` |

### Clients configuration

The clients configuration is a YAML file that contains a list of clients used for fetching metrics.
The file should look like this:

```yaml
- address: 127.0.0.1:3000  # AdGuard server address
  username: username       # AdGuard username
  password: password       # AdGuard password

- address: 127.0.0.1:4000
  username: username
  password: password
  
- address: 127.0.0.1:5000
  username: /path/to/username/file
  password: /path/to/password/file
```

## Usage

Once the exporter is running, you also have to update your `prometheus.yml` configuration to let it scrape the exporter:

```yaml
scrape_configs:
  - job_name: 'adguard'
    static_configs:
      - targets: ['localhost:9618']
```

If you want to strip the scheme and port out of the `server` label in the metrics, you can add a relabeling:

```yaml
- action: replace
  sourceLabels: ["server"]
  regex: http(|s):\/\/([0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}).*
  replacement: $2
  targetLabel: server
```

## Available Prometheus metrics

| Metric name                                     | Description                                                    |
|-------------------------------------------------|----------------------------------------------------------------|
| adguard_scrape_errors_total                     | The number of errors scraping a target                         |
| adguard_protection_enabled                      | Whether DNS filtering is enabled                               |
| adguard_running                                 | Whether adguard is running or not                              |
| adguard_queries                                 | Total queries processed in the last 24 hours                   |
| adguard_query_types                             | The number of DNS queries by adguard_query_types               |
| adguard_blocked_filtered                        | Total queries that have been blocked from filter lists         |
| adguard_blocked_safesearch                      | Total queries that have been blocked due to safesearch         |
| adguard_blocked_safebrowsing                    | Total queries that have been blocked due to safebrowsing       |
| adguard_avg_processing_time_seconds             | The average query processing time in seconds                   |
| adguard_avg_processing_time_milliseconds_bucket | The processing time of queries                                 |
| adguard_top_queried_domains                     | The number of queries for the top domains                      |
| adguard_top_blocked_domains                     | The number of blocked queries for the top domains              |
| adguard_top_clients                             | The number of queries for the top clients                      |
| adguard_top_upstreams                           | The number of repsonses for the top upstream servers           |
| adguard_top_upstreams_avg_response_time_seconds | The average response time for each of the top upstream servers |
| adguard_dhcp_enabled                            | Whether dhcp is enabled                                        |
| adguard_dhcp_leases                             | The dhcp leases                                                |
