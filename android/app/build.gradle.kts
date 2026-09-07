import org.jetbrains.kotlin.gradle.dsl.JvmTarget
import java.io.File

// Плагина `org.jetbrains.kotlin.android` здесь нет намеренно: AGP 9 несёт Kotlin встроенным и
// на попытку подключить его отдельно отвечает отказом.
plugins {
    alias(libs.plugins.android.application)
    alias(libs.plugins.kotlin.compose)
}

// Подпись: путь и пароль приходят снаружи — с ноутбука через Makefile, в CI из защищённых
// переменных. Файл в дерево репозитория не кладётся никогда.
val keystorePath: String? = System.getenv("FFS_KEYSTORE_PATH")
val keystorePassword: String? = System.getenv("FFS_KEYSTORE_PASSWORD")

android {
    namespace = "tech.shatrov.familyfinances"
    compileSdk = libs.versions.compileSdk.get().toInt()

    defaultConfig {
        applicationId = "tech.shatrov.familyfinances"
        minSdk = libs.versions.minSdk.get().toInt()
        targetSdk = libs.versions.targetSdk.get().toInt()
        versionCode = libs.versions.appVersionCode.get().toInt()
        versionName = libs.versions.appVersionName.get()
    }

    // Конфигурация заводится только когда ключ есть: пустой storeFile роняет даже те задачи,
    // которым подпись не нужна. Отсутствие ключа при сборке релиза ловит проверка ниже.
    val installerSigning = if (!keystorePath.isNullOrBlank()) {
        signingConfigs.create("installer") {
            storeFile = file(keystorePath)
            storePassword = keystorePassword
            keyAlias = "installer"
            keyPassword = keystorePassword
        }
    } else {
        null
    }

    buildTypes {
        release {
            isMinifyEnabled = true
            isShrinkResources = true
            proguardFiles(getDefaultProguardFile("proguard-android-optimize.txt"), "proguard-rules.pro")
            // Тот же сертификат, что и у предыдущей установки: другой даёт
            // INSTALL_FAILED_UPDATE_INCOMPATIBLE и требует снести приложение вместе с данными.
            // На телефоны ставится именно релизная сборка, поэтому ключ нужен только ей —
            // отладочная подписывается автоматическим ключом AGP и живёт на эмуляторе.
            signingConfig = installerSigning
        }
    }

    buildFeatures {
        compose = true
        // Базовый адрес сервиса приходит отсюда: `BuildConfig` у каждого модуля свой,
        // и читать его из `:core:api` нельзя.
        buildConfig = true
    }

    defaultConfig {
        buildConfigField("String", "API_BASE_URL", "\"https://ffs.shatrov.tech/\"")
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

    testOptions {
        // Robolectric-тесты живут в `src/test`, чтобы попадать в CI: инструментальные там не
        // гоняются, для них нужен /dev/kvm. Ресурсы Compose без этого флага им недоступны.
        unitTests.isIncludeAndroidResources = true
    }
}

// Подпись проверяется только когда собирается релиз: иначе отсутствие ключа ломало бы `check`.
tasks.matching { it.name == "assembleRelease" }.configureEach {
    // Путь снимается в локальную переменную до `doFirst`: действие задачи попадает в кеш
    // конфигурации, а ссылку на скрипт сборки туда не сериализовать.
    val path = keystorePath
    doFirst {
        require(!path.isNullOrBlank() && File(path).exists()) {
            "Нет keystore для подписи. Задайте FFS_KEYSTORE_PATH и FFS_KEYSTORE_PASSWORD " +
                "(на ноутбуке — через make, в CI — защищённые переменные)."
        }
    }
}

dependencies {
    implementation(project(":core:api"))
    implementation(platform(libs.compose.bom))
    implementation(libs.compose.ui)
    implementation(libs.compose.ui.graphics)
    implementation(libs.compose.ui.tooling.preview)
    implementation(libs.compose.material3)
    implementation(libs.androidx.core.ktx)
    implementation(libs.androidx.activity.compose)
    implementation(libs.androidx.lifecycle.runtime.ktx)
    implementation(libs.androidx.lifecycle.viewmodel.compose)
    implementation(libs.androidx.lifecycle.runtime.compose)
    implementation(libs.kotlinx.coroutines.android)

    debugImplementation(libs.compose.ui.tooling)
    debugImplementation(libs.compose.ui.test.manifest)

    testImplementation(libs.junit)
    testImplementation(libs.robolectric)
    testImplementation(libs.androidx.test.core)
    testImplementation(libs.androidx.test.junit)
    testImplementation(libs.kotlinx.coroutines.test)
    testImplementation(platform(libs.compose.bom))
    testImplementation(libs.compose.ui.test.junit4)
    testImplementation(libs.okhttp.mockwebserver)
}
