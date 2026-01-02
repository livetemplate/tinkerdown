#!/usr/bin/env python3
"""
Check if operator has a specific permission.
Example policy-encoded source for tinkerdown.

Usage in markdown:
```yaml
sources:
  perm:
    type: exec
    command: "./sources/check-permission.py"
    query:
      permission: "prod-db-write"
```

<div lvt-if="perm[0].allowed">✅ Access granted</div>
<div lvt-if="!perm[0].allowed">❌ Access denied</div>
"""
import sys
import json
import os

# Simulated permission database (replace with LDAP, API, etc.)
PERMISSIONS = {
    "alice": ["prod-db-read", "prod-db-write", "deploy-staging"],
    "bob": ["prod-db-read", "deploy-staging", "deploy-prod"],
    "charlie": ["prod-db-read"],
}

def main():
    try:
        input_data = json.load(sys.stdin)
        query = input_data.get("query", {})
        env = input_data.get("env", {})

        permission = query.get("permission")
        operator = env.get("operator", os.environ.get("USER", "unknown"))

        if not permission:
            print("Missing 'permission' in query", file=sys.stderr)
            sys.exit(1)

        user_perms = PERMISSIONS.get(operator, [])
        allowed = permission in user_perms

        print(json.dumps({
            "columns": ["operator", "permission", "allowed"],
            "rows": [{
                "operator": operator,
                "permission": permission,
                "allowed": allowed
            }]
        }))

    except json.JSONDecodeError as e:
        print(f"Invalid JSON input: {e}", file=sys.stderr)
        sys.exit(1)
    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)

if __name__ == "__main__":
    main()
