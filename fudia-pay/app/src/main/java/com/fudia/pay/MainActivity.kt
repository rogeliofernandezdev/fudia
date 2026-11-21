package com.fudia.pay

import androidx.activity.ComponentActivity
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.compose.setContent
import android.Manifest
import android.content.pm.PackageManager
import android.os.Build
import android.os.Bundle
import androidx.activity.SystemBarStyle
import androidx.activity.enableEdgeToEdge
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.navigationBars
import androidx.compose.foundation.layout.windowInsetsPadding
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import androidx.compose.ui.viewinterop.AndroidView
import androidx.core.content.ContextCompat
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.fudia.pay.core.design.FudiaCloud
import com.fudia.pay.core.design.FudiaGreen
import com.fudia.pay.core.design.FudiaPrimary
import com.fudia.pay.core.design.FudiaHeader
import com.fudia.pay.core.design.FudiaTheme
import com.fudia.pay.core.network.FudiaPayApi
import com.fudia.pay.core.network.SessionResponse
import com.fudia.pay.core.security.DeviceCredentials
import com.fudia.pay.features.pairing.PairingViewModel
import com.fudia.pay.features.notifications.ListenerHealth
import com.fudia.pay.features.notifications.ListenerStatus
import com.fudia.pay.features.notifications.ListeningService
import com.fudia.pay.features.settings.HomeScreen
import com.fudia.pay.features.sync.SyncScheduler
import com.journeyapps.barcodescanner.DecoratedBarcodeView
import dagger.hilt.android.AndroidEntryPoint
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.launch
import javax.inject.Inject

@HiltViewModel
class SessionViewModel @Inject constructor(
    private val api: FudiaPayApi,
    private val credentials: DeviceCredentials,
    private val scheduler: SyncScheduler,
) : ViewModel() {
    val paired: Boolean get() = credentials.paired

    fun refreshSession() {
        viewModelScope.launch {
            runCatching {
                val session: SessionResponse = api.session()
                credentials.organizationName = session.restaurant.name
                credentials.locationName = session.location.name
            }
            scheduler.schedulePeriodic()
        }
    }

    fun unlink(onDone: () -> Unit) {
        viewModelScope.launch {
            // La revocación remota se hace desde Caja; aquí solo se limpia el equipo.
            credentials.clear()
            onDone()
        }
    }
}

@AndroidEntryPoint
class MainActivity : ComponentActivity() {
    @Inject lateinit var credentials: DeviceCredentials
    @Inject lateinit var scheduler: SyncScheduler
    @Inject lateinit var dao: com.fudia.pay.features.payments.PendingEventDao
    @Inject lateinit var api: FudiaPayApi
    @Inject lateinit var listenerStatus: ListenerStatus

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        // Barras del sistema sobre la franja marino: iconos claros, contenido
        // con insets para no quedar debajo de la barra de estado.
        enableEdgeToEdge(
            statusBarStyle = SystemBarStyle.dark(android.graphics.Color.TRANSPARENT),
            navigationBarStyle = SystemBarStyle.dark(android.graphics.Color.TRANSPARENT),
        )
        var session by mutableStateOf(credentials.paired)
        if (credentials.paired) startListening()
        setContent {
            FudiaTheme {
                if (!session) WelcomeFlow(onPaired = { session = true; scheduler.schedulePeriodic(); startListening() })
                else {
                    val sessionViewModel: SessionViewModel = hiltViewModel()
                    val health by listenerStatus.health.collectAsState(initial = ListenerHealth(0, 0))
                    HomeScreen(
                        credentials = credentials,
                        listenerHealth = health,
                        dao = dao,
                        scheduler = scheduler,
                        onUnlink = { sessionViewModel.unlink { ListeningService.stop(this); session = false } },
                    )
                    sessionViewModel.refreshSession()
                }
            }
        }
    }

    private val notificationPermission = registerForActivityResult(ActivityResultContracts.RequestPermission()) {
        // Con o sin permiso, el servicio fijo corre; el permiso solo hace
        // visible su aviso «Detectando pagos».
        ListeningService.start(this)
    }

    /** Servicio fijo que impide que HONOR congele la app (ver ListeningService). */
    private fun startListening() {
        val needsPermission = Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU &&
            ContextCompat.checkSelfPermission(this, Manifest.permission.POST_NOTIFICATIONS) != PackageManager.PERMISSION_GRANTED
        if (needsPermission) notificationPermission.launch(Manifest.permission.POST_NOTIFICATIONS)
        else ListeningService.start(this)
    }
}

/** Bienvenida → escáner incrustado con la identidad de Fudia → vinculado.
 * El QR lo muestra Caja en la computadora: la cámara lo lee y el canje es
 * automático. La cámara no se abre sin permiso concedido. */
@Composable
private fun WelcomeFlow(onPaired: () -> Unit) {
    val viewModel: PairingViewModel = hiltViewModel()
    val state by viewModel.state.collectAsState()
    val context = LocalContext.current
    var cameraGranted by remember {
        mutableStateOf(
            ContextCompat.checkSelfPermission(context, Manifest.permission.CAMERA) == PackageManager.PERMISSION_GRANTED
        )
    }
    val cameraLauncher = rememberLauncherForActivityResult(ActivityResultContracts.RequestPermission()) {
        cameraGranted = it
    }
    val startCamera = { if (!cameraGranted) cameraLauncher.launch(Manifest.permission.CAMERA) }
    LaunchedEffect(Unit) { startCamera() }
    var scanning by remember { mutableStateOf(false) }

    Column(modifier = Modifier.fillMaxSize()) {
        // Cabecera compartida con Inicio (marca, nombre y descripción de la app).
        FudiaHeader()
        Column(
            modifier = Modifier
                .fillMaxSize()
                .windowInsetsPadding(WindowInsets.navigationBars)
                .padding(16.dp)
                .verticalScroll(rememberScrollState()),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            // Una sola acción por estado: «Vincular restaurante» solo antes de
            // abrir el escáner; mientras escanea, el escáner y «Volver».
            if (!state.done && cameraGranted && !scanning) {
                Button(
                    modifier = Modifier.fillMaxWidth().height(48.dp),
                    onClick = { scanning = true },
                ) { Text("Vincular restaurante") }
            }
            if (!state.done) {
                Text(
                    "En tu computadora, abre Locales › Editar local › Dispositivos de pago › «Vincular dispositivo» y apunta la cámara al QR que muestra.",
                    style = MaterialTheme.typography.bodySmall,
                    color = Color(0xFF39507A),
                )
            }

            if (state.done) {
                Card(colors = CardDefaults.cardColors(containerColor = Color(0xFFE3F8E9))) {
                    Column(modifier = Modifier.padding(12.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
                        Text("Vinculado", color = FudiaGreen, style = MaterialTheme.typography.titleMedium)
                        Text("Activa la detección de pagos para empezar a recibir avisos.", style = MaterialTheme.typography.bodyMedium)
                        Button(modifier = Modifier.fillMaxWidth(), onClick = onPaired) { Text("Continuar") }
                    }
                }
            } else if (!cameraGranted) {
                Card {
                    Column(modifier = Modifier.padding(12.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
                        Text("La cámara necesita permiso para leer el QR.")
                        Button(modifier = Modifier.fillMaxWidth(), onClick = startCamera) { Text("Conceder cámara") }
                    }
                }
            } else if (scanning) {
                // Escáner incrustado, con nuestra identidad y nuestras instrucciones.
                Text("Escanea el QR de vinculación", style = MaterialTheme.typography.titleMedium)
                val barcodeView = remember {
                    DecoratedBarcodeView(context).apply {
                        barcodeView.decoderFactory = com.journeyapps.barcodescanner.DefaultDecoderFactory(listOf(com.google.zxing.BarcodeFormat.QR_CODE))
                    }
                }
                DisposableEffect(state.pairing) {
                    if (!state.pairing) {
                        barcodeView.decodeSingle(object : com.journeyapps.barcodescanner.BarcodeCallback {
                            override fun barcodeResult(result: com.journeyapps.barcodescanner.BarcodeResult) {
                                viewModel.pair(result.text)
                            }
                            override fun possibleResultPoints(resultPoints: List<com.google.zxing.ResultPoint>) {}
                        })
                        barcodeView.resume()
                    } else barcodeView.pause()
                    onDispose { barcodeView.pause() }
                }
                AndroidView(
                    factory = { barcodeView },
                    modifier = Modifier.fillMaxWidth().aspectRatio(1f).background(Color(0xFF041A40), RoundedCornerShape(14.dp)),
                )
                if (state.pairing) Text("Vinculando dispositivo…", style = MaterialTheme.typography.bodyMedium)
                state.error?.let { Text(it, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodySmall) }
                OutlinedButton(modifier = Modifier.fillMaxWidth(), onClick = { scanning = false }) { Text("Volver") }
            }
        }
    }
}
