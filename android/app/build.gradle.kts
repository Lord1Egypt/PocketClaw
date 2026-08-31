import java.util.Base64
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
        versionCode = flutter.versionCode
        versionName = flutter.versionName
        buildConfigField("String", "PICOCLAW_ANALYTICS_PROVIDER", analyticsProvider.toQuotedBuildConfigValue())
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
            // Signing configuration loaded from environment variables
            // For CI/CD: set KEYSTORE_PATH, KEYSTORE_PASSWORD, KEY_ALIAS, KEY_PASSWORD as secrets
            val keystorePath = System.getenv("KEYSTORE_PATH") ?: ""
            val keystorePassword = System.getenv("KEYSTORE_PASSWORD") ?: ""
            val keyAlias = System.getenv("KEY_ALIAS") ?: ""
            val keyPassword = System.getenv("KEY_PASSWORD") ?: ""
            
            if (keystorePath.isNotEmpty() && keystorePassword.isNotEmpty() && 
                keyAlias.isNotEmpty() && keyPassword.isNotEmpty()) {
                storeFile = file(keystorePath)
                storePassword = keystorePassword
                this.keyAlias = keyAlias
                this.keyPassword = keyPassword
            }
        }
    }

    buildTypes {
        release {
            signingConfig = if (signingConfigs.named("release").get().storeFile?.exists() == true) {
                signingConfigs.getByName("release")
            } else {
                // Fallback to debug signing for local development
                signingConfigs.getByName("debug")
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
    implementation("com.umeng.umsdk:common:9.9.1")
    implementation("com.umeng.umsdk:asms:1.8.7.2")
    implementation("javax.xml.stream:stax-api:1.0-2")
    // The credential store's destroy-or-preserve rule is a pure function of the
    // failure, so it is checked on the JVM rather than only on a device.
    testImplementation("junit:junit:4.13.2")
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
