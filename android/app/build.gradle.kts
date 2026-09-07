import java.util.Base64
import java.util.Properties
import java.util.zip.ZipFile

plugins {
    id("com.android.application")
    id("kotlin-android")
    // The Flutter Gradle Plugin must be applied after the Android and Kotlin Gradle plugins.
    id("dev.flutter.flutter-gradle-plugin")
}

fun String.toQuotedBuildConfigValue(): String {
    return "\"${replace("\\", "\\\\").replace("\"", "\\\"")}\""
}

fun decodedDartDefines(project: Project): Map<String, String> {
    val encoded = project.findProperty("dart-defines") as String? ?: return emptyMap()
    return encoded
        .split(',')
        .mapNotNull { value ->
            runCatching {
                String(Base64.getDecoder().decode(value), Charsets.UTF_8)
            }.getOrNull()
        }
        .mapNotNull { entry ->
            val separatorIndex = entry.indexOf('=')
            if (separatorIndex <= 0) {
                null
            } else {
                entry.substring(0, separatorIndex) to entry.substring(separatorIndex + 1)
            }
        }
        .toMap()
}

val dartDefines = decodedDartDefines(project)
val analyticsProvider = dartDefines["PICOCLAW_ANALYTICS_PROVIDER"] ?: "none"
val umengAppKey = dartDefines["PICOCLAW_UMENG_APP_KEY"] ?: ""
val umengChannel = dartDefines["PICOCLAW_UMENG_CHANNEL"] ?: "official"
val umengLinkScheme = if (umengAppKey.isNotBlank()) {
    "um.$umengAppKey"
} else {
    "um.placeholder"
}

// Firebase Configuration from dart-define
val firebaseAppId = dartDefines["PICOCLAW_FIREBASE_APP_ID"] ?: ""
val firebaseApiKey = dartDefines["PICOCLAW_FIREBASE_API_KEY"] ?: ""
val firebaseProjectId = dartDefines["PICOCLAW_FIREBASE_PROJECT_ID"] ?: ""
val firebaseMessagingSenderId = dartDefines["PICOCLAW_FIREBASE_MESSAGING_SENDER_ID"] ?: ""
val firebaseStorageBucket = dartDefines["PICOCLAW_FIREBASE_STORAGE_BUCKET"] ?: ""

// ---------------------------------------------------------------------------
// Release contract. See DECISIONS.md, "Release integrity".
//
// Three separate things used to be implicit here and are now stated: how a
// release is signed, where its version number comes from, and whether the
// analytics SDK is part of the build at all. Each of the three had a silent
// default that produced a wrong artifact without failing.
// ---------------------------------------------------------------------------

/**
 * The last versionCode that passed physical acceptance, and therefore the
 * lowest a new build may carry.
 *
 * Read from a tracked file rather than written here so it advances with the
 * product: a constant 55 would keep accepting 56 long after 120 had shipped.
 * `android/release-baseline.properties` says when and how it moves.
 */
fun readAcceptedVersionCodeFloor(baseline: File): Int {
    if (!baseline.isFile) {
        throw GradleException(
            "Cannot determine the release baseline: ${baseline.path} is missing. " +
                "It records the last physically accepted versionCode."
        )
    }
    val properties = Properties()
    baseline.reader(Charsets.UTF_8).use(properties::load)
    val raw = properties.getProperty("lastAcceptedVersionCode")?.trim()
        ?: throw GradleException("${baseline.path} declares no lastAcceptedVersionCode.")
    return raw.toIntOrNull()
        ?: throw GradleException("${baseline.path} lastAcceptedVersionCode=\"$raw\" is not an integer.")
}

val acceptedVersionCodeFloor =
    readAcceptedVersionCodeFloor(rootProject.file("release-baseline.properties"))

/**
 * The tracked application version, read from `pubspec.yaml`.
 *
 * The Flutter Gradle plugin reads `flutter.versionCode` / `flutter.versionName`
 * from `android/local.properties`, which is gitignored, and silently defaults
 * to 1 / "1.0" when they are absent — so a clean checkout built with Gradle
 * produced versionCode 1 while the released artifact was 55. Reading the
 * tracked file directly makes the version reproducible from git alone.
 */
data class TrackedAppVersion(val name: String, val code: Int)

fun readTrackedAppVersion(pubspec: File): TrackedAppVersion {
    if (!pubspec.isFile) {
        throw GradleException("Cannot determine the app version: ${pubspec.path} is missing.")
    }
    val line = pubspec.readLines()
        .firstOrNull { it.startsWith("version:") }
        ?: throw GradleException("Cannot determine the app version: ${pubspec.path} declares no version.")
    val raw = line.removePrefix("version:").trim()
    val match = Regex("""^(\d+\.\d+\.\d+)\+(\d+)$""").matchEntire(raw)
        ?: throw GradleException(
            "Cannot parse the app version \"$raw\" in ${pubspec.path}. " +
                "Expected the form <name>+<code>, for example 0.2.0+55."
        )
    return TrackedAppVersion(match.groupValues[1], match.groupValues[2].toInt())
}

/**
 * Rejects a version declared in the gitignored local.properties.
 *
 * Leaving it readable would reintroduce exactly the split this contract
 * removes: two sources for one number, only one of them in git.
 */
fun assertLocalPropertiesCarriesNoVersion(localProperties: File) {
    if (!localProperties.isFile) return
    val properties = Properties()
    localProperties.reader(Charsets.UTF_8).use(properties::load)
    val declared = listOf("flutter.versionCode", "flutter.versionName").filter(properties::containsKey)
    if (declared.isNotEmpty()) {
        throw GradleException(
            buildString {
                appendLine("${localProperties.path} declares ${declared.joinToString(" and ")}.")
                appendLine("The app version is tracked in pubspec.yaml and nowhere else; a version")
                appendLine("in this gitignored file would override it invisibly.")
                appendLine("Remove those lines. To build a one-off version, pass it explicitly:")
                appendLine("  ./gradlew :app:assembleRelease -PversionCode=<n> -PversionName=<x.y.z>")
            }
        )
    }
}

fun resolveOverriddenVersionCode(project: Project, tracked: Int): Int {
    val raw = (project.findProperty("versionCode") as String?)?.trim() ?: return tracked
    val override = raw.toIntOrNull()
        ?: throw GradleException("-PversionCode=$raw is not an integer.")
    if (override < acceptedVersionCodeFloor) {
        throw GradleException(
            "-PversionCode=$override is below $acceptedVersionCodeFloor, the last " +
                "physically accepted build. Android refuses to install a lower " +
                "versionCode over a higher one."
        )
    }
    return override
}

assertLocalPropertiesCarriesNoVersion(rootProject.file("local.properties"))

val trackedAppVersion = readTrackedAppVersion(rootProject.file("../pubspec.yaml"))
if (trackedAppVersion.code < acceptedVersionCodeFloor) {
    throw GradleException(
        "pubspec.yaml declares versionCode ${trackedAppVersion.code}, below " +
            "$acceptedVersionCodeFloor, the last physically accepted build recorded in " +
            "android/release-baseline.properties. Android refuses to install a lower " +
            "versionCode over a higher one."
    )
}

val resolvedVersionCode = resolveOverriddenVersionCode(project, trackedAppVersion.code)
val resolvedVersionName =
    (project.findProperty("versionName") as String?)?.trim()?.takeIf(String::isNotEmpty)
        ?: trackedAppVersion.name

// --- Signing -------------------------------------------------------------
//
// Production signing material is read from the environment and never lives in
// this repository. When it is absent a release build FAILS: it used to fall
// through to the debug key silently, which shipped artifacts carrying a local
// development identity rather than a release one.
//
// Local physical testing still needs a release-shaped APK, so debug signing
// stays reachable — but only by asking for it in the command line, where it is
// visible in the build log and in shell history.
val releaseKeystorePath = System.getenv("KEYSTORE_PATH").orEmpty().trim()
val releaseKeystorePassword = System.getenv("KEYSTORE_PASSWORD").orEmpty()
val releaseKeyAlias = System.getenv("KEY_ALIAS").orEmpty().trim()
val releaseKeyPassword = System.getenv("KEY_PASSWORD").orEmpty()

val releaseSigningMaterialDeclared =
    releaseKeystorePath.isNotEmpty() &&
        releaseKeystorePassword.isNotEmpty() &&
        releaseKeyAlias.isNotEmpty() &&
        releaseKeyPassword.isNotEmpty()

val releaseSigningMaterialUsable =
    releaseSigningMaterialDeclared && file(releaseKeystorePath).isFile

val allowDebugSigning =
    (project.findProperty("allowDebugSigning") as String?)?.toBoolean() == true

// --- Analytics -----------------------------------------------------------
//
// The Umeng SDK used to be an unconditional dependency, so its manifest
// contributions — advertising ID, AdServices attribution, the Play install
// referrer service and an OEM push permission — merged into every build, and
// the app declared READ_PHONE_STATE for its device probe. None of that runs
// when the provider is "none", which is the default and the only configuration
// PocketClaw ships. An unused SDK must not cost the user a permission prompt.
val umengAnalyticsRequested = analyticsProvider.equals("umeng", ignoreCase = true)

android {
    namespace = "com.lord1egypt.pocketclaw"
    compileSdk = flutter.compileSdkVersion
    ndkVersion = flutter.ndkVersion

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
        isCoreLibraryDesugaringEnabled = true
    }

    kotlinOptions {
        jvmTarget = JavaVersion.VERSION_17.toString()
    }

    defaultConfig {
        // TODO: Specify your own unique Application ID (https://developer.android.com/studio/build/application-id.html).
        applicationId = "com.lord1egypt.pocketclaw"
        // You can update the following values to match your application needs.
        // For more information, see: https://flutter.dev/to/review-gradle-config.
        minSdk = flutter.minSdkVersion
        targetSdk = flutter.targetSdkVersion
        // Tracked in pubspec.yaml, never in the gitignored local.properties.
        versionCode = resolvedVersionCode
        versionName = resolvedVersionName
        buildConfigField("String", "PICOCLAW_ANALYTICS_PROVIDER", analyticsProvider.toQuotedBuildConfigValue())
        // Whether this APK actually packages the analytics SDK. It is set from
        // the same value that decides the dependency below, so the runtime
        // guard cannot drift away from what was built: a build that did not
        // package the SDK reports false, and AnalyticsReporter refuses to touch
        // a class that is not there.
        buildConfigField("boolean", "PICOCLAW_UMENG_PACKAGED", umengAnalyticsRequested.toString())
        buildConfigField("String", "PICOCLAW_UMENG_APP_KEY", umengAppKey.toQuotedBuildConfigValue())
        buildConfigField("String", "PICOCLAW_UMENG_CHANNEL", umengChannel.toQuotedBuildConfigValue())
        buildConfigField("String", "PICOCLAW_UMENG_LINK_SCHEME", umengLinkScheme.toQuotedBuildConfigValue())
        // Pass values to AndroidManifest.xml via manifestPlaceholders
        manifestPlaceholders["PICOCLAW_UMENG_APP_KEY"] = umengAppKey
        manifestPlaceholders["PICOCLAW_UMENG_CHANNEL"] = umengChannel
        manifestPlaceholders["PICOCLAW_UMENG_LINK_SCHEME"] = umengLinkScheme
    }

    signingConfigs {
        create("release") {
            // Production signing material comes from the environment and is
            // never stored in this repository: set KEYSTORE_PATH,
            // KEYSTORE_PASSWORD, KEY_ALIAS and KEY_PASSWORD as CI secrets or in
            // an OS keychain. Nothing here is echoed to the build log.
            if (releaseSigningMaterialUsable) {
                storeFile = file(releaseKeystorePath)
                storePassword = releaseKeystorePassword
                this.keyAlias = releaseKeyAlias
                this.keyPassword = releaseKeyPassword
            }
        }
    }

    buildTypes {
        release {
            // There is no third branch. When neither a real signer nor the
            // explicit local opt-in is present the config stays null and
            // validateReleaseSigning fails the build before anything is
            // packaged — an unsigned or debug-signed release must never be a
            // silent outcome. Debug signing is a development identity: it is
            // not a release identity, and an artifact signed with a different
            // key cannot update an existing installation in place.
            signingConfig = when {
                releaseSigningMaterialUsable -> signingConfigs.getByName("release")
                allowDebugSigning -> signingConfigs.getByName("debug")
                else -> null
            }
            isMinifyEnabled = true
            isShrinkResources = true
            proguardFiles(
                getDefaultProguardFile("proguard-android-optimize.txt"),
                "proguard-rules.pro"
            )
        }
    }

    buildFeatures {
        buildConfig = true
    }

    // jniLibs 打包配置：libpicoclaw*.so 是 Go 静态链接的可执行文件
    packaging {
        jniLibs {
            // 不要 strip libpicoclaw*.so（它们不是标准动态库）
            keepDebugSymbols += "**/libpicoclaw.so"
            keepDebugSymbols += "**/libpicoclaw-web.so"
            // Managed Runtime payloads are executables, not shared libraries.
            // Gradle's strip would rewrite the file and break the SHA-256 the
            // runtime catalog pins, so the tool would resolve as a corrupt
            // install on every device.
            keepDebugSymbols += "**/libpocketclaw-jq.so"
            keepDebugSymbols += "**/libpocketclaw-git.so"
            keepDebugSymbols += "**/libpocketclaw-git-remote-http.so"
            keepDebugSymbols += "**/libpocketclaw-gh.so"
            keepDebugSymbols += "**/libpocketclaw-curl.so"
            keepDebugSymbols += "**/libpocketclaw-rg.so"
            keepDebugSymbols += "**/libpocketclaw-sqlite3.so"
            // Python additionally carries its standard library appended after
            // the ELF image. Stripping would silently discard it and ship an
            // interpreter that cannot import anything at all.
            keepDebugSymbols += "**/libpocketclaw-python.so"
        }
    }
}

flutter {
    source = "../.."
}

dependencies {
    coreLibraryDesugaring("com.android.tools:desugar_jdk_libs:2.1.5")
    implementation("androidx.core:core-ktx:1.18.0")
    // AnalyticsReporter compiles against Umeng in every configuration and is
    // guarded at runtime by isUmengProviderEnabled(). compileOnly keeps that
    // compilation working while keeping the AAR — and therefore its manifest
    // contributions — out of a build that will never call it. An analytics
    // build asks for it by name and gets the real dependency.
    if (umengAnalyticsRequested) {
        implementation("com.umeng.umsdk:common:9.9.1")
        implementation("com.umeng.umsdk:asms:1.8.7.2")
    } else {
        compileOnly("com.umeng.umsdk:common:9.9.1")
        compileOnly("com.umeng.umsdk:asms:1.8.7.2")
    }
    // Referenced by Apache Tika, which arrives transitively with Umeng. Kept
    // unconditional: it contributes no manifest entry and no permission, and
    // dropping it would change what R8 sees in the analytics build for no gain.
    implementation("javax.xml.stream:stax-api:1.0-2")
    // The credential store's destroy-or-preserve rule is a pure function of the
    // failure, so it is checked on the JVM rather than only on a device.
    testImplementation("junit:junit:4.13.2")
}

// Fail a release build that has no authentic signer.
//
// This runs before anything is compiled, so the failure arrives in seconds
// rather than after a three-minute build, and it names the one way to proceed
// on purpose.
tasks.register("validateReleaseSigning") {
    doLast {
        if (releaseSigningMaterialUsable) {
            println("Release signing: production keystore (from the environment).")
            return@doLast
        }
        if (allowDebugSigning) {
            println("Release signing: DEBUG KEY, by explicit -PallowDebugSigning=true.")
            println("  For local development and testing only; not a production release")
            println("  identity. Debug signing material differs between development")
            println("  environments, and an artifact signed with a different key is not an")
            println("  in-place update of an existing installation.")
            return@doLast
        }
        throw GradleException(
            buildString {
                appendLine("Release signing material is missing, so this release cannot be signed.")
                appendLine()
                if (releaseSigningMaterialDeclared) {
                    appendLine("KEYSTORE_PATH is set but names no readable file.")
                } else {
                    appendLine("Set KEYSTORE_PATH, KEYSTORE_PASSWORD, KEY_ALIAS and KEY_PASSWORD")
                    appendLine("in the environment. They are secrets: keep them in CI secret storage")
                    appendLine("or an OS keychain, never in this repository, local.properties or")
                    appendLine("gradle.properties.")
                }
                appendLine()
                appendLine("Debug signing is no longer an implicit fallback. To build a")
                appendLine("release-shaped APK for local development and testing, ask for it:")
                appendLine("  ./gradlew :app:assembleRelease -Ptarget-platform=android-arm64 \\")
                appendLine("      -PallowDebugSigning=true")
                appendLine("That artifact is not a production release identity: debug signing")
                appendLine("material differs between development environments, and an artifact")
                appendLine("signed with a different key is not an in-place update of an")
                appendLine("existing installation.")
            }
        )
    }
}

afterEvaluate {
    // preBuild is the earliest hook that still belongs to the release variant;
    // the packaging tasks are belt-and-braces for anyone invoking them directly.
    listOf("preReleaseBuild", "packageRelease", "bundleRelease").forEach { name ->
        tasks.findByName(name)?.dependsOn("validateReleaseSigning")
    }
}

// Generate Firebase resources from dart-define
tasks.register("generateFirebaseResources") {
    doLast {
        val resDir = file("src/main/res/values")
        resDir.mkdirs()
        
        val stringsXml = file("$resDir/strings.xml")
        
        // Build the content - always generate required fields even if empty
        // to prevent AAPT errors when AndroidManifest references them
        val content = buildString {
            appendLine("<?xml version=\"1.0\" encoding=\"utf-8\"?>")
            appendLine("<resources>")
            appendLine("    <!-- Auto-generated from dart-define, do not edit manually -->")
            
            // Always generate google_app_id (required by AndroidManifest.xml)
            appendLine("    <string name=\"google_app_id\" translatable=\"false\">${firebaseAppId.xmlEscape()}</string>")
            
            if (firebaseApiKey.isNotEmpty()) {
                appendLine("    <string name=\"google_api_key\" translatable=\"false\">${firebaseApiKey.xmlEscape()}</string>")
            }
            if (firebaseProjectId.isNotEmpty()) {
                appendLine("    <string name=\"project_id\" translatable=\"false\">${firebaseProjectId.xmlEscape()}</string>")
                appendLine("    <string name=\"firebase_database_url\" translatable=\"false\">https://${firebaseProjectId.xmlEscape()}.firebaseio.com</string>")
            }
            if (firebaseMessagingSenderId.isNotEmpty()) {
                appendLine("    <string name=\"gcm_defaultSenderId\" translatable=\"false\">${firebaseMessagingSenderId.xmlEscape()}</string>")
            }
            if (firebaseStorageBucket.isNotEmpty()) {
                appendLine("    <string name=\"google_storage_bucket\" translatable=\"false\">${firebaseStorageBucket.xmlEscape()}</string>")
            } else if (firebaseProjectId.isNotEmpty()) {
                appendLine("    <string name=\"google_storage_bucket\" translatable=\"false\">${firebaseProjectId.xmlEscape()}.appspot.com</string>")
            }
            
            appendLine("</resources>")
        }
        
        stringsXml.writeText(content)
        println("Generated Firebase resources at: ${stringsXml.absolutePath}")
        println("Firebase Config: appId=${firebaseAppId.isNotEmpty()}, apiKey=${firebaseApiKey.isNotEmpty()}, projectId=${firebaseProjectId.isNotEmpty()}")
    }
}

// Helper function to escape XML
fun String.xmlEscape(): String {
    return this
        .replace("&", "&amp;")
        .replace("<", "&lt;")
        .replace(">", "&gt;")
        .replace("\"", "&quot;")
        .replace("'", "&apos;")
}

// Ensure resources are generated before any resource processing
afterEvaluate {
    // Hook into resource processing tasks which happen before AAPT linking
    tasks.findByName("mergeDebugResources")?.dependsOn("generateFirebaseResources")
    tasks.findByName("mergeReleaseResources")?.dependsOn("generateFirebaseResources")
    tasks.findByName("processDebugResources")?.dependsOn("generateFirebaseResources")
    tasks.findByName("processReleaseResources")?.dependsOn("generateFirebaseResources")
    // Also hook into pre-build tasks as fallback
    tasks.findByName("preBuild")?.dependsOn("generateFirebaseResources")
}

// Clean up sensitive resources after build
tasks.register("cleanupFirebaseResources") {
    doLast {
        val stringsXml = file("src/main/res/values/strings.xml")
        if (stringsXml.exists()) {
            stringsXml.delete()
            println("Cleaned up Firebase resources for security")
        }
    }
}

// Run cleanup after build completion - use afterEvaluate to ensure tasks exist
afterEvaluate {
    tasks.findByName("assembleDebug")?.finalizedBy("cleanupFirebaseResources")
    tasks.findByName("assembleRelease")?.finalizedBy("cleanupFirebaseResources")
    tasks.findByName("bundleRelease")?.finalizedBy("cleanupFirebaseResources")
}

// Fail the release build if the arm64 native payload is incomplete.
//
// A missing lib/arm64-v8a/libdartjni.so shipped silently for days and produced a
// black screen on every device: JniPlugin loads it from a static initializer and
// GeneratedPluginRegistrant only catches Exception, so the UnsatisfiedLinkError
// escapes FlutterActivity.onCreate before Flutter can draw a frame. The library
// is produced by an externalNativeBuild whose CMake configure is cached under
// ~/.pub-cache, outside this project's build directory, so a failed configure
// survives every local clean.
val requiredArm64NativeLibraries = listOf(
    "lib/arm64-v8a/libdartjni.so",
    "lib/arm64-v8a/libpicoclaw.so",
    "lib/arm64-v8a/libpicoclaw-web.so",
    // Managed Runtime payloads. Each must be packaged under lib/<abi>/lib*.so or
    // the installer never unpacks it into nativeLibraryDir, and nativeLibraryDir
    // is the only directory an app targeting API 29+ may execute from.
    "lib/arm64-v8a/libpocketclaw-jq.so",
    "lib/arm64-v8a/libpocketclaw-git.so",
    // git's transport helper. Without it every https:// remote fails, and the
    // failure would surface as a confusing "unable to find remote helper".
    "lib/arm64-v8a/libpocketclaw-git-remote-http.so",
    "lib/arm64-v8a/libpocketclaw-gh.so",
    "lib/arm64-v8a/libpocketclaw-curl.so",
    "lib/arm64-v8a/libpocketclaw-rg.so",
    "lib/arm64-v8a/libpocketclaw-sqlite3.so",
    // The Python interpreter carries its own standard library appended to the
    // ELF, so this one file is both the executable and the stdlib.
    "lib/arm64-v8a/libpocketclaw-python.so",
)

// The Python payload is an ELF with its standard library appended as a zip.
// Presence alone does not prove it survived packaging: Gradle's native-library
// strip rewrites the file and drops everything past the last section, which
// would ship an interpreter that cannot import anything. Look for the zip's
// end-of-central-directory record, which only an intact payload still has.
val pythonPayloadEntry = "lib/arm64-v8a/libpocketclaw-python.so"

fun verifyPythonStdlibSurvived(apk: File) {
    val bytes = ZipFile(apk).use { zip ->
        val entry = zip.getEntry(pythonPayloadEntry) ?: return
        zip.getInputStream(entry).readBytes()
    }
    val eocd = byteArrayOf(0x50, 0x4b, 0x05, 0x06)
    val from = maxOf(0, bytes.size - 65_557)
    var found = false
    for (i in bytes.size - eocd.size downTo from) {
        if (bytes[i] == eocd[0] && bytes[i + 1] == eocd[1] &&
            bytes[i + 2] == eocd[2] && bytes[i + 3] == eocd[3]
        ) {
            found = true
            break
        }
    }
    if (!found) {
        throw GradleException(
            buildString {
                appendLine("$pythonPayloadEntry has no appended standard library.")
                appendLine("Packaged size: ${bytes.size} bytes.")
                appendLine("Gradle stripped the payload. Add it to")
                appendLine("  packaging { jniLibs { keepDebugSymbols += ... } }")
                appendLine("or the interpreter ships unable to import anything.")
            }
        )
    }
    // The entry point PocketClaw runs is a module inside that same zip. Android's
    // CPython points sys.stdout and sys.stderr at the system log rather than at
    // file descriptors 1 and 2, so without this module every managed run reports
    // an empty stream: print() is a silent no-op that exits 0, and a traceback
    // never appears. A payload that shipped without it would pass every other
    // guard here and fail only on a device, silently.
    val bootstrap = "pocketclaw_bootstrap.py".toByteArray(Charsets.US_ASCII)
    val carriesBootstrap = (0..bytes.size - bootstrap.size).any { at ->
        bootstrap.indices.all { i -> bytes[at + i] == bootstrap[i] }
    }
    if (!carriesBootstrap) {
        throw GradleException(
            buildString {
                appendLine("$pythonPayloadEntry does not contain pocketclaw_bootstrap.py.")
                appendLine("Python would run with its output going to the Android log,")
                appendLine("which reads as an empty stream to the caller. Reinstall it:")
                appendLine("  runtime/install-python-bootstrap.py <payload>")
            }
        )
    }
    println("Verified appended Python standard library in ${apk.name}: ${bytes.size} bytes")
    println("Verified pocketclaw_bootstrap entry point in ${apk.name}")
}

fun verifyArm64NativePayload(apk: File) {
    val packaged = ZipFile(apk).use { zip ->
        zip.entries().asSequence().map { entry -> entry.name }.toSet()
    }
    val missing = requiredArm64NativeLibraries.filter { name -> name !in packaged }
    if (missing.isNotEmpty()) {
        throw GradleException(
            buildString {
                appendLine("Release APK ${apk.name} is missing required arm64-v8a native libraries:")
                missing.forEach { name -> appendLine("  - $name") }
                appendLine("This build would black-screen on a device. Do not ship it.")
                appendLine("If libdartjni.so is missing, purge the cached externalNativeBuild")
                appendLine("configure and rebuild:")
                appendLine("  rm -rf ~/.pub-cache/hosted/pub.dev/jni-*/android/.cxx")
            }
        )
    }
    println("Verified arm64-v8a native payload in ${apk.name}: ${requiredArm64NativeLibraries.joinToString(", ")}")
}

afterEvaluate {
    tasks.findByName("packageRelease")?.doLast {
        outputs.files.asFileTree
            .matching { include("**/*.apk") }
            .forEach { apk ->
                verifyArm64NativePayload(apk)
                verifyPythonStdlibSurvived(apk)
            }
    }
}
