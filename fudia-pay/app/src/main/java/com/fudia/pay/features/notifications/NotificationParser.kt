package com.fudia.pay.features.notifications

/**
 * Aplicaciones que la app escucha. Los nombres de paquete se definen aquí, en
 * código revisado, tras verificarlos en los APK reales de cada proveedor;
 * nunca se asume ni se amplía en tiempo de ejecución. Si una versión cambia de
 * paquete, se corrige aquí y se pruebe con una notificación real anonimizada.
 */
object KnownApps {
    /** Paquetes verificados en los APK reales del teléfono del local:
     *  Yape (BCP Innovación; versiones antiguas usaron com.yape.android) y
     *  Plin, que este restaurante recibe notificado por la app de Interbank
     *  (verificado con una notificación real: «… te ha plineado S/ 1.00»).
     *  Nunca se asume ni se amplía en tiempo de ejecución. */
    const val YAPE = "com.bcp.innovacxion.yapeapp"
    const val YAPE_LEGACY = "com.yape.android"
    const val PLIN = "pe.plin.mobile"
    const val PLIN_INTERBANK = "pe.com.interbank.mobilebanking"

    val all: Set<String> = setOf(YAPE, YAPE_LEGACY, PLIN, PLIN_INTERBANK)

    fun providerOf(packageName: String): String? = when (packageName) {
        YAPE, YAPE_LEGACY -> "yape"
        PLIN, PLIN_INTERBANK -> "plin"
        else -> null
    }
}

/** Pago detectado con confianza: remitente y monto legibles. */
data class DetectedPayment(
    val provider: String,
    val eventKey: String,
    val senderName: String,
    val amount: Double,
    val occurredAtMillis: Long,
)

/**
 * Parser por proveedor. El texto de las notificaciones cambia entre versiones
 * de las apps: cada parser es independiente y solo registra el evento cuando
 * el contenido identifica con confianza un pago recibido (remitente y monto).
 * Si no alcanza esa confianza, devuelve null y el evento no se registra.
 */
object NotificationParser {
    /** Montos en soles o dólar: "S/ 62.50", "62.50", "S/.35" y el formato
     *  actual de Yape sin decimales ("S/ 1"). Con prefijo "S/" los decimales
     *  son opcionales; el número suelto exige dos decimales para no confundir
     *  el monto con otro número del mensaje. */
    private val currencyAmount = Regex("""S/\.?\s*(\d{1,6}(?:[.,]\d{1,2})?)""")
    private val plainAmount = Regex("""(\d{1,6}(?:[.,]\d{2}))""")

    private fun amountOf(content: String): Double? =
        (currencyAmount.find(content) ?: plainAmount.find(content))
            ?.groupValues?.get(1)?.replace(",", ".")?.toDoubleOrNull()

    fun parse(provider: String, packageName: String, title: String?, text: String?, whenMillis: Long, key: String): DetectedPayment? {
        val content = listOfNotNull(title, text).joinToString(" ").trim()
        if (content.isEmpty()) return null
        // El texto sin el título: el título suele ser el nombre de la app
        // ("Interbank", "Confirmación de Pago") y pegado al remitente lo
        // contaminaría ("Interbank Rogelio…").
        val body = text?.takeIf { it.isNotBlank() } ?: content
        val confidence = when (provider) {
            "yape" -> YapeParser.confidence(content, body)
            "plin" -> PlinParser.confidence(content, body)
            else -> return null
        }
        if (!confidence.isPayment) return null
        val amount = amountOf(body) ?: amountOf(content) ?: return null
        if (amount <= 0.0) return null
        // Remitente sin puntuación de cierre ni espacios dobles del mensaje.
        val sender = confidence.sender
            ?.trim()?.trimEnd('.', ',', '!', '¡')
            ?.replace(Regex("""\s+"""), " ")
            ?.takeIf { it.isNotBlank() } ?: return null
        // Identidad estable: clave de la notificación + su texto. Dos pagos
        // legítimos con el mismo monto y hora tienen claves distintas.
        val eventKey = "$packageName:$key:${Integer.toHexString(content.hashCode())}"
        return DetectedPayment(provider, eventKey, sender, amount, whenMillis)
    }
}

/** Señales de confianza del proveedor; cada uno evoluciona por su cuenta. */
object YapeParser {
    private val received = Regex("""(?i)(recibiste|te envi[óoa]|yapeado a favor de|pago recibido)""")
    // Formato actual: "Yape! {Nombre} te envió un pago por S/ 1" — el nombre
    // va antes de "te envió". Preferente sobre el formato previo para no
    // confundir el remitente con palabras del título ("Confirmación de Pago").
    private val senderBefore = Regex("""(?i)([A-ZÁÉÍÓÚÑ][\wÁÉÍÓÚÑ. ]{2,39}?)\s+te envi[óoa]""")
    // Formatos previos: "Recibiste S/ 35.00 de Rosa Quispe".
    private val senderAfter = Regex("""(?i)(?:de|por|te envi[óoa])\s+([A-ZÁÉÍÓÚÑ][\wÁÉÍÓÚÑ. ]{2,39})""")
    fun confidence(content: String, body: String): Confidence {
        val sender = senderBefore.find(body)?.groupValues?.get(1)?.trim()
            ?: senderAfter.find(body)?.groupValues?.get(1)?.trim()
            ?: senderBefore.find(content)?.groupValues?.get(1)?.trim()
            ?: senderAfter.find(content)?.groupValues?.get(1)?.trim()
        return Confidence(received.containsMatchIn(content), sender)
    }
}

object PlinParser {
    private val received = Regex("""(?i)(recibiste|te envi[óoa]|te ha plinea|te plinearon|plin recibido|pago recibido)""")
    // Formato Interbank actual: "Rogelio Fernandez Aybar te ha plineado S/ 1.00"
    // — el nombre va antes del verbo.
    private val senderBefore = Regex("""(?i)([A-ZÁÉÍÓÚÑ][\wÁÉÍÓÚÑ. ]{2,39}?)\s+te (?:ha plinea|plinearon)""")
    // Formatos previos: "Te enviaron 10.00 de Ana Villanueva".
    private val senderAfter = Regex("""(?i)(?:de|por|te envi[óoa])\s+([A-ZÁÉÍÓÚÑ][\wÁÉÍÓÚÑ. ]{2,39})""")
    fun confidence(content: String, body: String): Confidence {
        val sender = senderBefore.find(body)?.groupValues?.get(1)?.trim()
            ?: senderAfter.find(body)?.groupValues?.get(1)?.trim()
            ?: senderBefore.find(content)?.groupValues?.get(1)?.trim()
            ?: senderAfter.find(content)?.groupValues?.get(1)?.trim()
        return Confidence(received.containsMatchIn(content), sender)
    }
}

data class Confidence(val isPayment: Boolean, val sender: String?)
