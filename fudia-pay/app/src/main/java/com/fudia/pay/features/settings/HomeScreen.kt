package com.fudia.pay.features.settings

import android.content.Context
import android.content.Intent
import android.provider.Settings
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.navigationBars
import androidx.compose.foundation.layout.windowInsetsPadding
import androidx.compose.foundation.layout.defaultMinSize
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalLifecycleOwner
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.LifecycleEventObserver
import androidx.compose.ui.unit.dp
import androidx.core.app.NotificationManagerCompat
import com.fudia.pay.core.design.FudiaGreenDark
import com.fudia.pay.core.design.FudiaHeader
import com.fudia.pay.core.design.FudiaPrimary
import com.fudia.pay.core.security.DeviceCredentials
import com.fudia.pay.features.notifications.ListenerHealth
import com.fudia.pay.features.payments.PendingEventDao
import com.fudia.pay.features.sync.SyncScheduler

/** Texto sincero sobre la vida del servicio: conectado o desconectado y hace
 *  cuánto, lo que el sistema reporta — no solo el permiso. */
internal fun livenessText(health: ListenerHealth, now: Long = System.currentTimeMillis()): String = when {
    health.connected -> "Servicio de escucha conectado ${sinceLabel(health.connectedAtMillis, now)}."
    health.disconnectedAtMillis == 0L -> "El servicio de escucha aún no se ha conectado."
    else -> "Servicio desconectado ${sinceLabel(health.disconnectedAtMillis, now)}; Android lo reconectará al usarse."
}

internal fun sinceLabel(atMillis: Long, now: Long): String {
    val minutes = ((now - atMillis) / 60_000).coerceAtLeast(0)
    return when {
        minutes == 0L -> "ahora mismo"
        minutes == 1L -> "hace 1 minuto"
        minutes < 60L -> "hace $minutes minutos"
        minutes < 1440L -> "hace ${minutes / 60} h"
        else -> "hace ${minutes / 1440} d"
    }
}

/** El permiso de escucha está concedido y la exclusión de batería otorgada:
 *  dos lecturas del sistema que cambian en Ajustes, no dentro de la app. */
internal fun listenerGranted(context: Context): Boolean =
    NotificationManagerCompat.getEnabledListenerPackages(context).contains(context.packageName)

internal fun batteryExempt(context: Context): Boolean =
    context.getSystemService(android.os.PowerManager::class.java)
        ?.isIgnoringBatteryOptimizations(context.packageName) ?: false

/** Inicio: el estado que importa en una mirada — detección, restaurante y
 * pendientes — y las dos acciones reales (activar permiso, desvincular).
 * Grupos apretados, secciones separadas, 48 dp de área táctil. */
@Composable
fun HomeScreen(
    credentials: DeviceCredentials,
    listenerHealth: ListenerHealth,
    dao: PendingEventDao,
    scheduler: SyncScheduler,
    onUnlink: () -> Unit,
) {
    val context = LocalContext.current
    val count by dao.countFlow().collectAsState(initial = 0)
    // Ambos estados los concede Ajustes del sistema: se recalculan al volver
    // de ahí (ON_RESUME), no solo al abrir la app, o el botón quedaría visible
    // después de concederlo.
    var enabled by remember { mutableStateOf(listenerGranted(context)) }
    var batteryExempt by remember { mutableStateOf(batteryExempt(context)) }
    val lifecycleOwner = LocalLifecycleOwner.current
    DisposableEffect(lifecycleOwner) {
        val observer = LifecycleEventObserver { _, event ->
            if (event == Lifecycle.Event.ON_RESUME) {
                enabled = listenerGranted(context)
                batteryExempt = batteryExempt(context)
            }
        }
        lifecycleOwner.lifecycle.addObserver(observer)
        onDispose { lifecycleOwner.lifecycle.removeObserver(observer) }
    }

    Column(modifier = Modifier.fillMaxSize()) {
        // Cabecera compartida con la bienvenida (marca, nombre y descripción).
        FudiaHeader()

        Column(
            modifier = Modifier
                .fillMaxSize()
                .windowInsetsPadding(WindowInsets.navigationBars)
                .padding(16.dp)
                .verticalScroll(rememberScrollState()),
            verticalArrangement = Arrangement.spacedBy(16.dp),
        ) {
            // Estado protagonista: una sola mirada dice si la app vive.
            Card(colors = CardDefaults.cardColors(containerColor = Color.White)) {
                Column(modifier = Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    Text("Detección de pagos", style = MaterialTheme.typography.labelMedium, color = Color(0xFF39507A))
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Surface(color = if (enabled) Color(0xFFE3F8E9) else Color(0xFFFFF0D8), shape = MaterialTheme.shapes.small) {
                            Text(
                                if (enabled) "Activa" else "Inactiva",
                                color = if (enabled) FudiaGreenDark else Color(0xFF9D5700),
                                style = MaterialTheme.typography.labelLarge,
                                modifier = Modifier.padding(horizontal = 10.dp, vertical = 4.dp),
                            )
                        }
                    }
                    Text(
                        if (enabled) "Android entrega las notificaciones de Yape y Plin; cada pago detectado avisa a tu caja."
                        else "Concede el acceso a notificaciones para empezar. La app no puede leerlas sin tu permiso.",
                        style = MaterialTheme.typography.bodyMedium,
                        color = Color(0xFF39507A),
                    )
                    if (!enabled) {
                        Button(
                            modifier = Modifier.fillMaxWidth().defaultMinSize(minHeight = 48.dp),
                            colors = ButtonDefaults.buttonColors(containerColor = FudiaPrimary),
                            onClick = { context.startActivity(Intent(Settings.ACTION_NOTIFICATION_LISTENER_SETTINGS)) },
                        ) { Text("Activar detección de pagos") }
                    }
                    // Salud real del servicio: el permiso puede estar concedido
                    // con el servicio muerto por la batería del fabricante.
                    Text(
                        livenessText(listenerHealth),
                        style = MaterialTheme.typography.bodySmall,
                        color = Color(0xFF39507A),
                    )
                    if (!batteryExempt) {
                        OutlinedButton(
                            modifier = Modifier.fillMaxWidth().defaultMinSize(minHeight = 48.dp),
                            onClick = {
                                runCatching {
                                    context.startActivity(
                                        Intent(
                                            android.provider.Settings.ACTION_REQUEST_IGNORE_BATTERY_OPTIMIZATIONS,
                                            android.net.Uri.parse("package:${context.packageName}"),
                                        )
                                    )
                                }
                            },
                        ) { Text("Excluir de optimización de batería") }
                    }
                }
            }

            // Restaurante vinculado: etiqueta 12 px, valor protagonista.
            Card(colors = CardDefaults.cardColors(containerColor = Color.White)) {
                Column(modifier = Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
                    Text("Restaurante", style = MaterialTheme.typography.labelMedium, color = Color(0xFF39507A))
                    Column {
                        Text(credentials.organizationName.ifBlank { "Sin vincular" }, style = MaterialTheme.typography.titleMedium)
                        if (credentials.locationName.isNotBlank()) {
                            Text(credentials.locationName, style = MaterialTheme.typography.bodySmall, color = Color(0xFF39507A))
                        }
                    }
                    Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                        Text("Pagos por sincronizar", style = MaterialTheme.typography.bodyMedium, color = Color(0xFF39507A))
                        Text("$count", style = MaterialTheme.typography.titleMedium)
                    }
                    if (count > 0) {
                        OutlinedButton(
                            modifier = Modifier.fillMaxWidth().defaultMinSize(minHeight = 48.dp),
                            onClick = { scheduler.enqueueNow() },
                        ) { Text("Sincronizar ahora") }
                    }
                }
            }

            Spacer(modifier = Modifier.height(4.dp))
            // Salida al final, en tono secundario: es una acción infrecuente y
            // irreversible desde Caja; aquí solo limpia este teléfono.
            TextButton(
                modifier = Modifier.fillMaxWidth().defaultMinSize(minHeight = 48.dp),
                onClick = onUnlink,
            ) { Text("Desvincular este teléfono") }
            Text(
                "La desvinculación remota se hace desde Caja. Este aviso es una notificación detectada, no una confirmación bancaria.",
                style = MaterialTheme.typography.bodySmall,
                color = Color(0xFF39507A),
                modifier = Modifier.padding(bottom = 16.dp),
            )
        }
    }
}
