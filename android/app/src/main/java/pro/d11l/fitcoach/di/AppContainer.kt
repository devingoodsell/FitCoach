package pro.d11l.fitcoach.di

import android.content.Context
import pro.d11l.fitcoach.BuildConfig
import pro.d11l.fitcoach.core.auth.KeystoreTokenStorage
import pro.d11l.fitcoach.core.db.FitCoachDatabase
import pro.d11l.fitcoach.core.network.NetworkModule
import pro.d11l.fitcoach.data.AuthRepository
import pro.d11l.fitcoach.data.ConsentRepository
import pro.d11l.fitcoach.data.DietRepository
import pro.d11l.fitcoach.data.DisclaimerRepository
import pro.d11l.fitcoach.data.HealthSignalsRepository
import pro.d11l.fitcoach.data.InjuryRepository
import pro.d11l.fitcoach.data.LocationRepository
import pro.d11l.fitcoach.data.MemoryRepository
import pro.d11l.fitcoach.data.OnboardingRepository
import pro.d11l.fitcoach.data.ReadinessRepository
import pro.d11l.fitcoach.data.RoomMemoryCache
import pro.d11l.fitcoach.data.SessionRepository
import pro.d11l.fitcoach.healthconnect.HealthConnectSource

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
    val disclaimerRepository = DisclaimerRepository(api)
    val consentRepository = ConsentRepository(api)
    val locationRepository = LocationRepository(api)
    val dietRepository = DietRepository(api)
    val readinessRepository = ReadinessRepository(api)

    // Health Connect ingestion (E4-PR5): reads recovery signals on device and
    // uploads them; the backend computes readiness from them. Gated by E1 consent.
    private val recoverySignalSource = HealthConnectSource(context.applicationContext)
    val healthSignalsRepository = HealthSignalsRepository(recoverySignalSource, api, consentRepository)

    val injuryRepository = InjuryRepository(api)
    val sessionRepository = SessionRepository(api)
}
