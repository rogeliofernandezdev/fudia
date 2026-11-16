package com.fudia.pay.features.pairing

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.fudia.pay.core.network.FudiaPayApi
import com.fudia.pay.core.network.PairRequest
import com.fudia.pay.core.network.ScanRequest
import com.fudia.pay.core.security.DeviceCredentials
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.launch
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import retrofit2.HttpException
import javax.inject.Inject

/** Vinculación como WhatsApp Web: al leer el QR avisa el escaneo (Caja cambia
 * el QR por «Vinculando…») y luego canjea el código por la credencial del
 * dispositivo. El backend resuelve empresa y local; el token se guarda cifrado
 * y solo viaja una vez. */
@HiltViewModel
class PairingViewModel @Inject constructor(
    private val api: FudiaPayApi,
    private val credentials: DeviceCredentials,
) : ViewModel() {

    data class State(
        val pairing: Boolean = false,
        val error: String? = null,
        val done: Boolean = false,
    )

    private val _state = MutableStateFlow(State())
    val state: StateFlow<State> = _state

    fun pair(code: String) {
        val clean = code.trim()
        if (_state.value.pairing) return
        if (!Regex("^[a-f0-9]{32}$").matches(clean)) {
            _state.value = State(error = "El QR no contiene un código de vinculación válido.")
            return
        }
        _state.value = State(pairing = true)
        val model = android.os.Build.MODEL ?: ""
        viewModelScope.launch {
            // El aviso de escaneo es de mejor esfuerzo: un código usado, vencido
            // o inválido detiene aquí con el mensaje del backend; un corte de
            // red no impide el canje, que valida lo mismo.
            val scan = runCatching { api.scanPairing(ScanRequest(code = clean, model = model)) }.getOrNull()
            if (scan != null && scan.code() in 400..499) {
                _state.value = State(error = backendMessage(scan.errorBody()?.string()))
                return@launch
            }
            _state.value = try {
                val response = api.pair(PairRequest(code = clean, platform = "ANDROID", model = model))
                credentials.token = response.token
                credentials.organizationName = response.organizationName
                credentials.locationName = response.locationName
                State(done = true)
            } catch (error: HttpException) {
                State(error = backendMessage(error.response()?.errorBody()?.string()))
            } catch (error: Exception) {
                State(error = "No pudimos conectar con Fudia. Revisa la conexión e inténtalo de nuevo.")
            }
        }
    }

    /** El mensaje en español que devuelve el backend (`{"message": …}`), sin códigos HTTP. */
    private fun backendMessage(body: String?): String =
        runCatching { Json.parseToJsonElement(body.orEmpty()).jsonObject["message"]?.jsonPrimitive?.content }
            .getOrNull()
            ?.takeIf { it.isNotBlank() }
            ?: "No pudimos completar la vinculación. Genera otro QR en Caja e inténtalo de nuevo."
}
