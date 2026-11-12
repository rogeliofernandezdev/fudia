# Contrato y límites del transporte

El contrato HTTP privado es `../api/openapi.yaml`. El servicio recibe país solo
para vincular/verificar la sesión; foods-backend decide cuál está activa.
Las referencias SQL de dispositivos anteriores permiten respuestas originales,
no números operativos secundarios. Redis/IA y reglas de negocio permanecen en
Concierge. Operación y protección del SQL store: `../README.md` y
`../../foods-infrastructure/docs/13_WHATSAPP_GATEWAY.md`.

Un envío nuevo espera aleatoriamente 2,5–5 segundos. La espera es cancelable;
la referencia idempotente evita repetir envíos registrados. Un estado incierto
requiere reconciliación, no un reenvío con otro identificador. El envío completo
está acotado a 50 segundos y el cliente privado concede margen para persistir
su resultado. Texto/captions se limitan por caracteres Unicode.
