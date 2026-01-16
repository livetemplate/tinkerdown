# Custom Sources: Authoring & Distribution

## Vision

Operators author **policy-encoded sources** that:
- Encode organizational rules, compliance checks, approval workflows
- Are easy for LLMs to generate
- Work in any language (Python, Go, shell, Rust, etc.)
- Can be distributed internally (git, registry, package manager)

## The Interface Contract

A custom source is any executable that:

```
INPUT:  JSON on stdin (query context)
OUTPUT: JSON on stdout (data rows)
EXIT:   0 = success, non-zero = error (message on stderr)
```

That's it. Unix philosophy. Language-agnostic.

### Input Format

```json
{
  "query": {},           // Query parameters from markdown
  "env": {               // Execution context
    "incident_id": "2024-01-15-api-outage",
    "operator": "alice",
    "timestamp": "2024-01-15T14:35:00Z"
  }
}
```

### Output Format

```json
{
  "columns": ["name", "status", "value"],
  "rows": [
    {"name": "disk", "status": "ok", "value": "45%"},
    {"name": "memory", "status": "warn", "value": "82%"}
  ],
  "meta": {              // Optional metadata
    "cached_at": "...",
    "ttl": 60
  }
}
```

### Error Handling

```bash
# Exit non-zero, write error to stderr
echo "Permission denied: requires admin role" >&2
exit 1
```

---

## Example: Permission Check (Shell)

```bash
#!/bin/bash
# sources/check-permission.sh
# Checks if operator has required permission

# Read input
INPUT=$(cat)
OPERATOR=$(echo "$INPUT" | jq -r '.env.operator')
PERMISSION=$(echo "$INPUT" | jq -r '.query.permission')

# Check against internal API (or local file, LDAP, etc.)
RESULT=$(curl -s "https://internal-api/permissions?user=$OPERATOR&perm=$PERMISSION")

if [ "$RESULT" = "granted" ]; then
  echo '{"columns":["allowed"],"rows":[{"allowed":true}]}'
else
  echo '{"columns":["allowed"],"rows":[{"allowed":false}]}'
fi
```

Usage in markdown:
```markdown
```yaml
sources:
  perm:
    type: exec
    command: "./sources/check-permission.sh"
    query:
      permission: "prod-db-write"
```​

<div lvt-if="perm[0].allowed">
  ✅ You have prod-db-write access
</div>
```

---

## Example: Approval Status (Python)

```python
#!/usr/bin/env python3
"""sources/approval-status.py - Check approval status in incident log"""
import sys
import json
import re

def main():
    input_data = json.load(sys.stdin)
    incident_file = input_data.get("query", {}).get("incident_file")

    if not incident_file:
        print("Missing incident_file parameter", file=sys.stderr)
        sys.exit(1)

    approvals = []
    with open(incident_file) as f:
        for line in f:
            # Parse approval log entries
            if "access_approved" in line:
                parts = line.split("|")
                if len(parts) >= 6:
                    approvals.append({
                        "time": parts[1].strip(),
                        "access": parts[4].strip(),
                        "approver": parts[5].strip(),
                    })

    print(json.dumps({
        "columns": ["time", "access", "approver"],
        "rows": approvals
    }))

if __name__ == "__main__":
    main()
```

---

## Example: Compliance Check (Go)

```go
// sources/compliance-check/main.go
package main

import (
    "encoding/json"
    "fmt"
    "os"
)

type Input struct {
    Query map[string]interface{} `json:"query"`
    Env   map[string]string      `json:"env"`
}

type Output struct {
    Columns []string                 `json:"columns"`
    Rows    []map[string]interface{} `json:"rows"`
}

func main() {
    var input Input
    if err := json.NewDecoder(os.Stdin).Decode(&input); err != nil {
        fmt.Fprintf(os.Stderr, "Invalid input: %v\n", err)
        os.Exit(1)
    }

    // Run compliance checks against org policy
    checks := runComplianceChecks(input.Query)

    output := Output{
        Columns: []string{"check", "status", "details"},
        Rows:    checks,
    }

    json.NewEncoder(os.Stdout).Encode(output)
}

func runComplianceChecks(query map[string]interface{}) []map[string]interface{} {
    // Implement org-specific compliance logic
    return []map[string]interface{}{
        {"check": "data-retention", "status": "pass", "details": "90 days configured"},
        {"check": "encryption", "status": "pass", "details": "AES-256"},
        {"check": "audit-log", "status": "warn", "details": "Log rotation needed"},
    }
}
```

Build and use:
```bash
go build -o sources/compliance-check sources/compliance-check/main.go
```

---

## LLM Authoring Guidelines

For LLMs generating custom sources:

### DO

```python
# Clear structure
input_data = json.load(sys.stdin)

# Validate input
if "required_field" not in input_data["query"]:
    print("Missing required_field", file=sys.stderr)
    sys.exit(1)

# Return structured data
print(json.dumps({"columns": [...], "rows": [...]}))
```

### DON'T

```python
# Don't use interactive input
input("Enter value: ")  # WRONG

# Don't print unstructured output
print("Status: OK")  # WRONG - must be JSON

# Don't hardcode paths
open("/Users/alice/data.json")  # WRONG - use query params
```

### Minimal Template (Python)

```python
#!/usr/bin/env python3
"""Template for custom tinkerdown source"""
import sys
import json

def main():
    try:
        input_data = json.load(sys.stdin)
        query = input_data.get("query", {})
        env = input_data.get("env", {})

        # TODO: Implement your logic here
        rows = []

        print(json.dumps({
            "columns": ["col1", "col2"],
            "rows": rows
        }))
    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)

if __name__ == "__main__":
    main()
```

### Minimal Template (Shell)

```bash
#!/bin/bash
set -e

# Read input
INPUT=$(cat)

# Extract query params
PARAM=$(echo "$INPUT" | jq -r '.query.param // empty')

# TODO: Implement your logic here

# Return JSON
echo '{"columns":["result"],"rows":[{"result":"value"}]}'
```

---

## Distribution

### Option 1: Git Repository

```
org-tinkerdown-sources/
├── README.md
├── sources/
│   ├── check-permission.py
│   ├── compliance-check
│   ├── approval-status.py
│   └── oncall-lookup.sh
└── install.sh
```

Teams clone and add to PATH:
```bash
git clone git@internal:org-tinkerdown-sources.git ~/.tinkerdown-sources
export PATH="$PATH:$HOME/.tinkerdown-sources/sources"
```

Reference in markdown:
```yaml
sources:
  oncall:
    type: exec
    command: "oncall-lookup.sh"
```

### Option 2: Package Manager

For compiled sources (Go, Rust):
```bash
# Internal package registry
brew install internal/tinkerdown-sources

# Or Go install
go install internal.company/tinkerdown-sources/...@latest
```

### Option 3: Container Registry

For complex dependencies:
```yaml
sources:
  compliance:
    type: exec
    command: "docker run --rm -i internal-registry/compliance-check:latest"
```

### Option 4: Embedded in Runbook Repo

Keep sources alongside templates:
```
runbooks/
├── templates/
│   └── incident-response.md
├── sources/
│   ├── check-permission.py
│   └── approval-status.py
└── README.md
```

---

## Policy Source Examples

### 1. On-Call Lookup

```python
#!/usr/bin/env python3
"""Fetch current on-call from alerting system"""
import sys, json, os, requests

def main():
    input_data = json.load(sys.stdin)
    schedule_id = input_data["query"].get("schedule", "PRIMARY")

    resp = requests.get(
        f"https://api.alerting.internal/oncalls",
        headers={"Authorization": f"Token token={os.environ['ALERTING_API_TOKEN']}"},
        params={"schedule_ids[]": schedule_id}
    )

    oncalls = []
    for oc in resp.json().get("oncalls", []):
        oncalls.append({
            "name": oc["user"]["summary"],
            "email": oc["user"]["email"],
            "until": oc["end"],
        })

    print(json.dumps({"columns": ["name", "email", "until"], "rows": oncalls}))

if __name__ == "__main__":
    main()
```

### 2. Change Freeze Check

```bash
#!/bin/bash
# Check if we're in a change freeze period

INPUT=$(cat)
NOW=$(date +%s)

# Check internal calendar API
FREEZE=$(curl -s "https://internal/change-freeze" | jq -r '.active')

if [ "$FREEZE" = "true" ]; then
  echo '{"columns":["frozen","reason"],"rows":[{"frozen":true,"reason":"Holiday freeze until Jan 2"}]}'
else
  echo '{"columns":["frozen","reason"],"rows":[{"frozen":false,"reason":""}]}'
fi
```

### 3. Runbook Version Check

```python
#!/usr/bin/env python3
"""Check if runbook template is up to date"""
import sys, json, hashlib, requests

def main():
    input_data = json.load(sys.stdin)
    template = input_data["query"]["template"]
    current_hash = input_data["query"]["hash"]

    # Fetch latest from central repo
    resp = requests.get(f"https://runbooks.internal/templates/{template}/hash")
    latest_hash = resp.text.strip()

    print(json.dumps({
        "columns": ["up_to_date", "current", "latest"],
        "rows": [{
            "up_to_date": current_hash == latest_hash,
            "current": current_hash[:8],
            "latest": latest_hash[:8]
        }]
    }))

if __name__ == "__main__":
    main()
```

---

## Security Considerations

### Source Validation

Before using a source:
1. **Review the code** - What does it access?
2. **Check permissions** - What env vars/files does it need?
3. **Audit network calls** - What APIs does it contact?

### Sandboxing (Future)

```yaml
sources:
  untrusted:
    type: exec
    command: "./external-source.py"
    sandbox:
      network: false      # No network access
      filesystem: readonly  # Read-only FS
      env: [ALLOWED_VAR]  # Only these env vars
```

### Signing (Future)

```bash
# Sign a source
tinkerdown source sign ./my-source.py --key ~/.tinkerdown/signing-key

# Verify on use
sources:
  verified:
    type: exec
    command: "./my-source.py"
    require_signature: true
```

---

## Integration with Runbooks

### Pre-flight Checks

```markdown
## Pre-flight Checks

```yaml
sources:
  oncall:
    type: exec
    command: "oncall-lookup.sh"
  freeze:
    type: exec
    command: "change-freeze-check.sh"
  perms:
    type: exec
    command: "check-permission.py"
    query:
      permission: "prod-db-write"
```​

<div lvt-if="freeze[0].frozen" class="warning">
  ⚠️ Change freeze active: {{freeze[0].reason}}
</div>

<div lvt-if="!perms[0].allowed" class="error">
  ❌ You don't have prod-db-write permission
</div>

**Current on-call:** {{oncall[0].name}} ({{oncall[0].email}})
```

### Approval Workflow

```markdown
## Approval Required

```yaml
sources:
  approval:
    type: exec
    command: "approval-status.py"
    query:
      incident_file: "{{env.INCIDENT_FILE}}"
```​

<div lvt-if="approval.length == 0" class="warning">
  ⏳ Waiting for approval...

  <button lvt-on:click="exec:request-approval.sh">Request Approval</button>
</div>

<div lvt-if="approval.length > 0">
  ✅ Approved by {{approval[0].approver}} at {{approval[0].time}}
</div>
```

---

## Summary

| Aspect | Approach |
|--------|----------|
| **Interface** | stdin JSON → stdout JSON, exit codes |
| **Languages** | Any (Python, Go, Shell, Rust, etc.) |
| **LLM-friendly** | Simple template, clear contract |
| **Distribution** | Git repo, package manager, containers |
| **Security** | Review, sandbox (future), signing (future) |

The goal: **Anyone can write a source in 20 lines of Python**, and orgs can build libraries of policy-encoded sources that LLMs can compose into runbooks.
