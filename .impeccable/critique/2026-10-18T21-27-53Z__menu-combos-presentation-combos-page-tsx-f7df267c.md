---
target: asistente Arma tu menú o combo
total_score: 22
max_score: 40
na_heuristics: 
p0_count: 1
p1_count: 3
target_identity: "file:C:\\rep\\foods\\foods-admin-web\\src\\modules\\menu\\combos\\presentation\\combos-page.tsx"
target_fingerprint: "sha256:18f59f438c36164b58c754c76362a96825c448facaef7e200069f32f4847bd5c"
target_path: "C:\\rep\\foods\\foods-admin-web\\src\\modules\\menu\\combos\\presentation\\combos-page.tsx"
timestamp: 2026-10-18T21-27-53Z
slug: menu-combos-presentation-combos-page-tsx-f7df267c
---
# Crítica: asistente «Arma tu menú o combo» (2026-10-18)

Método: dual-agent (A diseño · B detector + navegador). Escritorio 1536×639, móvil 427×933 y 390×844 (iframe).

## Salud (Nielsen) 22/40 — Aceptable
1 Estado 2 (error fuera de vista en paso 2) · 2 Mundo real 3 · 3 Control 2 (quitar parte sin deshacer, plantilla sobrescribe) · 4 Consistencia 2 (dos primarios en paso 2) · 5 Prevención 1 (Hoy+días imposibles, Continuar/Guardar misma posición) · 6 Reconocimiento 3 · 7 Eficiencia 2 · 8 Minimalismo 2 (título del paso ×3, 9+ X rojas, tarjetas anidadas) · 9 Recuperación 2 · 10 Ayuda 3

## Detector
CLI limpio (0). Runtime: 3 hallazgos en el asistente, títulos del resumen a 10 px (combo-wizard.css:544). Resto fuera del objetivo.

## Prioridades
- [P0] Error invisible en pasos largos: #combo-wizard-error arriba de .combo-wizard-body (combos-page.tsx:193).
- [P1] Paso 2: 2207 px de scroll a 427 px, tres niveles de tarjetas, nombres de parte como inputs sin jerarquía, Reservar/Recargo siempre abiertos, dos primarios.
- [P1] Acciones destructivas sin red: quitar parte, plantilla que reemplaza.
- [P1] Disponibilidad: «Hoy» mezclado con días, label envuelve botones (activa Hoy), fecha del navegador en vez del local.
- [P2] Redundancia y exceso de índigo: hints del stepper, iconos de sección, notas del pie, chips sólidos; precio sin protagonismo en el resumen.

## Personas
Jordan: «Reservar», «0», «Obligatorio» sin significado. Sam: nombres accesibles repetidos, chips sin aria-pressed. Casey: paso 2 de 3+ pantallas, X roja arriba a la derecha sin deshacer.
