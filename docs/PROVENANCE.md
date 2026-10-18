# Original mechanisms and maintenance scope

Smallwinner was a personal Go/Android device-control project. The recovered source snapshot was preserved on 2026-10-17. Its local Git HEAD was `4ebd99a2680179a724c4726bc59b4b52cde4c7a9`, one commit ahead of its then-private remote, and it contained additional uncommitted Android/UI work. The public package is maintained separately; the original working tree remains preserved.

The original `backend/main.go` SHA-256 is `a5ba83e909ef5728c7ed49c244542ba1ac3f6ebf8b73e275fe99173f2cbf8d46`.

| Original mechanism | Public extraction |
| --- | --- |
| `optimizerTick`, `findOptimizerRule`, `clearOptimizerRule` | `Target` and `Controller`: choose by foreground package, apply replacement targets from captured baseline, restore when no rule remains |
| `shouldThermalThrottle` | `ShouldThrottle`: enter at limit, remain at the lower threshold, exit strictly below it; thresholds validated explicitly |
| `applyThermalOverride` | Thermal target chooses balanced CPU mode and captured display density |
| `AppCatalogDb`, Android package query and launcher | Maintained Android native catalog/launcher behind explicit native interaction; see Android provenance notes |
| Native shell plus backend dashboard | Restricted local WebView and separate native controls; no JavaScript/native bridge |

The port retains policy behavior, not the original shell-command implementation or its full feature set. The original also included Linux launchers, network integrations/scanning, root recovery actions, heap/display-size controls, accessibility-key experiments, and broad process/device control. Those are outside this focused release.

New maintenance work includes the simulator's app/process fixtures, typed action/deadline/output boundary, revisioned atomic local store with OS ownership, authenticated same-origin HTTP mutations, embedded dashboard, security refactor, test suite, and CI. CPU modes and display density here are modeled values; no actual governor, display, thermal sensor, process, or package is changed by the Go service.

Historical local notes reported some device/UI testing and an unresolved physical Overview-key issue. Those observations are not current validation, are not used as performance evidence, and are not included as operational logs. The current release is intentionally demonstrated with the simulator.
