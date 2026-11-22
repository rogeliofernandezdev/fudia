package com.fudia.pay.features.sync

import android.content.Context
import android.util.Log
import com.fudia.pay.BuildConfig
import com.fudia.pay.core.network.FudiaPayApi
import com.fudia.pay.core.network.PushTokenRequest
import com.fudia.pay.core.security.DeviceCredentials
import com.google.android.gms.tasks.Task
import com.google.firebase.FirebaseApp
import com.google.firebase.FirebaseOptions
import com.google.firebase.messaging.FirebaseMessaging
import dagger.hilt.android.qualifiers.ApplicationContext
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.suspendCancellableCoroutine
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock
import javax.inject.Inject
import javax.inject.Singleton
import kotlin.coroutines.resume
import kotlin.coroutines.resumeWithException

/** ¿Hay que (re)registrar el token? Solo vinculado, con token y si cambió. */
internal fun shouldUploadPushToken(paired: Boolean, current: String, uploaded: String): Boolean =
    paired && current.isNotBlank() && current != uploaded

/**
 * Inicialización de Firebase sin google-services: las opciones llegan por
 * BuildConfig al compilar (ver build.gradle.kts). Solo se usa la mensajería,
 * para el push silencioso que despierta la app (ver [WakeMessagingService]).
 */
object PushSetup {
    val configured: Boolean
        get() = BuildConfig.FCM_APP_ID.isNotBlank() && BuildConfig.FCM_API_KEY.isNotBlank() &&
            BuildConfig.FCM_PROJECT_ID.isNotBlank() && BuildConfig.FCM_SENDER_ID.isNotBlank()

    fun initialize(context: Context): Boolean {
        if (!configured) return false
        if (FirebaseApp.getApps(context).isEmpty()) {
            FirebaseApp.initializeApp(
                context,
                FirebaseOptions.Builder()
                    .setApplicationId(BuildConfig.FCM_APP_ID)
                    .setApiKey(BuildConfig.FCM_API_KEY)
                    .setProjectId(BuildConfig.FCM_PROJECT_ID)
                    .setGcmSenderId(BuildConfig.FCM_SENDER_ID)
                    .build(),
            )
        }
        return true
    }
}

/**
 * Registra en el backend el token de push del teléfono. El token no es una
 * credencial: solo permite al backend pedir «procesa tu bandeja». Se reintenta
 * en cada barrido hasta que el backend lo acepta.
 */
@Singleton
class PushTokens @Inject constructor(
    @ApplicationContext private val context: Context,
    private val api: FudiaPayApi,
    private val credentials: DeviceCredentials,
) {
    private val mutex = Mutex()

    /** Lee el token actual (o usa el recibido en onNewToken) y lo sube si cambió. */
    suspend fun sync(fresh: String? = null) = mutex.withLock {
        if (!credentials.paired || !PushSetup.initialize(context)) return@withLock
        try {
            val token = fresh ?: FirebaseMessaging.getInstance().token.await()
            if (!shouldUploadPushToken(credentials.paired, token, credentials.uploadedPushToken)) return@withLock
            if (api.registerPush(PushTokenRequest(token)).isSuccessful) {
                credentials.uploadedPushToken = token
                if (BuildConfig.DEBUG) Log.d(TAG, "token de push registrado")
            }
        } catch (error: CancellationException) {
            throw error
        } catch (error: Exception) {
            // Sin Google Play Services o sin red: queda la alarma de respaldo.
            if (BuildConfig.DEBUG) Log.d(TAG, "token de push pendiente: ${error.javaClass.simpleName}")
        }
    }

    /** Al desvincular: el backend deja de despertar este teléfono (mejor esfuerzo). */
    suspend fun forget() {
        if (!credentials.paired) return
        try {
            api.registerPush(PushTokenRequest(""))
        } catch (error: CancellationException) {
            throw error
        } catch (_: Exception) {
            // Revocar desde Caja también borra el token.
        }
    }

    private companion object {
        const val TAG = "FudiaPay"
    }
}

private suspend fun <T> Task<T>.await(): T = suspendCancellableCoroutine { continuation ->
    addOnCompleteListener { task ->
        val error = task.exception
        if (task.isSuccessful) continuation.resume(task.result)
        else continuation.resumeWithException(error ?: IllegalStateException("tarea de Firebase sin resultado"))
    }
}
