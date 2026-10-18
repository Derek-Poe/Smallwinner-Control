# Reproduce the policy walkthrough

Run `go run ./cmd/smallwinner` on Linux/WSL, open the printed loopback URL, and unlock controls using that process's token. The original Android/root service must not be started for this demo.

| Step | Foreground | Temperature | Expected policy | CPU / density |
| --- | --- | --- | --- | --- |
| Startup | Home | 35°C | baseline | balanced / 240 |
| 1 | Game | 35°C | demo.game | performance / 280 |
| 2 | Game | 80°C | thermal | balanced / 240 |
| 3 | Game | 72°C | thermal | balanced / 240 |
| 4 | Game | 69°C | demo.game | performance / 280 |
| 5 | Home | 35°C | baseline | balanced / 240 |

These values are fixtures, not measurements. The temperature is an input; changing CPU mode does not model heat production, battery drain, frame rate, or actual kernel governor behavior.

`-demo` prints the same six-step walkthrough without opening a listener. Use a fresh `-state` directory to get the default rules; an existing directory uses its saved thresholds and rules. The output deliberately contains no timestamps or random identifiers, so the same configuration produces byte-identical JSON.

For API clients, use the exact origin and a token from the process. This example uses a variable already populated in your shell; avoid putting real tokens into shared command histories or screenshots:

```sh
curl --fail-with-body http://127.0.0.1:1717/api/v1/simulation \
  -H 'Origin: http://127.0.0.1:1717' \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer $SMALLWINNER_TOKEN" \
  --data '{"foreground":"demo.game","temperature_c":80}'
```

For configuration edits, fetch `/api/v1/config`, change thresholds or rules, and POST the full object including the fetched revision. A concurrent successful writer makes that revision stale: HTTP 409 is expected, and the client must reload before deciding what to change. The dashboard keeps this revision and reloads it after success or conflict.

Stop the server with Ctrl+C. Shutdown stops admission, waits up to two seconds for HTTP handlers, and requests baseline restoration after a graceful drain. A forced termination cannot execute restoration; the included simulator disappears with the process, while committed configuration survives. This behavior must be reconsidered before implementing real hardware effects.

The companion Android project can be built and unit-tested without a device. A backend on a desktop is not reachable at the Android device's own loopback address without a separately reviewed development connection. Device setup, port forwarding, rooting, and physical keys are intentionally outside this simulator release.
