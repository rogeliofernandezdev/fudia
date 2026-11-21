package com.fudia.pay

import android.app.Application
import androidx.hilt.work.HiltWorkerFactory
import androidx.work.Configuration
import com.fudia.pay.features.notifications.ListenerRepair
import dagger.hilt.android.HiltAndroidApp
import javax.inject.Inject

@HiltAndroidApp
class FudiaPayApplication : Application(), Configuration.Provider {
    @Inject lateinit var workerFactory: HiltWorkerFactory
    override val workManagerConfiguration: Configuration
        get() = Configuration.Builder().setWorkerFactory(workerFactory).build()

    override fun onCreate() {
        super.onCreate()
        // Si el proceso murió a mitad de una reparación del listener, el
        // componente quedaría desactivado: se restaura al arrancar.
        ListenerRepair.ensureEnabled(this)
    }
}
