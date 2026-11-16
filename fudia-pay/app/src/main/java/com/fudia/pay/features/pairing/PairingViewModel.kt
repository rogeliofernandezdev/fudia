package com.fudia.pay.features.pairing

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.fudia.pay.core.network.FudiaPayApi
import com.fudia.pay.core.network.PairRequest
import com.fudia.pay.core.security.DeviceCredentials
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.launch
import javax.inject.Inject

/** Vinculación: canjea el código del QR por la credencial del dispositivo. El
 * backend resuelve empresa y local; el token se guarda cifrado y solo viaja
 * una vez. */
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
        viewModelScope.launch {
            _state.value = try {
                val response = api.pair(PairRequest(code = clean, platform = "ANDROID", model = android.os.Build.MODEL ?: ""))
                credentials.token = response.token
                credentials.organizationName = response.organizationName
                credentials.locationName = response.locationName
                State(done = true)
            } catch (error: Exception) {
                State(error = error.message ?: "No pudimos completar la vinculación. Inténtalo de nuevo.")
            }
        }
    }
}
