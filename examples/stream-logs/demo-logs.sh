#!/bin/sh
# Writes a mix of JSON, logfmt and plain-text log lines, some to stderr,
# forever. Used by examples/stream-logs to show a stream source.
i=0
while true; do
  i=$((i + 1))
  echo "{\"level\":\"info\",\"msg\":\"GET /api/items/$i 200\",\"ms\":$((i % 50))}"
  if [ $((i % 5)) -eq 0 ]; then
    echo "level=warn msg=\"slow query\" table=orders ms=$((800 + i % 200))" 1>&2
  fi
  if [ $((i % 13)) -eq 0 ]; then
    echo "panic: item $i not found" 1>&2
  fi
  sleep 0.5
done
