#!/usr/bin/env bash

cd /dingofs/monitor
exec python3 target_json.py >> /dingofs/monitor/monitor_sync.log 2>&1
