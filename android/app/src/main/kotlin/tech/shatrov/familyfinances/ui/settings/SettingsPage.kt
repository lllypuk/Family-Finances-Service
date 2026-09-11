package tech.shatrov.familyfinances.ui.settings

import java.util.UUID

private const val SLUG_ROOT = "root"
private const val SLUG_PROFILE = "profile"
private const val SLUG_PASSWORD = "password"
private const val SLUG_SESSIONS = "sessions"
private const val SLUG_USERS = "users"
private const val SLUG_USER_EDIT = "user-edit"
private const val SLUG_USER_PASSWORD = "user-password"
private const val SLUG_FAMILY = "family"
private const val SLUG_BACKUPS = "backups"

/**
 * Страница настроек. `visit` — идентичность захода: он же ключ модели, поэтому повторный заход
 * получает чистую модель, а данные перечитываются.
 */
sealed interface SettingsPage {
    val visit: UUID

    /** Слаг страницы: ключ в сохранённом экране и в ключе модели. */
    val slug: String

    /** Чья запись правится; у страниц без цели — `null`. */
    val target: UUID? get() = null

    val modelKey: String get() = "$slug-$visit"

    data class Root(override val visit: UUID = UUID.randomUUID()) : SettingsPage {
        override val slug: String get() = SLUG_ROOT
    }

    data class Profile(override val visit: UUID = UUID.randomUUID()) : SettingsPage {
        override val slug: String get() = SLUG_PROFILE
    }

    data class Password(override val visit: UUID = UUID.randomUUID()) : SettingsPage {
        override val slug: String get() = SLUG_PASSWORD
    }

    data class Sessions(override val visit: UUID = UUID.randomUUID()) : SettingsPage {
        override val slug: String get() = SLUG_SESSIONS
    }

    data class Users(override val visit: UUID = UUID.randomUUID()) : SettingsPage {
        override val slug: String get() = SLUG_USERS
    }

    /** Форма пользователя; `id` = `null` — новый. */
    data class UserEdit(
        val id: UUID?,
        override val visit: UUID = UUID.randomUUID(),
    ) : SettingsPage {
        override val slug: String get() = SLUG_USER_EDIT
        override val target: UUID? get() = id
    }

    data class UserPassword(
        val id: UUID,
        override val visit: UUID = UUID.randomUUID(),
    ) : SettingsPage {
        override val slug: String get() = SLUG_USER_PASSWORD
        override val target: UUID get() = id
    }

    data class Family(override val visit: UUID = UUID.randomUUID()) : SettingsPage {
        override val slug: String get() = SLUG_FAMILY
    }

    data class Backups(override val visit: UUID = UUID.randomUUID()) : SettingsPage {
        override val slug: String get() = SLUG_BACKUPS
    }
}

/** Часть ключа `AppScreenSaver` — фиксированные три поля, чтобы `restore` разбирал их как есть. */
internal fun SettingsPage.saveKey(): String = "$slug:${target ?: ""}:$visit"

/** Обратная сторона [saveKey]; `null` — ключ не от страницы настроек. */
internal fun restoreSettingsPage(
    slug: String,
    target: String,
    visit: String,
): SettingsPage? {
    val id = target.takeIf { it.isNotEmpty() }?.let(UUID::fromString)
    val at = UUID.fromString(visit)
    return when (slug) {
        SLUG_ROOT -> SettingsPage.Root(at)
        SLUG_PROFILE -> SettingsPage.Profile(at)
        SLUG_PASSWORD -> SettingsPage.Password(at)
        SLUG_SESSIONS -> SettingsPage.Sessions(at)
        SLUG_USERS -> SettingsPage.Users(at)
        SLUG_USER_EDIT -> SettingsPage.UserEdit(id, at)
        SLUG_USER_PASSWORD -> id?.let { SettingsPage.UserPassword(it, at) }
        SLUG_FAMILY -> SettingsPage.Family(at)
        SLUG_BACKUPS -> SettingsPage.Backups(at)
        else -> null
    }
}
