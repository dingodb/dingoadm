#!/usr/bin/env bash
# usage: bash extract_hosts.sh /path/to/output_hosts_file

awk '
  /^[[:space:]]*$/ { next }                # ignore blank lines from host
  /^[[:space:]]*#/ { next }                # ignore comments from host
  {
    ip=$1

    # Skip loopback mappings
    if (ip=="127.0.0.1" || ip=="::1") next

    # Skip the common IPv6 reference entries (fe00::0, ff00::0, ff02::..., etc.)
    # by excluding any IPv6 address starting with "f"
    if (tolower(ip) ~ /^f/) next

    print
  }
' /etc/hosts >> "$1"

