# Third-party notices

The Go backend uses only the Go standard library. The Go toolchain and standard library retain the [Go project's BSD-style license](https://go.dev/LICENSE); no toolchain binaries are shipped in this repository.

The original Smallwinner source carried an MIT notice under the project name USPWIN (2026). This maintenance extraction is also MIT, with attribution retained in `LICENSE`.

The Android module's Gradle wrapper JAR and launcher scripts are included under Apache-2.0; the corresponding [license](android/gradle/GRADLE-LICENSE.txt) and [notice](android/gradle/NOTICE.txt) are bundled. The full Gradle distribution, Android SDK, Java runtime, build caches, compiled APKs, and signing material are not source-package contents. Build dependencies are downloaded by the Android build; see its [build notes](android/README.md).

No third-party media, AI models, operational captures, or device recordings are included. The screenshots show this package's synthetic simulator.
