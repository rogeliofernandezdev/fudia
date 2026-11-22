package com.fudia.pay.features.settings

import android.content.Context
import android.content.Intent
import android.net.Uri
import android.provider.Settings
import androidx.annotation.DrawableRes
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.defaultMinSize
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.navigationBars
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.windowInsetsPadding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
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
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalLifecycleOwner
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.core.app.NotificationManagerCompat
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.LifecycleEventObserver
import com.fudia.pay.R
import com.fudia.pay.core.design.FudiaCloudLine
import com.fudia.pay.core.design.FudiaGreenDark
import com.fudia.pay.core.design.FudiaHeader
import com.fudia.pay.core.design.FudiaPrimary
import com.fudia.pay.core.design.FudiaPrimaryDark
import com.fudia.pay.core.security.DeviceCredentials
import com.fudia.pay.features.notifications.ListenerHealth
import com.fudia.pay.features.payments.PendingEventDao
import com.fudia.pay.features.sync.SyncScheduler

/** Estado de la detección tal como lo ve quien usa el teléfono. */
internal enum class DetectionState { Off, Connecting, Active }

/** Sin permiso no hay nada que escuchar; con permiso, la verdad la dice el
 *  propio servicio: si no está conectado, la app lo admite («Conectando»). */
internal fun detectionState(granted: Boolean, health: ListenerHealth): DetectionState = when {
    !granted -> DetectionState.Off
    !health.connected -> DetectionState.Connecting
    else -> DetectionState.Active
}

/** El permiso de escucha está concedido y la exclusión de batería otorgada:
 *  dos lecturas del sistema que cambian en Ajustes, no dentro de la app. */
internal fun listenerGranted(context: Context): Boolean =
    NotificationManagerCompat.getEnabledListenerPackages(context).contains(context.packageName)

internal fun batteryExempt(context: Context): Boolean =
    context.getSystemService(android.os.PowerManager::class.java)
        ?.isIgnoringBatteryOptimizations(context.packageName) ?: false

private val InkSoft = Color(0xFF39507A)
private val GreenTint = Color(0xFFE3F8E9)
private val AmberTint = Color(0xFFFFF0D8)
private val AmberText = Color(0xFF9D5700)
private val BlueTint = Color(0xFFE7EEFF)

/** Tono de una fila: fondo del cuadro del icono y color del icono. */
private data class RowTone(val tint: Color, val ink: Color)

private val ToneOk = RowTone(GreenTint, FudiaGreenDark)
private val ToneWarn = RowTone(AmberTint, AmberText)
private val ToneInfo = RowTone(BlueTint, FudiaPrimary)

/**
 * Inicio como lista de controles: una fila por punto que importa —detección,
 * batería (solo si falta el permiso), restaurante y pagos sin enviar— con su
 * valor a la derecha. Sin textos de apoyo: el color aparece solo donde hay
 * algo que hacer, y esa fila entera es la acción (48 dp de alto mínimo).
 */
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
    // de ahí (ON_RESUME), no solo al abrir la app, o la fila seguiría pidiendo
    // algo ya concedido.
    var granted by remember { mutableStateOf(listenerGranted(context)) }
    var exempt by remember { mutableStateOf(batteryExempt(context)) }
    val lifecycleOwner = LocalLifecycleOwner.current
    DisposableEffect(lifecycleOwner) {
        val observer = LifecycleEventObserver { _, event ->
            if (event == Lifecycle.Event.ON_RESUME) {
                granted = listenerGranted(context)
                exempt = batteryExempt(context)
            }
        }
        lifecycleOwner.lifecycle.addObserver(observer)
        onDispose { lifecycleOwner.lifecycle.removeObserver(observer) }
    }
    val detection = detectionState(granted, listenerHealth)

    Column(modifier = Modifier.fillMaxSize().windowInsetsPadding(WindowInsets.navigationBars)) {
        FudiaHeader(description = null)

        Column(
            modifier = Modifier
                .weight(1f)
                .fillMaxWidth()
                .verticalScroll(rememberScrollState())
                .padding(16.dp),
        ) {
            Card(colors = CardDefaults.cardColors(containerColor = Color.White)) {
                when (detection) {
                    DetectionState.Off -> ControlRow(
                        icon = R.drawable.ic_bell,
                        tone = ToneWarn,
                        title = "Detección de pagos",
                        value = "Activar",
                        valueColor = FudiaPrimaryDark,
                        onClick = { context.startActivity(Intent(Settings.ACTION_NOTIFICATION_LISTENER_SETTINGS)) },
                    )
                    DetectionState.Connecting -> ControlRow(
                        icon = R.drawable.ic_bell,
                        tone = ToneWarn,
                        title = "Detección de pagos",
                        value = "Conectando",
                        valueColor = AmberText,
                    )
                    DetectionState.Active -> ControlRow(
                        icon = R.drawable.ic_bell,
                        tone = ToneOk,
                        title = "Detección de pagos",
                        value = "Activa",
                        valueColor = FudiaGreenDark,
                    )
                }
                // La exclusión de batería refuerza el push y la alarma de
                // respaldo; solo se pide cuando falta.
                if (!exempt) {
                    RowDivider()
                    ControlRow(
                        icon = R.drawable.ic_battery,
                        tone = ToneWarn,
                        title = "Batería",
                        value = "Permitir",
                        valueColor = FudiaPrimaryDark,
                        onClick = {
                            runCatching {
                                context.startActivity(
                                    Intent(
                                        Settings.ACTION_REQUEST_IGNORE_BATTERY_OPTIMIZATIONS,
                                        Uri.parse("package:${context.packageName}"),
                                    )
                                )
                            }
                        },
                    )
                }
                RowDivider()
                ControlRow(
                    icon = R.drawable.ic_store,
                    tone = ToneInfo,
                    title = credentials.organizationName.ifBlank { "Sin vincular" },
                    subtitle = credentials.locationName.ifBlank { null },
                )
                RowDivider()
                ControlRow(
                    icon = R.drawable.ic_cloud_upload,
                    tone = ToneInfo,
                    title = "Pagos sin enviar",
                    value = if (count > 0) "Enviar $count" else "0",
                    valueColor = if (count > 0) FudiaPrimaryDark else MaterialTheme.colorScheme.onSurface,
                    onClick = if (count > 0) ({ scheduler.enqueueNow() }) else null,
                )
            }
        }

        // Salida al pie, en tono secundario: acción infrecuente que solo limpia
        // este teléfono.
        TextButton(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 16.dp, vertical = 8.dp)
                .defaultMinSize(minHeight = 48.dp),
            onClick = onUnlink,
        ) { Text("Desvincular este teléfono") }
    }
}

/** Fila de control: icono en cuadro de color, título (y subtítulo) a la
 *  izquierda y el valor a la derecha. Con [onClick] la fila entera es táctil. */
@Composable
private fun ControlRow(
    @DrawableRes icon: Int,
    tone: RowTone,
    title: String,
    subtitle: String? = null,
    value: String? = null,
    valueColor: Color = MaterialTheme.colorScheme.onSurface,
    onClick: (() -> Unit)? = null,
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .defaultMinSize(minHeight = 56.dp)
            .then(if (onClick != null) Modifier.clickable(onClick = onClick) else Modifier)
            .padding(horizontal = 14.dp, vertical = 10.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        Box(
            modifier = Modifier.size(36.dp).clip(RoundedCornerShape(10.dp)).background(tone.tint),
            contentAlignment = Alignment.Center,
        ) {
            Icon(painterResource(icon), contentDescription = null, tint = tone.ink, modifier = Modifier.size(20.dp))
        }
        Column(modifier = Modifier.weight(1f)) {
            Text(title, style = MaterialTheme.typography.bodyLarge)
            if (subtitle != null) {
                Text(subtitle, style = MaterialTheme.typography.bodySmall, color = InkSoft)
            }
        }
        if (value != null) {
            Text(value, style = MaterialTheme.typography.labelLarge, fontWeight = FontWeight.Bold, color = valueColor)
        }
    }
}

/** Línea fina entre filas, alineada con el texto (no con el icono). */
@Composable
private fun RowDivider() {
    HorizontalDivider(color = FudiaCloudLine, modifier = Modifier.padding(start = 62.dp))
}
