# Fudia Pay

Aplicación nativa de Android que detecta las notificaciones de pagos recibidos
por **Yape** y **Plin** en el teléfono del restaurante y las envía al backend de
Fudia, donde Caja las revisa, las asocia a una venta y las confirma. No es una
transacción bancaria: es una alerta detectada que el cajero confirma.

## Cómo funciona

1. El administrador genera un **código temporal** de vinculación en
   `foods-admin-web` › Mi restaurante › Dispositivos de pago › «Vincular dispositivo»
   y lo muestra como QR.
2. En el Android del local se instala Fudia Pay, se escanea el QR y el backend
   devuelve la credencial del dispositivo y el restaurante/local asociado.
   La credencial se guarda cifrada con Keystore y no se vuelve a mostrar.
3. El operador activa **el acceso a notificaciones** desde la configuración de
   Android (permiso explícito del sistema).
4. Cada notificación de Yape o Plin se procesa con un parser por proveedor,
   entra a una cola local persistente y se sincroniza con WorkManager y
   reintentos con backoff.
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

## Estructura

```
app/src/main/java/com/fudia/pay/
  core/network    cliente HTTP (Retrofit/OkHttp) e interceptor de credencial
  core/security   credencial cifrada con Keystore
  features/
    pairing       bienvenida, escaneo del QR y activación del dispositivo
    notifications NotificationListenerService y parsers por proveedor
    payments      eventos detectados y su cola local (Room)
    sync          sincronización con WorkManager y reintentos
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

`foods-backend/api/openapi.yaml` › *Fudia Pay*. El dispositivo se autentica con
su credencial revocable; el backend resuelve empresa y local; la plataforma del
dispositivo se declara como `ANDROID` o `IOS` (solo Android en esta etapa).

Detalle de la funcionalidad en `foods-backend/docs/16_FUDIA_PAY.md`.
