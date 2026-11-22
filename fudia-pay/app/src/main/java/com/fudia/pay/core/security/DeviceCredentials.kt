package com.fudia.pay.core.security

import android.content.Context
import androidx.security.crypto.EncryptedSharedPreferences
import androidx.security.crypto.MasterKey
import dagger.hilt.android.qualifiers.ApplicationContext
import javax.inject.Inject
import javax.inject.Singleton

/**
 * Credencial del dispositivo: se guarda cifrada con Keystore y nunca se
 * registra en logs. Solo el token y el nombre del restaurante vinculado.
 */
@Singleton
class DeviceCredentials @Inject constructor(@ApplicationContext context: Context) {
    private val prefs = EncryptedSharedPreferences.create(
        context,
        "fudia-pay-credentials",
        MasterKey.Builder(context).setKeyScheme(MasterKey.KeyScheme.AES256_GCM).build(),
        EncryptedSharedPreferences.PrefKeyEncryptionScheme.AES256_SIV,
        EncryptedSharedPreferences.PrefValueEncryptionScheme.AES256_GCM,
    )

    var token: String
        get() = prefs.getString(KEY_TOKEN, "") ?: ""
        set(value) = prefs.edit().putString(KEY_TOKEN, value).apply()

    var organizationName: String
        get() = prefs.getString(KEY_ORG, "") ?: ""
        set(value) = prefs.edit().putString(KEY_ORG, value).apply()

    var locationName: String
        get() = prefs.getString(KEY_LOCATION, "") ?: ""
        set(value) = prefs.edit().putString(KEY_LOCATION, value).apply()

    /** Último token de push que el backend aceptó; se borra al desvincular,
     *  así una vinculación nueva vuelve a registrarlo. */
    var uploadedPushToken: String
        get() = prefs.getString(KEY_PUSH, "") ?: ""
        set(value) = prefs.edit().putString(KEY_PUSH, value).apply()

    val paired: Boolean get() = token.isNotBlank()

    fun clear() = prefs.edit().clear().apply()

    private companion object {
        const val KEY_TOKEN = "device_token"
        const val KEY_ORG = "organization_name"
        const val KEY_LOCATION = "location_name"
        const val KEY_PUSH = "uploaded_push_token"
    }
}
