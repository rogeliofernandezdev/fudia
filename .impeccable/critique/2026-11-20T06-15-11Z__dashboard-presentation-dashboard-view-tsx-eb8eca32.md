---
target: dashboard
total_score: 24
max_score: 40
na_heuristics: 
p0_count: 1
p1_count: 3
target_identity: "file:C:\\rep\\foods\\foods-admin-web\\src\\modules\\dashboard\\presentation\\dashboard-view.tsx"
target_fingerprint: "sha256:4ec35d05a537789790cd36ff64d608646baa60e1b9c5925c9262e0df571ec4f8"
target_path: "C:\\rep\\foods\\foods-admin-web\\src\\modules\\dashboard\\presentation\\dashboard-view.tsx"
timestamp: 2026-11-20T06-15-11Z
slug: dashboard-presentation-dashboard-view-tsx-eb8eca32
---
# Critique: Dashboard (foods-admin-web) — 2026-11-20

Method: dual-agent (A: design review · B: detector + browser) + market research (Toast, Square, Lightspeed, R365, Loyverse, TouchBistro, Fudo).

## Heuristics (24/40, Acceptable)
| # | Heuristic | Score | Key issue |
|---|---|---|---|
| 1 | System status | 3 | «En vivo» without «Actualizado hh:mm»; CountUp shows intermediate amounts |
| 2 | Real world | 3 | «Pedidos cobrados» hides the "fully paid" rule |
| 3 | User control | 3 | No custom range |
| 4 | Consistency | 2 | Mixed ranking basis; 6 por cobrar vs 5 activos; two sales definitions |
| 5 | Error prevention | 2 | Partial day vs full day comparison -> false −100% |
| 6 | Recognition | 3 | ok |
| 7 | Flexibility | 1 | No drill-down, no "ver todo", no multi-location |
| 8 | Minimalist | 3 | One-slice donut wastes space; empty chart area |
| 9 | Error recovery | 3 | Raw error.message, developer copy |
| 10 | Help | 1 | No definition of what Ventas/Promedio include |

## Priority issues
- P0 Biased comparisons: «Hoy» compares today-so-far vs the whole same weekday last week (PreviousOffsets 7,7; dashboardPaidOrdersSQL). Promedio −100% with 0 orders.
- P1 Inconsistent definitions: Ventas cash-basis; Pedidos = fully paid orders now; Promedio = sum(total)/count incl. delivery_fee; refunds rewrite past periods.
- P1 Ranking: products ordered by qty, bars by revenue; categories by revenue; products capped at 5 without «otros».
- P1 Mobile: KPI labels truncated at 427/390 («Pedidos cobra…», «Promedio por p…»); «Ahora mismo» after ~4 screens.
- P2 Live panel: openOrders vs active_orders definitions differ; tablesOccupied lacks service_type='table'; hourlySales/kitchenPending payload unused; trend `previous` unused.

## Missing metrics with existing data
Propinas (payment_tips), devoluciones/anulaciones, ventas por mozo (waiter_id), tiempo de cocina (order_service_items.delivered_at), tiempo de mesa, comensales de reservas, fantasma semana pasada en gráfico, delivery fees. Need new data: comensales por pedido, descuentos/cortesías, food cost, labor.

## Detector
CLI clean (0). Live overlay: 9 findings, all in shell (sidebar labels gray-on-navy ~7:1, «Local activo» 10px, overused font = approved). No overflow at 390/427.
