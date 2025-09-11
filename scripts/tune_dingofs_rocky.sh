#!/usr/bin/env bash
set -e 

# check time zone is Asia/Shanghai or not, if not, set it to Asia/Shanghai
#if [ "$(timedatectl | grep "Time zone" | awk '{print $3}')" != "Asia/Shanghai" ]; then
#    echo "Setting time zone to Asia/Shanghai..."
#    sudo timedatectl set-timezone Asia/Shanghai
#fi

# check user_allow_other in  /etc/fuse.conf, add config if absent
if ! grep -q "^user_allow_other" /etc/fuse.conf; then
    echo "Adding user_allow_other to /etc/fuse.conf..."
    echo "user_allow_other" | sudo tee -a /etc/fuse.conf > /dev/null
fi

echo "Checking if 'tuned' package is installed..."
if ! rpm -q tuned &>/dev/null; then
    echo "'tuned' is not installed. Installing now..."
    sudo dnf install -y tuned
else
    echo "'tuned' is already installed."
fi

echo "Starting and enabling 'tuned' service..."
sudo systemctl start tuned
sudo systemctl enable tuned

echo "Checking 'tuned' service status..."
if ! sudo systemctl is-active --quiet tuned; then
    echo "Error: 'tuned' service is not running!" >&2
    exit 1
fi

# Create the directory if it doesn't exist
TUNED_DIR="/etc/tuned/dingofs"
echo "Creating tuned profile directory: $TUNED_DIR"
sudo mkdir -p "$TUNED_DIR"

# Use tee to create the tuned profile
echo "Creating tuned profile configuration..."
cat << EOF | sudo tee "$TUNED_DIR/tuned.conf" > /dev/null
[main]
summary=Optimizations for DataCanvas DingoFS
include=throughput-performance

[vm]
transparent_hugepages=always

[cpu]
governor=performance
energy_perf_bias=performance
min_perf_pct=100

[sysctl]
kernel.sched_min_granularity_ns = 10000000
kernel.sched_wakeup_granularity_ns = 15000000
kernel.numa_balancing = 1
kernel.io_uring_disabled = 0
vm.dirty_ratio = 40
vm.dirty_background_ratio = 10
vm.swappiness=10
vm.nr_hugepages=1024
vm.max_map_count=655360
net.ipv4.tcp_window_scaling = 1
net.ipv4.tcp_timestamps = 1

[disk-sas]
type=disk
devices = sd*
elevator = mq-deadline
readahead = 0

[disk-nvme]
type=disk
devices = nvme*
elevator = none
readahead = 0
EOF

echo "Switching to 'dingofs' tuned profile..."
sudo tuned-adm profile dingofs

echo "Verifying active tuned profile..."
sudo tuned-adm active

# Fetch the current user
current_user=$(whoami)

# Configure ulimit for the user
# LIMITS_FILE="/etc/security/limits.d/${current_user}.conf"
# echo "Configuring ulimit settings for user ${current_user} in $LIMITS_FILE"
# cat << EOF | sudo tee "$LIMITS_FILE"
# ${current_user} soft nofile 65536
# ${current_user} hard nofile 65536
# EOF

# configue all user ulimit in /etc/security/limits.conf
echo "Configuring ulimit settings in /etc/security/limits.conf"
cat << EOF | sudo tee -a /etc/security/limits.conf > /dev/null
* soft nofile 65536
* hard nofile 65536
EOF

# Apply ulimit settings immediately
echo "Applying ulimit settings..."
ulimit -n 65536

echo "Tuned script execution completed successfully!"
