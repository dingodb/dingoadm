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

echo "Configuring system performance settings for DingoFS on Ubuntu..."

# Set CPU governor to performance
if [ -d /sys/devices/system/cpu/cpu0/cpufreq ]; then
    echo "Setting CPU governor to performance..."
    for cpu in /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor; do
        echo performance | sudo tee "$cpu"
    done
else
    echo "CPU frequency scaling not available. Skipping."
fi

# Configure NVMe I/O Scheduler
echo "Checking and configuring NVMe I/O scheduler..."
for nvme_dev in /sys/block/nvme*n1/queue/scheduler; do
    if [ -f "$nvme_dev" ]; then
        current_scheduler=$(cat "$nvme_dev")
        echo "Current I/O scheduler for $nvme_dev: $current_scheduler"
        if [[ "$current_scheduler" != *"[none]"* ]]; then
            echo "Setting I/O scheduler to 'none' for $nvme_dev..."
            echo none | sudo tee "$nvme_dev" > /dev/null
        else
            echo "I/O scheduler is already set to 'none' for $nvme_dev. Skipping."
        fi
    fi
done

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

# Apply sysctl tuning for hugepages and memory settings
SYSCTL_CONF="/etc/sysctl.d/99-dingofs.conf"

cat << EOF | sudo tee -a "$SYSCTL_CONF" > /dev/null
kernel.sched_min_granularity_ns = 10000000
kernel.sched_wakeup_granularity_ns = 15000000
kernel.numa_balancing = 1
kernel.io_uring_disabled=0
vm.dirty_ratio = 40
vm.dirty_background_ratio = 10
vm.swappiness=10
vm.nr_hugepages=1024
vm.max_map_count=655360
net.ipv4.tcp_window_scaling = 1
net.ipv4.tcp_timestamps = 1
EOF

# Apply sysctl changes
sudo sysctl --system

# Configure Huge Pages (HP) and Transparent Huge Pages (THP)
if grep -q '\[always\]' /sys/kernel/mm/transparent_hugepage/enabled; then
    echo "Huge Pages already set to 'always'. Skipping."
else
    echo "Configuring Huge Pages..."
    echo "always" | sudo tee /sys/kernel/mm/transparent_hugepage/enabled > /dev/null
fi

echo "DingoFS tuning applied successfully on Ubuntu."
