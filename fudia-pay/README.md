# Fudia Pay

Aplicación nativa de Android que detecta las notificaciones de pagos recibidos
por **Yape** y **Plin** en el teléfono del restaurante y las envía al backend de
Fudia, donde Caja las revisa, las asocia a una venta y las confirma. No es una
transacción bancaria: es una alerta detectada que el cajero confirma.

## Cómo funciona

1. El administrador genera un **código temporal** de vinculación en
   `foods-admin-web` › Locales › Editar local › Dispositivos de pago › «Vincular dispositivo»
   y lo muestra como QR.
2. En el Android del local se instala Fudia Pay, se escanea el QR y el backend
   devuelve la credencial del dispositivo y el restaurante/local asociado.
   La credencial se guarda cifrada con Keystore y no se vuelve a mostrar.
3. El operador activa **el acceso a notificaciones** desde la configuración de
   Android (permiso explícito del sistema).
4. Cada notificación de Yape o Plin se procesa con un parser por proveedor,
   entra a una cola local persistente y **se envía al instante desde el propio
   servicio de escucha**, sin abrir la app. WorkManager queda de respaldo
   (sin red o backend caído) con reintentos con backoff y un barrido cada
   15 minutos; al reenlazarse el servicio también vacía lo pendiente.
5. El backend deduplica por dispositivo y clave de evento, expone los pagos
   detectados en Caja y el cajero los confirma con el flujo de pagos existente.

## Requisitos

- Android Studio (JDK 17), `minSdk 24`, `targetSdk 35`.
- Un backend de Fudia accesible por HTTPS con la funcionalidad de
  vinculación activa (`FOODS_API_URL` del despliegue).

## Compilar y probar

```bash
./gradlew :app:testDebugUnitTest    # pruebas unitarias
./gradlew :app:assembleDebug        # APK de depuración (backend de desarrollo)
./gradlew :app:assembleDebug -PfoodsApiUrl=https://tu-backend/   # URL explícita
./gradlew :app:assembleRelease -PfoodsApiUrl=https://produccion/ # APK de producción
```

Las variantes apuntan por defecto: `debug` al backend de desarrollo y
`release` al de producción; los valores exactos se ajustan en
`app/build.gradle.kts` (placeholders con tus URLs de Cloud Run).

El APK se instala directamente en el Android del local
(`adb install -r app/build/outputs/apk/debug/app-debug.apk`).

**Permisos restringidos (HONOR/Huawei/Xiaomi/Oppo):** al instalar el APK por
fuera de la tienda, MagicOS/HyperOS/ColorOS bloquea permisos delicados. En el
teléfono: Ajustes › Apps › Gestionar apps › Fudia Pay › menú ⋮ ›
**«Permitir ajustes restringidos»** → confirma. Después, el acceso a
notificaciones se puede activar con normalidad. Es un paso único por teléfono.

**Vida del servicio en teléfonos con batería agresiva:** el permiso de escucha
puede estar concedido con el servicio muerto. Por eso:

- Tras **actualizar el APK, abre Fudia Pay una vez**: MagicOS no reconecta el
  `NotificationListenerService` solo y el sistema no reenvía notificaciones
  anteriores al reenlace.
- El envío no depende de WorkManager: en segundo plano Android y MagicOS
  aplazan sus trabajos hasta que se abre la app, por eso el listener (que el
  sistema mantiene enlazado con prioridad alta) envía cada pago él mismo.
- El servicio se **reenlaza solo** cuando Android lo desconecta
  (`onListenerDisconnected` → `requestRebind`), sin abrir la app.
- La pantalla de inicio muestra la **salud real** (conectado/desconectado y
  hace cuánto, según lo que reporta el propio servicio) y ofrece excluir la
  app de la **optimización de batería**; en ajustes de marca conviene además
  activar «Iniciar automáticamente» y fijar la app en recientes.

## Estructura

```
app/src/main/java/com/fudia/pay/
  core/network    cliente HTTP (Retrofit/OkHttp) e interceptor de credencial
  core/security   credencial cifrada con Keystore
  features/
    pairing       bienvenida, escaneo del QR y activación del dispositivo
    notifications NotificationListenerService y parsers por proveedor
    payments      eventos detectados y su cola local (Room)
    sync          envío de la cola (PaymentSyncer) y respaldo con WorkManager
    settings      inicio, historial y configuración
```

## Interfaz

Sigue el sistema visual de `foods-admin-web` (`docs/02_DESIGN_SYSTEM.md`):
colores en `core/design/Theme.kt` y una cabecera compartida
(`core/design/FudiaHeader.kt`) igual en bienvenida e inicio, como la cabecera
marino del panel web: fondo `navigation-900`, la marca en un cuadro blanco de
44 dp, «Fudia Pay» en negrita blanca y «Pagos de Yape y Plin para tu caja» en
`navigation-text`.

La bienvenida muestra una sola acción por estado: «Vincular restaurante» (azul,
48 dp) con la instrucción de Caja debajo; con el escáner abierto, solo el
escáner y «Volver»; sin permiso de cámara, «Conceder cámara»; al vincular,
«Continuar».

Al leer el QR, la app avisa el escaneo (`POST /v1/mobile/pair/scan`, con el
modelo del teléfono) y luego canjea el código: así Caja muestra «Vinculando…»
como WhatsApp Web y cierra su modal sola al terminar. Los errores muestran el
mensaje del backend («El código de vinculación ya se usó. Genera otro.»), sin
códigos HTTP.

## Contrato con el backend

El parser recoge el número de operación cuando la notificación incluye una
referencia explícita («Número de operación», «N° de operación», «Operación» o
«Código de operación»). También lee el texto expandido; no usa la clave de
Android, el importe ni códigos de seguridad como sustitutos. Si no existe una
referencia inequívoca, reporta el pago sin número y Caja muestra «No informado».
La cola Room versión 2 conserva los eventos pendientes mediante migración.
Requiere desplegar primero la migración 000135 y la API con `operationNumber`.

`foods-backend/api/openapi.yaml` › *Fudia Pay*. El dispositivo se autentica con
su credencial revocable; el backend resuelve empresa y local; la plataforma del
dispositivo se declara como `ANDROID` o `IOS` (solo Android en esta etapa).

Detalle de la funcionalidad en `foods-backend/docs/16_FUDIA_PAY.md`.
