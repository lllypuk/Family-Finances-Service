import org.gradle.api.attributes.Bundling

// AGP 9 приносит собственный Kotlin, но более старой версии. Нужный KGP кладётся на classpath
// сборки здесь, а в модулях подключается `apply(plugin = "…")` без версии: версионированные
// alias'ы kotlin-плагинов в модулях конфликтуют с classpath-KGP.
buildscript {
    dependencies {
        classpath(libs.kotlin.gradle.plugin)
        classpath(libs.kotlin.serialization.plugin)
    }
}

plugins {
    alias(libs.plugins.android.application) apply false
    alias(libs.plugins.android.library) apply false
    alias(libs.plugins.kotlin.compose) apply false
}

// ktlint запускается своей конфигурацией, а не сторонним плагином: версия живёт в каталоге,
// в CI он приезжает вместе с проектом, и совместимость с AGP 9 ни от кого не зависит.
// ktlint-cli публикует обычный и shadow-вариант; без явного `Bundling` Gradle не выбирает
// между ними и падает на резолве. Нужен shadow: в обычном варианте clikt объявлен
// не-транзитивно, и запуск падает на ClassNotFoundException.
val ktlint = configurations.create("ktlint") {
    attributes {
        attribute(Bundling.BUNDLING_ATTRIBUTE, objects.named(Bundling::class.java, Bundling.SHADOWED))
    }
}

dependencies {
    ktlint(libs.ktlint.cli)
}

// Сгенерированный клиент лежит вне `src/`, поэтому под глоб не попадает и правилам не подчиняется.
val ktlintSources = listOf("**/src/**/*.kt", "**/*.gradle.kts")

tasks.register<JavaExec>("ktlintCheck") {
    group = "verification"
    description = "Проверить формат Kotlin"
    classpath = ktlint
    mainClass.set("com.pinterest.ktlint.Main")
    // ktlint обращается к внутренним пакетам JDK; без этого падает на JDK 17+.
    jvmArgs("--add-opens=java.base/java.lang=ALL-UNNAMED")
    args(ktlintSources)
}

tasks.register<JavaExec>("ktlintFormat") {
    group = "formatting"
    description = "Переформатировать Kotlin по ktlint"
    classpath = ktlint
    mainClass.set("com.pinterest.ktlint.Main")
    jvmArgs("--add-opens=java.base/java.lang=ALL-UNNAMED")
    args(listOf("-F") + ktlintSources)
}
