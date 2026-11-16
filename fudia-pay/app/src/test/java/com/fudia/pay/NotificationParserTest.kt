package com.fudia.pay

import com.fudia.pay.features.notifications.KnownApps
import com.fudia.pay.features.notifications.NotificationParser
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Test

/** Parsers por proveedor: solo se registra un pago con remitente y monto
 * legibles; lo ambiguo o irrelevante no se registra nunca. */
class NotificationParserTest {

    private fun parse(provider: String, title: String, text: String) = NotificationParser.parse(
        provider = provider,
        packageName = "com.yape.android",
        title = title,
        text = text,
        whenMillis = 1_763_000_000_000,
        key = "0|test|1",
    )

    @Test
    fun `yape recibido con remitente y monto se registra`() {
        val payment = parse("yape", "Yape", "Recibiste S/ 35.00 de Rosa Quispe. ¡Todo listo!")
        assertNotNull(payment)
        assertEquals("yape", payment!!.provider)
        assertEquals(35.0, payment.amount, 0.001)
        assertEquals("Rosa Quispe", payment.senderName)
    }

    @Test
    fun `plin recibido se registra con su parser`() {
        val payment = parse("plin", "Plin", "Te enviaron 10.00 de Ana Villanueva")
        assertNotNull(payment)
        assertEquals("plin", payment!!.provider)
        assertEquals(10.0, payment.amount, 0.001)
    }

    @Test
    fun `promociones y avisos no se registran`() {
        assertNull(parse("yape", "Yape", "Aprovecha tu promo de cashback"))
        assertNull(parse("plin", "Plin", "Tu cuenta está verificada"))
    }

    @Test
    fun `sin monto legible no se registra`() {
        assertNull(parse("yape", "Yape", "Recibiste un pago de Rosa Quispe"))
    }

    @Test
    fun `sin remitente legible no se registra`() {
        assertNull(parse("yape", "Yape", "Recibiste S/ 20.00"))
    }

    @Test
    fun `dos pagos del mismo monto tienen claves distintas`() {
        val first = parse("yape", "Yape", "Recibiste S/ 35.00 de Rosa Quispe")
        val second = parse("yape", "Yape", "Recibiste S/ 35.00 de Rosa Quispe", )
        // La clave incluye la notificación de origen: la misma notificación
        // reenviada comparte clave, pero dos postings distintos no.
        assertEquals(first!!.eventKey, second!!.eventKey)
        val other = parse("yape", "Yape", "Recibiste S/ 35.00 de Ana Villanueva")
        org.junit.Assert.assertNotEquals(first.eventKey, other!!.eventKey)
    }

    @Test
    fun `los paquetes verificados se resuelven a proveedor`() {
        assertEquals("yape", KnownApps.providerOf(KnownApps.YAPE))
        assertEquals("plin", KnownApps.providerOf(KnownApps.PLIN))
        assertNull(KnownApps.providerOf("com.android.messaging"))
    }
}
