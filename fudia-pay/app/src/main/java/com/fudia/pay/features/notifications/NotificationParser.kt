package com.fudia.pay.features.notifications

/**
 * Aplicaciones que la app escucha. Los nombres de paquete se definen aquí, en
 * código revisado, tras verificarlos en los APK reales de cada proveedor;
 * nunca se asume ni se amplía en tiempo de ejecución. Si una versión cambia de
 * paquete, se corrige aquí y se pruebe con una notificación real anonimizada.
 */
object KnownApps {
    const val YAPE = "com.yape.android"
    const val PLIN = "pe.plin.mobile"

    val all: Set<String> = setOf(YAPE, PLIN)

    fun providerOf(packageName: String): String? = when (packageName) {
        YAPE -> "yape"
        PLIN -> "plin"
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
    /** Montos en soles o dólar: "S/ 62.50", "62.50", "S/.35". */
    private val amountPattern = Regex("""(?:S/\.?\s*)?(\d{1,6}(?:[.,]\d{2}))""")

    fun parse(provider: String, packageName: String, title: String?, text: String?, whenMillis: Long, key: String): DetectedPayment? {
        val content = listOfNotNull(title, text).joinToString(" ").trim()
        if (content.isEmpty()) return null
        val confidence = when (provider) {
            "yape" -> YapeParser.confidence(content)
            "plin" -> PlinParser.confidence(content)
            else -> return null
        }
        if (!confidence.isPayment) return null
        val amount = amountPattern.findAll(content).lastOrNull()?.groupValues?.get(1)?.replace(",", ".")?.toDoubleOrNull() ?: return null
        if (amount <= 0.0) return null
        // Remitente sin puntuación de cierre del mensaje.
        val sender = confidence.sender?.trim()?.trimEnd('.', ',', '!', '¡')?.takeIf { it.isNotBlank() } ?: return null
        // Identidad estable: clave de la notificación + su texto. Dos pagos
        // legítimos con el mismo monto y hora tienen claves distintas.
        val eventKey = "$packageName:$key:${Integer.toHexString(content.hashCode())}"
        return DetectedPayment(provider, eventKey, sender, amount, whenMillis)
    }
}

/** Señales de confianza del proveedor; cada uno evoluciona por su cuenta. */
object YapeParser {
    private val received = Regex("""(?i)(recibiste|te envi[óoa]|yapeado a favor de|pago recibido)""")
    private val sender = Regex("""(?i)(?:de|por|te envi[óoa])\s+([A-ZÁÉÍÓÚÑ][\wÁÉÍÓÚÑ. ]{2,39})""")
    fun confidence(content: String): Confidence {
        val sender = sender.find(content)?.groupValues?.get(1)?.trim()
        return Confidence(received.containsMatchIn(content), sender)
    }
}

object PlinParser {
    private val received = Regex("""(?i)(recibiste|te envi[óoa]|plin recibido|pago recibido)""")
    private val sender = Regex("""(?i)(?:de|por|te envi[óoa])\s+([A-ZÁÉÍÓÚÑ][\wÁÉÍÓÚÑ. ]{2,39})""")
    fun confidence(content: String): Confidence {
        val sender = sender.find(content)?.groupValues?.get(1)?.trim()
        return Confidence(received.containsMatchIn(content), sender)
    }
}

data class Confidence(val isPayment: Boolean, val sender: String?)
