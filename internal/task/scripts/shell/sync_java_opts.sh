#!/usr/bin/env bash

if [ $# -ne 1 ]; then
  echo "Usage: $0 /opt/dingo/bin/start-executor.sh"
  exit 1
fi

BOOTSTRAP_SCRIPT="$1"

if [ ! -f "$BOOTSTRAP_SCRIPT" ]; then
  echo "Error: File '$BOOTSTRAP_SCRIPT' not found."
  exit 2
fi

# Read environment overrides, defaults empty
Xms_OVERRIDE=${xms:-}
Xmx_OVERRIDE=${xmx:-}
SOFTMAXHEAP_OVERRIDE=${softmaxheapsize:-}
MAXDIRECTMEM_OVERRIDE=${maxdirectmemorysize:-}
ALWAYSPRE_TOUCH_KEEP=${alwayspretouch,,}  # lowercase

# Extract original JAVA_OPTS block (multi-line) from file
JAVA_OPTS_BLOCK=$(sed -n '/^JAVA_OPTS="/,/^"/p' "$BOOTSTRAP_SCRIPT")

if [ -z "$JAVA_OPTS_BLOCK" ]; then
  echo "Error: JAVA_OPTS block not found in $BOOTSTRAP_SCRIPT"
  exit 3
fi

# Extract the options content inside JAVA_OPTS=" ... "
JAVA_OPTS_CONTENT=$(echo "$JAVA_OPTS_BLOCK" | sed '1d;$d' | tr -d '\\' | xargs)

# Convert JAVA_OPTS content to an array for manipulation
read -ra OPTS_ARRAY <<< "$JAVA_OPTS_CONTENT"

# Function to update or add option key=value or flag
update_opt() {
  local key=$1
  local newval=$2
  local found=0
  for i in "${!OPTS_ARRAY[@]}"; do
    # Handle both key=value and boolean flags (-XX:+Option)
    if [[ "${OPTS_ARRAY[i]}" == "$key"* ]]; then
      OPTS_ARRAY[i]="$key$newval"
      found=1
      break
    fi
  done
  if [ $found -eq 0 ]; then
    OPTS_ARRAY+=("$key$newval")
  fi
}

# Function to remove option by exact match or prefix
remove_opt_prefix() {
  local prefix=$1
  local tmp=()
  for val in "${OPTS_ARRAY[@]}"; do
    if [[ "$val" != "$prefix"* ]]; then
      tmp+=("$val")
    fi
  done
  OPTS_ARRAY=("${tmp[@]}")
}

# Update Java options based on env vars
if [ -n "$Xms_OVERRIDE" ]; then
  # Replace all starting with -Xms
  for i in "${!OPTS_ARRAY[@]}"; do
    if [[ "${OPTS_ARRAY[i]}" =~ ^-Xms ]]; then
      OPTS_ARRAY[i]="-Xms${Xms_OVERRIDE}"
    fi
  done
fi

if [ -n "$Xmx_OVERRIDE" ]; then
  for i in "${!OPTS_ARRAY[@]}"; do
    if [[ "${OPTS_ARRAY[i]}" =~ ^-Xmx ]]; then
      OPTS_ARRAY[i]="-Xmx${Xmx_OVERRIDE}"
    fi
  done
fi

if [ -n "$SOFTMAXHEAP_OVERRIDE" ]; then
  update_opt "-XX:SoftMaxHeapSize=" "$SOFTMAXHEAP_OVERRIDE"
fi

if [ -n "$MAXDIRECTMEM_OVERRIDE" ]; then
  update_opt "-XX:MaxDirectMemorySize=" "$MAXDIRECTMEM_OVERRIDE"
fi

# Remove -XX:+AlwaysPreTouch if env AlwaysPreTouch is not "true"
if [[ "$ALWAYSPRE_TOUCH_KEEP" != "true" ]]; then
  remove_opt_prefix "-XX:+AlwaysPreTouch"
fi

# Rebuild JAVA_OPTS block string with line continuation and indentation as original
JAVA_OPTS_NEW="JAVA_OPTS=\"\\
"
for opt in "${OPTS_ARRAY[@]}"; do
  JAVA_OPTS_NEW+="$opt "
done
JAVA_OPTS_NEW+=\"

# Update the start-executor.sh file content by replacing old JAVA_OPTS block
# Use awk to do this carefully for multi-line block replacement
awk -v newblock="$JAVA_OPTS_NEW" '
  BEGIN { in_block=0 }
  /^JAVA_OPTS="/ { in_block=1; print newblock; next }
  /^[^"]*"/ && in_block==1 { in_block=0; next }
  { if (in_block==0) print }
' "$BOOTSTRAP_SCRIPT" > "${BOOTSTRAP_SCRIPT}.tmp" && mv "${BOOTSTRAP_SCRIPT}.tmp" "$BOOTSTRAP_SCRIPT"

chmod +x "$BOOTSTRAP_SCRIPT"

echo "Successfully updated $BOOTSTRAP_SCRIPT JAVA_OPTS based on environment variables."
