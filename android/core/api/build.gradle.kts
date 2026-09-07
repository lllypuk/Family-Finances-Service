import org.jetbrains.kotlin.gradle.dsl.JvmTarget

plugins {
    alias(libs.plugins.android.library)
}

apply(plugin = "org.jetbrains.kotlin.plugin.serialization")

// Каталог с результатом генератора. Он коммитится: сборка и `check` обязаны работать без сети,
// а генерация тянет артефакты из интернета и запускается отдельной целью.
val generatedDir = layout.projectDirectory.dir("generated")
val apiSpec = rootProject.layout.projectDirectory.file("../docs/api/openapi.yaml")

android {
    namespace = "tech.shatrov.familyfinances.core.api"
    compileSdk = libs.versions.compileSdk.get().toInt()

    defaultConfig {
        minSdk = libs.versions.minSdk.get().toInt()
    }

    compileOptions {
        sourceCompatibility = JavaVersion.toVersion(libs.versions.jvmTarget.get())
        targetCompatibility = JavaVersion.toVersion(libs.versions.jvmTarget.get())
    }

    kotlin {
        compilerOptions {
            jvmTarget.set(JvmTarget.fromTarget(libs.versions.jvmTarget.get()))
        }
    }

    sourceSets {
        getByName("main") {
            kotlin.directories.add(generatedDir.dir("kotlin").asFile.path)
        }
    }

    testOptions {
        unitTests.isIncludeAndroidResources = true
    }
}

// Генератор резолвится своей конфигурацией: в classpath сборки он не попадает, поэтому
// `assemble` и `check` не ходят за ним в сеть.
val apiGenerator = configurations.create("apiGenerator")

dependencies {
    apiGenerator(libs.openapi.generator.cli)

    api(libs.retrofit)
    api(libs.kotlinx.serialization.json)
    implementation(libs.retrofit.converter.kotlinx)
    implementation(libs.okhttp)
    // SharedFlow «сессия кончилась» видна из :app, поэтому не implementation.
    api(libs.kotlinx.coroutines.android)

    testImplementation(libs.junit)
    testImplementation(libs.robolectric)
    testImplementation(libs.androidx.test.core)
    testImplementation(libs.kotlinx.coroutines.test)
    testImplementation(libs.okhttp.mockwebserver)
}

tasks.register<JavaExec>("generateApiClient") {
    group = "build"
    description = "Сгенерировать клиент из docs/api/openapi.yaml (нужна сеть)"
    classpath = apiGenerator
    mainClass.set("org.openapitools.codegen.OpenAPIGenerator")

    inputs.file(apiSpec).withPathSensitivity(PathSensitivity.RELATIVE)
    outputs.dir(generatedDir)

    // Каталог вычищается перед запуском: генератор не удаляет файлы, и схема, выпавшая из
    // спеки, осталась бы закоммиченным мусором, который ещё и компилируется.
    val outDir = generatedDir.asFile
    doFirst {
        outDir.deleteRecursively()
    }

    args(
        "generate",
        "-i", apiSpec.asFile.absolutePath,
        "-g", "kotlin",
        "-o", outDir.absolutePath,
        "--additional-properties",
        listOf(
            "library=jvm-retrofit2",
            "serializationLibrary=kotlinx_serialization",
            "dateLibrary=java8",
            // Без useCoroutines методы возвращают Call<T>, без useResponseAsReturnType
            // тело ошибки до разбора конверта не доходит.
            "useCoroutines=true",
            "useResponseAsReturnType=true",
            "packageName=tech.shatrov.familyfinances.core.api",
            "apiPackage=tech.shatrov.familyfinances.core.api",
            "modelPackage=tech.shatrov.familyfinances.core.api",
            // Иначе вывод ляжет в generated/src/main/kotlin.
            "sourceFolder=kotlin",
        ).joinToString(","),
        // Только интерфейсы, модели и один вспомогательный файл, на который ссылаются
        // интерфейсы: иначе рядом окажутся чужие build.gradle, README и docs/.
        "--global-property",
        "apis,models,supportingFiles=CollectionFormats.kt," +
            "apiDocs=false,modelDocs=false,apiTests=false,modelTests=false",
    )
}
