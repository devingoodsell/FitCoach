package pro.d11l.fitcoach.core.auth

/** A persisted session: the access JWT and the opaque refresh token. */
data class Tokens(val accessToken: String, val refreshToken: String)

/**
 * Secure storage for session tokens. Abstracted so ViewModels/repositories can be
 * unit-tested with an in-memory fake; the production impl is Keystore-backed.
 */
interface TokenStorage {
    fun save(tokens: Tokens)
    fun load(): Tokens?
    fun clear()
}
