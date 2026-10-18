# Android shell

This source module retains Smallwinner's native app catalog, explicit app launching, transactional SQLite cache, and offline recovery screen. The public release uses the loopback **simulator**. The desktop simulator does not provide an Android device adapter.

The web panel opens only `http://127.0.0.1:1717/`. Its resource allowlist is `/`, `/app.js`, `/style.css`, `/api/v1/state`, `/api/v1/config`, `/api/v1/simulation`, and `/api/v1/actions/restore`. Other origins, ports, paths, URL credentials, encoded paths, query strings, fragments, and external navigation are denied. The native shell has **no JavaScript/native bridge**. Apps, Refresh apps, and the launch confirmation are Android controls outside the page. Web content cannot request an app launch.

The backend UI accepts its random session token through its own input and keeps it in JavaScript memory. Android stores no token, disables DOM storage/cookies, disables file/content access and WebView debugging, and keeps app names in a native SQLite cache that is never sent to the simulator. Cloud-backup and device-transfer rules exclude all app data. A page reload loses the token. Cleartext is allowed only for `127.0.0.1`; all other hosts inherit the network configuration's cleartext denial. The fixed loopback address is a routing boundary, not authentication of the local server process. Run the trusted simulator; the page and its API authorization remain the backend's responsibility. The native shell never gives that page device-control privileges.

If the simulator is unavailable, the shell renders a bundled static recovery document with JavaScript disabled. Native Retry and app controls remain available. Retry reconnects to the same fixed address. There are no privileged shell commands, reboot controls, service management, accessibility service, default-home role, broad package visibility permission, device installation steps, or remote-origin setting in this module.

## Build and checks

Use JDK 17 or 21 and Android SDK Platform 35 with Build Tools 35.0.0. Set `ANDROID_HOME` to the SDK directory, then from `android/` run:

```sh
./gradlew --no-daemon testDebugUnitTest lintDebug assembleDebug assembleRelease
python3 verify_source.py
```

On Windows use `gradlew.bat` and `python` if installed. The wrapper pins Gradle 8.9 with the official distribution checksum; the build pins Android Gradle Plugin 8.7.3. Release compilation produces an **unsigned APK**; no signing credentials are included. The debug build uses only the Android toolchain's generated development key and is not a distributable signed release. Build outputs are ignored.

Tests exercise adversarial URLs, deterministic and capped catalog selection, stale selections, explicit launcher intents, package visibility filters, and actual SQLite transaction rollback under Robolectric. The manifest/source check separately guards the capability boundary. APK compilation and JVM tests are not an Android WebView/device execution test. This release has no physical-device installation, device app-launch test, hardware recovery validation, or on-device simulator connection claim. The desktop simulator is validated separately by the repository's Go/browser checks.

## Native catalog

The launcher query uses the manifest's `MAIN`/`LAUNCHER` visibility declaration. Enabled exported activities are normalized by exact package/activity, excluding this app, sorted deterministically using `Locale.ROOT`, and capped at 512 displayed/cached entries. Multiple launch activities from the same package remain distinct. A refresh replaces the cache in one SQLite transaction; failed insertion retains the previous catalog. Database work and closing share one FIFO worker.

Selection requires a native confirmation dialog and a fresh launcher query. The intent specifies the selected component, `MAIN`, and `LAUNCHER`; it carries no URL, extras, or command. Apps can still disappear or become restricted between the check and launch; that failure returns a generic native status. The count cap bounds the displayed/cache rows, not Android's initial PackageManager result allocation. Catalog discovery can reflect the device's user/profile visibility rules.

Original icon serialization was omitted: a text catalog avoids storing arbitrary app icon payloads and keeps the public module focused. See [PROVENANCE.md](PROVENANCE.md) for retained and modernized portions.

Android guidance: [native bridge risks](https://developer.android.com/privacy-and-security/risks/insecure-webview-native-bridges), [loading local content](https://developer.android.com/develop/ui/views/layout/webapps/load-local-content), and [declaring package visibility](https://developer.android.com/training/package-visibility/declaring).

## Build dependency notices

The generated Gradle wrapper is the only bundled third-party binary; its Apache 2.0 license and notice are included in [gradle/GRADLE-LICENSE.txt](gradle/GRADLE-LICENSE.txt) and [gradle/NOTICE.txt](gradle/NOTICE.txt). Android Gradle Plugin 8.7.3 is fetched from Google's Maven repository; its [published POM](https://dl.google.com/dl/android/maven2/com/android/tools/build/gradle/8.7.3/gradle-8.7.3.pom) records its license. Android SDK tools/platforms are separately installed under the [Android SDK terms](https://developer.android.com/studio/terms). Test dependencies are [Robolectric 4.14.1 (MIT, with Apache 2.0 files identified upstream)](https://github.com/robolectric/robolectric/blob/master/LICENSE) and [JUnit 4.13.2 (Eclipse Public License 1.0)](https://github.com/junit-team/junit4/blob/main/LICENSE-junit.txt). Their binaries and transitive dependencies are downloaded into local build caches rather than vendored or included as app runtime dependencies.
