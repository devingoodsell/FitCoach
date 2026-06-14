package pro.d11l.fitcoach

import android.app.Application
import pro.d11l.fitcoach.di.AppContainer

/** Application entry point; owns the process-wide dependency container. */
class FitCoachApp : Application() {
    lateinit var container: AppContainer
        private set

    override fun onCreate() {
        super.onCreate()
        container = AppContainer(this)
    }
}
