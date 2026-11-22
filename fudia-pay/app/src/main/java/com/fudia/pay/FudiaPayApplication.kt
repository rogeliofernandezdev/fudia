package com.fudia.pay

import android.app.Application
import android.app.NotificationManager
import android.os.Build
import androidx.hilt.work.HiltWorkerFactory
import androidx.work.Configuration
import com.fudia.pay.core.security.DeviceCredentials
import com.fudia.pay.features.notifications.ListenerRepair
import com.fudia.pay.features.sync.PushSetup
import com.fudia.pay.features.sync.WakeAlarm
import dagger.hilt.android.HiltAndroidApp
import javax.inject.Inject

@HiltAndroidApp
class FudiaPayApplication : Application(), Configuration.Provider {
    @Inject lateinit var workerFactory: HiltWorkerFactory
    @Inject lateinit var credentials: DeviceCredentials
    override val workManagerConfiguration: Configuration
        get() = Configuration.Builder().setWorkerFactory(workerFactory).build()

    override fun onCreate() {
        super.onCreate()
        // Si el proceso murió a mitad de una reparación del listener, el
        // componente quedaría desactivado: se restaura al arrancar.
        ListenerRepair.ensureEnabled(this)
        // Push que despierta la app (si el APK se compiló con Firebase).
        PushSetup.initialize(this)
        if (credentials.paired) WakeAlarm.schedule(this)
        // El aviso fijo «Detectando pagos» (servicio en primer plano retirado el
        // 2026-11-21) no volverá: se borra su canal de los ajustes.
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            getSystemService(NotificationManager::class.java)?.deleteNotificationChannel(LEGACY_LISTENING_CHANNEL)
        }
    }

    private companion object {
        const val LEGACY_LISTENING_CHANNEL = "fudia-pay-listening"
    }
}
