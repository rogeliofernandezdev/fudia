package com.fudia.pay

import com.fudia.pay.features.notifications.KnownApps
import com.fudia.pay.features.notifications.NotificationParser
import com.fudia.pay.features.sync.isoUtc
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Test

/** Parsers por proveedor: solo se registra un pago con remitente y monto
 * legibles; lo ambiguo o irrelevante no se registra nunca. */
class NotificationParserTest {

    @Test
    fun `yape conserva el numero de operacion y sus ceros iniciales`() {
        val payment = parse("yape", "Yape", "Recibiste S/ 35.00 de Rosa Quispe\nNúmero de operación: 0012345678")!!
        assertEquals("0012345678", payment.operationNumber)
        assertEquals("Rosa Quispe", payment.senderName)
        assertEquals(35.0, payment.amount, 0.001)
    }

    @Test
    fun `plin conserva la referencia explicita del banco`() {
        val payment = parse("plin", "Interbank", "Ana Villanueva te ha plineado S/ 10.00\nN° de operación: IBK-001234")!!
        assertEquals("IBK-001234", payment.operationNumber)
        assertEquals("Ana Villanueva", payment.senderName)
    }

    @Test
    fun `no inventa numero de operacion desde monto seguridad o clave tecnica`() {
        assertNull(parse("yape", "Yape", "Maria Torres te envió un pago por S/ 62.50")!!.operationNumber)
        assertNull(parse("yape", "Yape", "Maria Torres te envió un pago por S/ 62.50\nCódigo de seguridad: 123456")!!.operationNumber)
        assertNull(parse("plin", "Plin", "Ana Villanueva te ha plineado S/ 10.00")!!.operationNumber)
    }

    @Test
    fun `referencias contradictorias no se presentan como una operacion cierta`() {
        val payment = parse("yape", "Yape", "Maria Torres te envió un pago por S/ 10\nOperación: 001234\nOperación: 005678")!!
        assertNull(payment.operationNumber)
    }

    @Test
    fun `la referencia antes del monto no se interpreta como importe`() {
        val payment = parse("yape", "Yape", "Nro. de operación: 0012345678\nRecibiste S/ 35.00 de Rosa Quispe")!!
        assertEquals(35.0, payment.amount, 0.001)
        assertEquals("0012345678", payment.operationNumber)
    }

    @Test
    fun `el texto expandido aporta operacion sin cambiar la clave idempotente`() {
        val summary = "Maria Torres te envió un pago por S/ 10"
        val original = NotificationParser.parse("yape", KnownApps.YAPE, "Yape", summary, 1000, "stable-key")!!
        val expanded = NotificationParser.parse("yape", KnownApps.YAPE, "Yape", summary, 1000, "stable-key", summary + "\nOperación: 0012345678")!!
        assertEquals(original.eventKey, expanded.eventKey)
        assertEquals("0012345678", expanded.operationNumber)
        assertEquals(original.amount, expanded.amount, 0.001)
    }

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
    fun `yape formato actual con monto sin decimales se registra`() {
        val payment = parse("yape", "Confirmación de Pago", "Yape! Rogelio  Fernandez Aybar te envió un pago por S/ 1")
        assertNotNull(payment)
        assertEquals("yape", payment!!.provider)
        assertEquals(1.0, payment.amount, 0.001)
        assertEquals("Rogelio Fernandez Aybar", payment.senderName)
    }

    @Test
    fun `yape formato actual con monto con decimales se registra`() {
        val payment = parse("yape", "Confirmación de Pago", "Yape! Maria Torres te envió un pago por S/ 62.50")
        assertNotNull(payment)
        assertEquals(62.5, payment!!.amount, 0.001)
        assertEquals("Maria Torres", payment.senderName)
    }

    @Test
    fun `plin formato interbank con nombre antes del verbo se registra`() {
        val payment = parse("plin", "Interbank", "Rogelio Fernandez Aybar te ha plineado S/ 1.00")
        assertNotNull(payment)
        assertEquals("plin", payment!!.provider)
        assertEquals(1.0, payment.amount, 0.001)
        assertEquals("Rogelio Fernandez Aybar", payment.senderName)
    }

    @Test
    fun `el titulo de la app no contamina el remitente`() {
        // Sin el texto, el título es la única fuente: si algún formato pusiera
        // el remitente en el título, se captura; pero con texto presente el
        // nombre no arrastra el título pegado.
        val payment = NotificationParser.parse(
            provider = "plin",
            packageName = "pe.com.interbank.mobilebanking",
            title = "Interbank",
            text = "Rogelio Fernandez Aybar te ha plineado S/ 1.00",
            whenMillis = 1_763_000_000_000,
            key = "0|real|2",
        )
        assertNotNull(payment)
        assertEquals("Rogelio Fernandez Aybar", payment!!.senderName)
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
    fun `la marca de tiempo se envia como fecha iso utc`() {
        assertEquals("1970-01-01T00:00:00Z", isoUtc(0L))
        assertEquals("1970-01-02T00:00:00Z", isoUtc(86_400_000L))
        assertEquals("2026-11-18T20:46:00Z", isoUtc(1_795_034_760_000L))
    }

    @Test
    fun `los paquetes verificados se resuelven a proveedor`() {
        assertEquals("yape", KnownApps.providerOf(KnownApps.YAPE))
        assertEquals("yape", KnownApps.providerOf(KnownApps.YAPE_LEGACY))
        assertEquals("yape", KnownApps.providerOf("com.bcp.innovacxion.yapeapp"))
        assertEquals("plin", KnownApps.providerOf(KnownApps.PLIN))
        assertEquals("plin", KnownApps.providerOf(KnownApps.PLIN_INTERBANK))
        assertEquals("plin", KnownApps.providerOf("pe.com.interbank.mobilebanking"))
        assertNull(KnownApps.providerOf("com.android.messaging"))
    }

    @Test
    fun `paquete real de yape en el telefono del local se acepta`() {
        // El APK real de Yape (BCP Innovación) publica la notificación desde
        // su paquete propio; el evento se registra igual que con el paquete
        // histórico.
        val payment = NotificationParser.parse(
            provider = KnownApps.providerOf("com.bcp.innovacxion.yapeapp")!!,
            packageName = "com.bcp.innovacxion.yapeapp",
            title = "Confirmación de Pago",
            text = "Yape! Rogelio  Fernandez Aybar te envió un pago por S/ 1",
            whenMillis = 1_763_000_000_000,
            key = "0|real|1",
        )
        assertNotNull(payment)
        assertEquals(1.0, payment!!.amount, 0.001)
        assertEquals("Rogelio Fernandez Aybar", payment.senderName)
    }
}
