# facepunch_rust_exporter

A prometheus exporter for Facepunch Rust servers


![hazzy](https://user-images.githubusercontent.com/1617698/169644095-868b5548-2702-4dbb-a40d-fbfcc937d327.png)


## Supported Metrics

| name         | description                                         |
|--------------|-----------------------------------------------------|
| facepunch_rust_players | The number of currently connected players |
| facepunch_rust_players_queued | The number of players queued to connect |
| facepunch_rust_players_joining | The number of players connecting |
| facepunch_rust_server_max_players | Max number of players allowed to join |
| facepunch_rust_server_entity_count | Number of entities loaded in game |
| facepunch_rust_server_uptime | How long the server has been up for |
| facepunch_rust_server_framerate | Server framerate |
| facepunch_rust_server_memory | Server memory consumption |
| facepunch_rust_server_collections | Number of collections loaded in game |
| facepunch_rust_server_network_in | Ingress networking traffic |
| facepunch_rust_server_network_out | Egress networking traffic |
| facepunch_rust_server_restarting | 1 if the server is restarting, 0 for running |

## Getting Started

An example grafana-agent configuration file is provided in `examples/`, but generally:

```yaml
scrape_configs:
  - job_name: rust
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
    static_configs:
      - targets:
          - '<RUST_SERVER_HOST>:<RUST_RCON_PORT>'
```

Once configured, build and run:

```sh
go build .
./facepunch_rust_exporter -rcon.password <RCON_PASSWORD>
```

An example grafana dashboard can also be found in `examples/`
