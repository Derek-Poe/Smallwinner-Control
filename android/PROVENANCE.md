# Android provenance

Author: Derek Poe. License: the repository's MIT license.

This module is derived from the preserved newer Smallwinner Android shell (`uspwin-shell`). Original files remain unchanged in the author's private archive. Private records preserve exact source paths and SHA-256 hashes; machine names and deployment identifiers are not part of this package.

| Original file | Retained engineering | Public maintenance |
| --- | --- | --- |
| `AppCatalogDb.java` | SQLite catalog table, transaction-based replace-all cache, cursor cleanup | Typed rows, package/activity composite key, insertion failures throw and roll back, bounded rows; icon payloads omitted |
| `UspwinBridge.java` | PackageManager `MAIN`/`LAUNCHER` discovery, label sorting, native launch intent | Moved to `AppCatalog.java` and `CatalogPolicy.java`, deterministic sort/deduplication, exported/enabled filtering, exact component selection and fresh-query validation |
| `MainActivity.java` | Embedded loopback web panel and offline recovery concept | New native toolbar/confirmation, exact URL policy, no JavaScript bridge, static offline document, restricted WebView settings and explicit lifecycle cleanup |
| `AndroidManifest.xml`, `network_security_config.xml`, Gradle files | Android application and loopback backend arrangement | Narrow launcher queries, Internet permission only, cleartext exception only for loopback, no accessibility/default-home features, pinned build tool versions |

The original bridge's device-property and privileged recovery/service command paths are excluded. The public module is a maintained extraction, not a claim that the original prototype already had these trust boundaries, tests, or build configuration. Source policy tests, Robolectric tests, the URL policy, native confirmation, bounded catalog logic, and build modernization were added for this release. No original application or operational command was executed to create it.
