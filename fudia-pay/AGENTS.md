# AGENTS — Fudia Pay (Android)

Aplicación nativa de Android del ecosistema Fudia. Aplicar el `AGENTS.md` raíz
y leer el `README.md` de este proyecto antes de implementar cambios relevantes.

## Alcance

Una sola función: detectar las notificaciones de pagos recibidos por Yape y Plin
en el Android del restaurante y reportarlas al backend de Fudia para que Caja
las revise y confirme. Sin ventas, inventario, carta, administración ni cobros.

## Reglas

1. Kotlin + Jetpack Compose, Clean Architecture con MVVM, coroutines y Flow.
   Sin Flutter, React Native ni frameworks híbridos.
2. Organización por funcionalidades: `features/{pairing,notifications,payments,
   sync,settings}` y `core/{network,security}`. Sin dependencias innecesarias.
3. Solo `NotificationListenerService` para capturar notificaciones. Prohibido
   `AccessibilityService`, SMS, contactos, cuentas o datos privados adicionales.
4. Solo se procesan los paquetes Android verificados de Yape y Plin; el nombre
   del paquete se define en código revisado, nunca se asume.
5. Parsers independientes por proveedor. Si el texto no identifica con
   confianza un pago recibido (remitente y monto), el evento no se registra.
6. El dispositivo nunca envía empresa, local ni identificadores: el backend los
   resuelve desde la vinculación. La credencial del dispositivo se guarda solo
   en almacenamiento cifrado con Keystore y se muestra una sola vez.
7. Idempotencia de extremo a extremo: `providerEventKey` estable por evento,
   cola local persistente (Room) y reintentos con WorkManager y backoff.
   Dos pagos legítimos con el mismo monto y hora no comparten clave.
8. Nada se presenta como transacción bancaria verificada: es una notificación
   detectada que Caja confirma de forma manual. Los estados son `detected`,
   `associated` y `confirmed`; los define el backend.
9. Sin WebSocket permanente ni servicios foreground sin justificación. La app
   admite que Android no entregue notificaciones y lo muestra con sinceridad.
10. Los logs nunca contienen tokens, credenciales ni contenido de notificaciones.
11. Interfaz en español claro y nombres técnicos en inglés. Pantallas mínimas:
    bienvenida, vinculación, inicio, historial y configuración.
12. Los contratos con el backend son independientes de la plataforma
    (`ANDROID|IOS`); la versión iOS no existe y no debe asumirse que la captura
    de notificaciones podrá replicarse en ella.
13. Antes de cerrar un cambio: `gradlew :app:testDebugUnitTest` y, si hay
    dispositivo o emulador disponible, `gradlew :app:assembleDebug`.
14. Preparación para iOS: backend y contratos primero; no desarrollar la app.
