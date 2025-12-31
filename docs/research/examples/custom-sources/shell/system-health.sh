#!/bin/bash
# System health check source for tinkerdown
# Returns disk, memory, and load status
#
# Usage in markdown:
# ```yaml
# sources:
#   health:
#     type: exec
#     command: "./sources/system-health.sh"
# ```
#
# | check | status | value |
# |-------|--------|-------|
# {{#health}}
# | {{check}} | {{status}} | {{value}} |
# {{/health}}

set -e

# Read input (we don't need it for this source, but follow the contract)
cat > /dev/null

# Collect system metrics
get_disk_usage() {
    df -h / | awk 'NR==2 {print $5}' | tr -d '%'
}

get_memory_usage() {
    if command -v free &> /dev/null; then
        free | awk '/Mem:/ {printf "%.0f", $3/$2 * 100}'
    else
        # macOS fallback
        vm_stat | awk '/Pages active/ {active=$3} /Pages wired/ {wired=$4} /Pages free/ {free=$3} END {printf "%.0f", (active+wired)/(active+wired+free)*100}'
    fi
}

get_load_average() {
    uptime | awk -F'load average:' '{print $2}' | awk -F',' '{print $1}' | tr -d ' '
}

# Determine status based on thresholds
status_for_percent() {
    local value=$1
    local warn=${2:-80}
    local crit=${3:-90}

    if [ "$value" -ge "$crit" ]; then
        echo "critical"
    elif [ "$value" -ge "$warn" ]; then
        echo "warning"
    else
        echo "ok"
    fi
}

# Gather data
DISK=$(get_disk_usage)
MEMORY=$(get_memory_usage)
LOAD=$(get_load_average)

DISK_STATUS=$(status_for_percent "$DISK")
MEMORY_STATUS=$(status_for_percent "$MEMORY")

# Load status (rough: ok < 2, warn < 4, critical >= 4)
LOAD_INT=${LOAD%.*}
if [ "${LOAD_INT:-0}" -ge 4 ]; then
    LOAD_STATUS="critical"
elif [ "${LOAD_INT:-0}" -ge 2 ]; then
    LOAD_STATUS="warning"
else
    LOAD_STATUS="ok"
fi

# Output JSON
cat <<EOF
{
  "columns": ["check", "status", "value"],
  "rows": [
    {"check": "disk", "status": "$DISK_STATUS", "value": "${DISK}%"},
    {"check": "memory", "status": "$MEMORY_STATUS", "value": "${MEMORY}%"},
    {"check": "load", "status": "$LOAD_STATUS", "value": "$LOAD"}
  ]
}
EOF
