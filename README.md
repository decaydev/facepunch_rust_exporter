# facepunch_rust_exporter

## Getting Started

Prometheus config example:

```yaml
scrape_configs:
  - job_name: rust
    static_configs:
      - targets:
          - '<RUST_SERVER_IP>:<RUST_SERVER_RCON_PORT>'
        labels:
          process: RustDedicated
        metrics_path: /scrape
        relabel_configs:
          - source_labels:
              - __address__
            target_label: __param_target
          - source_labels:
              - __param_target
            target_label: instance
          - target_label: __address__
            replacement: '<EXPORTER_HOST>:<EXPORTER_PORT>'
```

`docker-compose.yml` has been provided to supply a victoriametrics and grafana instance. 

TODO:

- insert example grafana dashboard json
- provide metrics for most of these: https://www.corrosionhour.com/rust-admin-commands/