---
target: interfaces de operación
total_score: 26
max_score: 40
na_heuristics: 
p0_count: 0
p1_count: 3
target_identity: "file:C:\\rep\\foods\\foods-admin-web\\src\\modules\\operations"
timestamp: 2026-11-06T04-29-41Z
slug: src-modules-operations
---
# Crítica: interfaces de operación (foods-admin-web) vs referencia Zillo

## Design Health Score — 26/40 (Aceptable)
| # | Heurística | Puntaje | Problema clave |
|---|---|---|---|
| 1 | Visibilidad del estado | 3 | «N con demora» en Cocina también cuenta «Por vencer» |
| 2 | Lenguaje del usuario | 3 | Buen vocabulario; mayúsculas forzadas en JS (MESA, carriles, HOY) |
| 3 | Control y libertad | 3 | «Iniciar»/«Marcar listo» sin deshacer |
| 4 | Consistencia | 2 | ~15 modales con subtítulo en cabecera; Productos/Dashboard con estados propios |
| 5 | Prevención de errores | 3 | Cobro bloqueado sin turno; cancelar con confirmación |
| 6 | Reconocimiento | 3 | «Cobrar» en POS es solo icono |
| 7 | Flexibilidad | 2 | Sin atajos en Cocina/POS; filtros no persisten |
| 8 | Minimalismo | 2 | Color sólido disperso: tiles vivos por fila, caja, salón, cabeceras de ticket |
| 9 | Recuperación de errores | 3 | ErrorState con reintento; mensajes de cobro específicos |
| 10 | Ayuda | 2 | Descripciones de página genéricas |

## Prioridades
1. [P1] Color sólido por encima de la referencia (tiles vivos por línea, caja, salón, cabecera de ticket) — quieter
2. [P1] Pedidos en 390 px: tabla con scroll horizontal, acción fuera de pantalla — adapt
3. [P1] Contadores blancos sobre naranja #FD9F0A (2.1:1) en pestañas — polish
4. [P2] Componentes compartidos inconsistentes (subtítulo en cabecera, estados propios en Productos/Dashboard) — polish
5. [P2] Pie del detalle de pedido con 5+ acciones — distill
6. [P2] «Por vencer» y «Con demora» casi idénticos; badge mal rotulado — clarify
