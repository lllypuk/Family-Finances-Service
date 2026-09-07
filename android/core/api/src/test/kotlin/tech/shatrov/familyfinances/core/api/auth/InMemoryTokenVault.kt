package tech.shatrov.familyfinances.core.api.auth

/** Хранилище для тестов: Keystore в Robolectric не поднимается. */
class InMemoryTokenVault(private var stored: SessionToken? = null) : TokenVault {
    override fun read(): SessionToken? = stored

    override fun write(token: SessionToken) {
        stored = token
    }

    override fun clear() {
        stored = null
    }
}
