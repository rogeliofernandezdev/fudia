package com.fudia.pay.core.network

import kotlinx.serialization.Serializable
import retrofit2.http.Body
import retrofit2.http.GET
import retrofit2.http.POST

/** Contrato con foods-backend (docs/16_FUDIA_PAY.md). La empresa y el local
 * los resuelve el backend desde la vinculación; el dispositivo nunca los envía. */

@Serializable
data class PairRequest(val code: String, val platform: String, val model: String)

/** Aviso de escaneo: Caja cambia el QR por «Vinculando…» antes del canje. */
@Serializable
data class ScanRequest(val code: String, val model: String)

@Serializable
data class PairDevice(val id: String, val name: String)

@Serializable
data class PairResponse(val device: PairDevice, val token: String, val organizationName: String, val locationName: String)

@Serializable
data class SessionOrganization(val name: String)

@Serializable
data class SessionResponse(val device: PairDevice, val restaurant: SessionOrganization, val location: SessionOrganization, val currency: String)

@Serializable
data class PaymentEvent(
    val provider: String,
    val eventKey: String,
    val senderName: String,
    val amount: Double,
    /** Época en milisegundos; el backend la convierte a UTC. */
    val occurredAt: Long,
)

@Serializable
data class ReportRequest(val events: List<PaymentEvent>)

@Serializable
data class EventOutcome(val eventKey: String, val status: String)

@Serializable
data class ReportResponse(val registered: Int, val duplicates: Int, val events: List<EventOutcome>)

interface FudiaPayApi {
    /** 204 sin cuerpo; 4xx si el código ya se usó, venció o no es válido. */
    @POST("v1/mobile/pair/scan")
    suspend fun scanPairing(@Body request: ScanRequest): retrofit2.Response<Unit>

    @POST("v1/mobile/pair")
    suspend fun pair(@Body request: PairRequest): PairResponse

    @GET("v1/mobile/session")
    suspend fun session(): SessionResponse

    @POST("v1/mobile/notifications")
    suspend fun report(@Body request: ReportRequest): ReportResponse
}
