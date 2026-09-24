# Observabilidad — Fudia Concierge

## Principio

La observabilidad no debe convertir los logs en una copia de las conversaciones.
No se registran mensajes, tokens QR, credenciales ni números telefónicos en claro.

## Eventos estructurados

Concierge emite logs JSON para:

- mensaje procesado correctamente;
- mensaje fallido;
- llamada LLM completada.

Los identificadores externos se reducen a fingerprints SHA-256 truncados. Para
el LLM se registran modelo, latencia, cantidad de tools e input/output tokens,
pero no prompt ni respuesta.

## Trazabilidad persistente

foods-backend conserva la fuente auditable de acciones de negocio:

- pedidos creados o ampliados por Concierge;
- conversationId usado para idempotencia;
- solicitudes de handoff humano;
- usuario interno que resuelve un handoff.

## Métricas derivables

A partir de logs y base de datos se pueden calcular:

- mensajes procesados/fallidos;
- latencia conversacional;
- consumo de tokens por modelo;
- tools por interacción;
- pedidos confirmados;
- rondas adicionales;
- solicitudes de atención humana;
- tiempo entre solicitud y resolución del handoff.

La siguiente capa de infraestructura puede exportar estos eventos a la solución
de logs/métricas elegida sin cambiar el dominio de Concierge.
