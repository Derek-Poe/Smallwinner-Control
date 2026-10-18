# Local trust and request contract

The listener binds only IPv4 loopback. Its accepted Host is exactly `127.0.0.1:<port>` and its origin is exactly `http://127.0.0.1:<port>` (1717 by default). No proxy headers are trusted. Do not put a reverse proxy or remote tunnel in front of it.

| Request | Contract |
| --- | --- |
| `GET /`, `/app.js`, `/style.css` | Embedded UI, no token embedded |
| `GET /api/v1/state` | Synthetic catalog/processes, observation, baseline, target, policy and errors |
| `GET /api/v1/config` | Current configuration including revision |
| `POST /api/v1/simulation` | Exact known foreground package and finite temperature from 0 through 120 |
| `POST /api/v1/config` | Full validated config with current revision; successful commit increments revision |
| `POST /api/v1/actions/restore` | Empty JSON object; baseline until next observation/config mutation |

Every POST requires `Content-Type: application/json`, one exact `Origin` header, and one `Authorization: Bearer <session-token>` header. Any supplied Origin is checked even on reads; nonbrowser local GET requests may omit it. Unknown routes, arbitrary actions, query parameters, duplicate authentication/origin headers, unknown JSON fields, trailing documents, oversize bodies, and unsupported methods are rejected. JSON syntax follows Go's standard decoder (including its ordinary duplicate-field behavior); configurations still pass complete semantic validation.

The random 256-bit token lives in process memory and is printed once to the local operator. The dashboard retains it in JavaScript memory only, not URLs, cookies, local storage, or rendered HTML. A reload requires reentry. CSP restricts scripts, styles, network connections, and images to the application itself; framing and forms are disabled. No third-party scripts or CDN resources are loaded.

Four admission slots bound simultaneous mutations; excess requests receive 429. Controller operations are serialized. HTTP header/read/write/idle deadlines and a 16 KiB body limit bound ordinary client input; a hostile local process can still exhaust local OS resources. The application is not intended to run under an untrusted multi-user service account.

The command runner accepts only `apply_resources` with validated CPU mode and density. No command string, executable name, filesystem path, package launch request, or host address can be submitted through the public API. Output is counted and discarded with a 1 KiB cap. A 250 ms context bounds the cooperative simulator. Future adapters are trusted code and must honor cancellation; there is no detached goroutine that continues a timed-out command behind a released slot.

The Android module has an independent, native application catalog and explicit app-launch confirmation. It exposes no JavaScript/native bridge and cannot turn a web request into a native launch. Backend tokens remain the dashboard's responsibility. See its README for navigation, offline page, and package-visibility details.

Configuration files are not secret-bearing. The directory and its parents must be controlled by the current user; adversarial symlink replacement, network filesystem locking semantics, and distributed writers are out of scope. The public package contains no historical operational configuration or runtime logs.
