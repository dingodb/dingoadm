#!/usr/bin/env bash
# usage: sync_prometheus.sh <prometheus_config_path> <node_exporter_addrs> 
PROMETHEUS_CONFIG_PATH=$1
NODE_EXPORTER_ADDRS=$2

cat <<EOF >> ${PROMETHEUS_CONFIG_PATH}

  - job_name: 'node'
    # Override the global default and scrape targets from this job every 5 seconds.
    scrape_interval: 5s
    static_configs:
      - targets: ${NODE_EXPORTER_ADDRS}
        labels:
          group: 'server'
EOF