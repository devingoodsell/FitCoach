package pro.d11l.fitcoach.di

import android.content.Context
import pro.d11l.fitcoach.BuildConfig
import pro.d11l.fitcoach.core.auth.KeystoreTokenStorage
import pro.d11l.fitcoach.core.db.FitCoachDatabase
import pro.d11l.fitcoach.core.network.NetworkModule
import pro.d11l.fitcoach.data.AuthRepository
import pro.d11l.fitcoach.data.MemoryRepository
import pro.d11l.fitcoach.data.OnboardingRepository
import pro.d11l.fitcoach.data.RoomMemoryCache

/**
 * Manual dependency container (lightweight DI). Holds singletons for the process
 * lifetime; created in [pro.d11l.fitcoach.FitCoachApp].
 */
class AppContainer(context: Context) {
    private val tokenStorage = KeystoreTokenStorage(context.applicationContext)
    private val api = NetworkModule.create(BuildConfig.BACKEND_BASE_URL, tokenStorage)
    private val db = FitCoachDatabase.create(context.applicationContext)
    private val memoryCache = RoomMemoryCache(db.memorySectionDao())

    val authRepository = AuthRepository(api, tokenStorage, memoryCache)
    val memoryRepository = MemoryRepository(api, memoryCache)
    val onboardingRepository = OnboardingRepository(api)
}
