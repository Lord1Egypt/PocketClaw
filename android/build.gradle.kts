allprojects {
    repositories {
        google()
        mavenCentral()
    }
}

val newBuildDir: Directory =
    rootProject.layout.buildDirectory
        .dir("../../build")
        .get()
rootProject.layout.buildDirectory.value(newBuildDir)

subprojects {
    val newSubprojectBuildDir: Directory = newBuildDir.dir(project.name)
    project.layout.buildDirectory.value(newSubprojectBuildDir)
}
subprojects {
    project.evaluationDependsOn(":app")
}

// libdartjni.so is built by the jni plugin's own CMake project. Clang records
// the CMake build directory (keyed by a per-checkout hash) and the pub-cache
// source path in DWARF, and the linker hashes both into the GNU build ID, so
// two checkouts produced libraries that differed only in that ID. These flags
// make the recorded paths machine-independent; the shipped library is stripped
// either way.
fun pinnedNdkVersion(): String =
    rootProject.file("../runtime/toolchains.env").readLines()
        .first { it.startsWith("POCKETCLAW_NDK_VERSION=") }
        .substringAfter("=").trim()

fun androidSdkDir(): String? {
    val properties = java.util.Properties()
    rootProject.file("local.properties").takeIf { it.isFile }?.reader()?.use(properties::load)
    return properties.getProperty("sdk.dir") ?: System.getenv("ANDROID_HOME")
}

subprojects {
    if (name == "jni") {
        plugins.withId("com.android.library") {
            val pubCache = System.getenv("PUB_CACHE")
                ?: "${System.getProperty("user.home")}/.pub-cache"
            // F-Droid's builder exports the NDK location; elsewhere AGP uses the
            // SDK's copy of the version Flutter pins, which runtime/toolchains.env
            // pins too.
            val ndkRoot = listOf("ANDROID_NDK_HOME", "ANDROID_NDK_ROOT", "ANDROID_NDK")
                .firstNotNullOfOrNull { System.getenv(it)?.takeIf(String::isNotBlank) }
                ?: androidSdkDir()?.let { "$it/ndk/${pinnedNdkVersion()}" }
            val flags = mutableListOf(
                "-fdebug-compilation-dir=.",
                "-ffile-prefix-map=$pubCache=/pub-cache",
            )
            if (ndkRoot != null) flags += "-ffile-prefix-map=$ndkRoot=/android-ndk"
            extensions.getByName("android").withGroovyBuilder {
                "defaultConfig" {
                    "externalNativeBuild" {
                        "cmake" {
                            "cFlags"(*flags.toTypedArray())
                        }
                    }
                }
            }
        }
    }
}

tasks.register<Delete>("clean") {
    delete(rootProject.layout.buildDirectory)
}
