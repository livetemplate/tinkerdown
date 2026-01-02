#!/usr/bin/env node
/**
 * API endpoint status checker for tinkerdown
 * Checks health of configured API endpoints
 *
 * Usage in markdown:
 * ```yaml
 * sources:
 *   apis:
 *     type: exec
 *     command: "node ./sources/api-status.js"
 *     query:
 *       endpoints: "https://api.example.com/health,https://auth.example.com/ping"
 * ```
 *
 * | endpoint | status | latency |
 * |----------|--------|---------|
 * {{#apis}}
 * | {{endpoint}} | {{status}} | {{latency}}ms |
 * {{/apis}}
 */

const https = require('https');
const http = require('http');

async function checkEndpoint(url) {
  const start = Date.now();
  const client = url.startsWith('https') ? https : http;

  return new Promise((resolve) => {
    const req = client.get(url, { timeout: 5000 }, (res) => {
      const latency = Date.now() - start;
      resolve({
        endpoint: url,
        status: res.statusCode >= 200 && res.statusCode < 300 ? 'ok' : 'error',
        latency: latency,
        code: res.statusCode
      });
    });

    req.on('error', (err) => {
      resolve({
        endpoint: url,
        status: 'error',
        latency: Date.now() - start,
        error: err.message
      });
    });

    req.on('timeout', () => {
      req.destroy();
      resolve({
        endpoint: url,
        status: 'timeout',
        latency: 5000,
        error: 'Request timed out'
      });
    });
  });
}

async function main() {
  let inputData = '';

  // Read stdin
  for await (const chunk of process.stdin) {
    inputData += chunk;
  }

  let input;
  try {
    input = JSON.parse(inputData);
  } catch (e) {
    console.error(`Invalid JSON input: ${e.message}`);
    process.exit(1);
  }

  const endpointsStr = input.query?.endpoints || '';
  if (!endpointsStr) {
    console.error("Missing 'endpoints' in query");
    process.exit(1);
  }

  const endpoints = endpointsStr.split(',').map(e => e.trim()).filter(Boolean);

  // Check all endpoints in parallel
  const results = await Promise.all(endpoints.map(checkEndpoint));

  console.log(JSON.stringify({
    columns: ['endpoint', 'status', 'latency', 'code'],
    rows: results
  }));
}

main().catch(err => {
  console.error(`Error: ${err.message}`);
  process.exit(1);
});
