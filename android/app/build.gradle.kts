import java.io.File
import java.util.Base64
import java.util.Properties
import java.util.zip.ZipFile
import groovy.json.JsonSlurper

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

// PC-DEF-060. The Dashboard opened in a desktop browser has no Android host to run
// managed Telegram pairing, so Core has to run it instead -- and Core does not know
// where the onboarding service lives. The URL already reaches the APK as a dart-define
// from android/official-onboarding.properties; reading it here too means Kotlin can
// hand the same value to Core's environment without a second place to set it.
val onboardingBaseUrl = dartDefines["POCKETCLAW_ONBOARDING_BASE_URL"] ?: ""


// ---------------------------------------------------------------------------
// Release contract. See DECISIONS.md, "Release integrity".
//
// Two things used to be implicit here and are now stated: how a release is
// signed and where its version number comes from. Each had a silent default
// that produced a wrong artifact without failing.
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

// The four fields, named once. The names are safe to print; the values never
// are, so only the keys of this map ever reach a log or an error message.
val releaseSigningFields = linkedMapOf(
    "KEYSTORE_PATH" to releaseKeystorePath,
    "KEYSTORE_PASSWORD" to releaseKeystorePassword,
    "KEY_ALIAS" to releaseKeyAlias,
    "KEY_PASSWORD" to releaseKeyPassword,
)

val missingReleaseSigningFields = releaseSigningFields.filterValues { it.isEmpty() }.keys.toList()

val releaseSigningMaterialDeclared = missingReleaseSigningFields.isEmpty()

// Some but not all. This is the dangerous shape: it says someone meant to sign
// for production and got a name wrong, so it must never resolve to anything —
// least of all to the debug key, which would hand back a plausible-looking
// artifact carrying a development identity.
val releaseSigningPartiallyDeclared =
    missingReleaseSigningFields.isNotEmpty() &&
        missingReleaseSigningFields.size < releaseSigningFields.size

// A production keystore inside the repository is refused outright.
//
// .gitignore stops an accidental `git add`; it does nothing about `git add -f`,
// a future pattern change, or a keystore copied in "just for this build" and
// forgotten. The signing path is the last place that can still say no, so it
// does. Resolved canonically, because a relative path or a symlink out and back
// in would otherwise walk straight past a prefix comparison.
val repositoryRoot = rootProject.projectDir.parentFile.canonicalFile

fun keystoreIsInsideRepository(path: String): Boolean {
    if (path.isEmpty()) return false
    val candidate = File(path).let { if (it.isAbsolute) it else File(rootProject.projectDir, path) }
    val resolved = runCatching { candidate.canonicalFile }.getOrElse { return false }
    return generateSequence(resolved) { it.parentFile }.any { it == repositoryRoot }
}

val releaseKeystoreInsideRepository = keystoreIsInsideRepository(releaseKeystorePath)

val releaseSigningMaterialUsable =
    releaseSigningMaterialDeclared &&
        !releaseKeystoreInsideRepository &&
        file(releaseKeystorePath).isFile

val allowDebugSigning =
    (project.findProperty("allowDebugSigning") as String?)?.toBoolean() == true

// A repository-signable release: built without any signature so a repository
// (F-Droid) can sign it with its own key, or attach the upstream signature once
// the build reproduces. It is the production build with the signing step left
// out -- same R8, Dart hardening and ABI -- so its bytes are what the upstream
// APK is before signing. It is its own explicit mode, never a fallback: it
// refuses to run beside production signing material or the debug opt-in.
val unsignedReleaseRequested =
    (project.findProperty("pocketclawUnsignedRelease") as String?)?.toBoolean() == true

// --- Dart release hardening ---------------------------------------------
//
// Flutter 3.47.1's Gradle plugin consumes the target, obfuscation, and split
// properties below. PocketClaw adds one explicit mode marker so a release
// compile cannot be mistaken for a hardened compile merely because one flag
// happened to be set. The canonical entry point is
// tool/build_hardened_android.py.
val dartHardeningMode = (project.findProperty("pocketclawDartHardening") as String?)?.trim()
val dartObfuscationProperty = (project.findProperty("dart-obfuscation") as String?)?.trim()
val splitDebugInfoProperty = (project.findProperty("split-debug-info") as String?)?.trim()
val dartTargetPlatformProperty = (project.findProperty("target-platform") as String?)?.trim()

fun validateGeneratedDartPackageMapping(packageConfig: File) {
    if (!packageConfig.isFile) {
        throw GradleException(
            "${packageConfig.path} is missing. Run the pinned `flutter pub get`, then use " +
                "tool/build_hardened_android.py."
        )
    }
    val parsed = runCatching { JsonSlurper().parse(packageConfig) as Map<*, *> }
        .getOrElse { error ->
            throw GradleException("Cannot parse ${packageConfig.path}: ${error.message}")
        }
    val packages = parsed["packages"] as? List<*>
        ?: throw GradleException("${packageConfig.path} has no package list.")
    val mappings = packages
        .filterIsInstance<Map<*, *>>()
        .filter { entry -> entry["name"] == "pocketclaw_generated" }
    val mappingIsControlled = mappings.size == 1 &&
        mappings.single()["rootUri"] == "flutter_build/" &&
        mappings.single()["packageUri"] == "./"
    if (!mappingIsControlled) {
        throw GradleException(
            buildString {
                appendLine("The controlled generated-Dart package mapping is absent or malformed.")
                appendLine("A release compile would embed the checkout-specific absolute URI for")
                appendLine(".dart_tool/flutter_build/dart_plugin_registrant.dart.")
                appendLine("Use tool/build_hardened_android.py; it prepares the deterministic")
                appendLine("package:pocketclaw_generated mapping before Gradle starts.")
            }
        )
    }
}

fun validatePrivateDartSymbolDirectory(raw: String) {
    val repository = rootProject.projectDir.parentFile.canonicalFile
    val requested = File(raw).let { candidate ->
        if (candidate.isAbsolute) candidate else File(repository, raw)
    }.canonicalFile
    if (requested == repository || generateSequence(repository) { it.parentFile }.any { it == requested }) {
        throw GradleException("split-debug-info cannot name the repository or one of its parents.")
    }
    if (generateSequence(requested) { it.parentFile }.any { it == repository }) {
        val relative = requested.relativeTo(repository).invariantSeparatorsPath
        val safelyIgnored = relative == "build/private-symbols" ||
            relative.startsWith("build/private-symbols/") ||
            relative == "split-debug-info" || relative.startsWith("split-debug-info/") ||
            relative == "symbols" || relative.startsWith("symbols/")
        if (!safelyIgnored) {
            throw GradleException(
                "Repository-local split-debug-info must be under build/private-symbols/, " +
                    "split-debug-info/, or symbols/."
            )
        }
    }
}

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
        // arm64-v8a is the only ABI PocketClaw runs on: Core, the Managed
        // Runtime and the Flutter engine are built for it alone. Without this
        // filter, plugin stubs (libdartjni, libdatastore_shared_counter) for
        // armeabi-v7a and x86_64 made the APK advertise ABIs it cannot start on.
        ndk {
            abiFilters += listOf("arm64-v8a")
        }
        buildConfigField(
            "String",
            "POCKETCLAW_ONBOARDING_BASE_URL",
            onboardingBaseUrl.toQuotedBuildConfigValue(),
        )
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
            // Partial production material outranks the debug opt-in: someone
            // who set three of the four fields was aiming at a production
            // build, and quietly giving them a debug-signed one instead is the
            // silent downgrade this whole arrangement exists to prevent.
            signingConfig = when {
                unsignedReleaseRequested -> null
                releaseSigningMaterialUsable -> signingConfigs.getByName("release")
                releaseSigningPartiallyDeclared -> null
                releaseKeystoreInsideRepository -> null
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

    // jniLibs packaging: libpocketclaw*.so are statically linked Go executables.
    packaging {
        jniLibs {
            // Never strip libpocketclaw*.so: they are not ordinary shared libraries.
            keepDebugSymbols += "**/libpocketclaw.so"
            keepDebugSymbols += "**/libpocketclaw-web.so"
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
    // The credential store's destroy-or-preserve rule is a pure function of the
    // failure, so it is checked on the JVM rather than only on a device.
    testImplementation("junit:junit:4.13.2")
}

// Fail every release compile unless the complete H3A Dart-hardening contract is
// selected. Debug builds are unaffected. The Python entry point also verifies
// the actual APK and split artifact after assembly; this task protects direct
// Gradle callers before the compiler does expensive work.
tasks.register("validateDartHardening") {
    doLast {
        if (dartHardeningMode != "true") {
            throw GradleException(
                buildString {
                    appendLine("Dart-hardened release mode was not selected.")
                    appendLine("Release compilation is supported through:")
                    appendLine("  python3 tool/build_hardened_android.py --signing <local-test|production>")
                    appendLine("That entry point enables obfuscation, split debug info, and the")
                    appendLine("controlled generated-source URI as one fail-closed contract.")
                }
            )
        }
        if (dartObfuscationProperty != "true") {
            throw GradleException(
                "Hardened mode requires the exact project property -Pdart-obfuscation=true."
            )
        }
        val splitDebugInfo = splitDebugInfoProperty
        if (splitDebugInfo.isNullOrEmpty()) {
            throw GradleException(
                "Hardened mode requires a non-empty -Psplit-debug-info directory."
            )
        }
        if (dartTargetPlatformProperty != "android-arm64") {
            throw GradleException(
                "Hardened PocketClaw APKs require -Ptarget-platform=android-arm64."
            )
        }
        val competingUriProperties = listOf(
            "filesystem-roots",
            "filesystem-scheme",
            "extra-front-end-options",
        ).filter(project::hasProperty)
        if (competingUriProperties.isNotEmpty()) {
            throw GradleException(
                "Hardened mode rejects competing generated-source options: " +
                    competingUriProperties.joinToString(", ")
            )
        }
        validatePrivateDartSymbolDirectory(splitDebugInfo)
        validateGeneratedDartPackageMapping(rootProject.file("../.dart_tool/package_config.json"))
        println("Dart hardening: obfuscation + private split debug info + controlled package URI.")
    }
}

// Fail a release build that has no authentic signer.
//
// This runs before anything is compiled, so the failure arrives in seconds
// rather than after a three-minute build, and it names the one way to proceed
// on purpose.
tasks.register("validateReleaseSigning") {
    doLast {
        if (unsignedReleaseRequested) {
            val declared = releaseSigningFields.filterValues { it.isNotEmpty() }.keys
            if (declared.isNotEmpty() || allowDebugSigning) {
                throw GradleException(
                    buildString {
                        appendLine("An unsigned release was requested beside a signing configuration.")
                        appendLine()
                        if (declared.isNotEmpty()) {
                            appendLine("Declared: " + declared.joinToString(", "))
                        }
                        if (allowDebugSigning) appendLine("Declared: -PallowDebugSigning=true")
                        appendLine()
                        appendLine("-PpocketclawUnsignedRelease=true builds an artifact for a repository")
                        appendLine("to sign. It never uses a key, so a key being present means two")
                        appendLine("intentions were mixed. Unset the signing variables, or drop the")
                        appendLine("unsigned property to build a signed release.")
                    }
                )
            }
            println("Release signing: NONE. UNSIGNED / REPOSITORY-SIGNABLE, by explicit")
            println("  -PpocketclawUnsignedRelease=true. Not installable until a repository signs it.")
            return@doLast
        }
        if (releaseSigningMaterialUsable) {
            println("Release signing: production keystore (from the environment).")
            return@doLast
        }
        // Checked before the debug opt-in, deliberately. Both of these mean
        // "production signing was intended and is wrong", and answering them
        // with a debug-signed artifact would be answering a different question.
        if (releaseKeystoreInsideRepository) {
            throw GradleException(
                buildString {
                    appendLine("KEYSTORE_PATH points inside this repository.")
                    appendLine()
                    appendLine("A production keystore must live outside the working tree. Ignoring")
                    appendLine("it is not protection: `git add -f`, a changed ignore pattern or a")
                    appendLine("copy left behind after a build would all commit it, and a signing")
                    appendLine("key in history cannot be un-published — it can only be rotated,")
                    appendLine("which invalidates every update path for already-installed apps.")
                    appendLine()
                    appendLine("Move the keystore somewhere outside the repository and point")
                    appendLine("KEYSTORE_PATH at it. See docs/RELEASE_SIGNING.md.")
                }
            )
        }
        if (releaseSigningPartiallyDeclared) {
            throw GradleException(
                buildString {
                    appendLine("Production signing is partially configured, so this build stops.")
                    appendLine()
                    appendLine("Missing: " + missingReleaseSigningFields.joinToString(", "))
                    appendLine()
                    appendLine("Field names only — no value is read back or printed. Some of the")
                    appendLine("four are set, which means production signing was intended; a")
                    appendLine("typo in one name would otherwise fall through to the debug key")
                    appendLine("and hand back an artifact that looks like a release and is not.")
                    appendLine("-PallowDebugSigning=true does not apply here: fix the")
                    appendLine("configuration, or unset all four to build a local test artifact.")
                }
            )
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
    listOf("preReleaseBuild", "compileFlutterBuildRelease", "packageRelease", "bundleRelease")
        .forEach { name ->
            tasks.findByName(name)?.dependsOn("validateDartHardening")
        }
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
    "lib/arm64-v8a/libpocketclaw.so",
    "lib/arm64-v8a/libpocketclaw-web.so",
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
