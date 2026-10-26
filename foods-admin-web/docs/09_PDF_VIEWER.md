# Visor PDF compartido

`src/shared/pdf-viewer` encapsula el visor oficial completo de Mozilla PDF.js
6.4.299. Usa `pdfjs-dist` directamente, sin React-PDF ni React PDF Viewer.
Integra los tickets de pago (80 mm, Ventas/POS) y los informes de cierre de caja
(A4 horizontal), conservando generadores, contenido y consultas autorizadas.

## Reutilización

```tsx
import {PdfViewer} from "@/shared/pdf-viewer";

return previewOpen ? <PdfViewer
  document={pdfBlob}
  filename="informe.pdf"
  title="Vista previa del informe"
  loading={preparing}
  error={errorMessage}
  onRetry={preparePdf}
  onClose={() => setPreviewOpen(false)}
/> : null;
```

`document` es un Blob o `null` mientras se prepara. `notice` añade contexto
opcional y `details` habilita «Ver detalle / Ver PDF». Mantener el Blob estable
durante la visualización. Cambiarlo carga un documento nuevo y libera el anterior.
El llamador conserva permisos, descarga autenticada/generación y errores de esa
operación. No añadir botones externos de imprimir o abrir.
El zoom inicial es «Ajuste de la página»: muestra la hoja completa inmediatamente,
sin abrir impresión. No cambia el formato de 80 mm ni A4. El control «Anchura de
la página» sigue disponible para leer con más detalle. `size="sm"` conserva el
modal de ticket de 480 px y `size="lg"` el informe de 960 px; respetan el límite
del viewport y las hojas móviles del sistema visual. `subtitle` conserva el
contexto de la cabecera original.
La barra, superficie de lectura y acciones del visor usan gris oscuro neutro,
con controles claros y hoja blanca, como un visor PDF de escritorio. La cabecera
del modal conserva FudIA. El tema reside en tokens `pdf-viewer-*` de
`variables.css`, incluidos menús, búsqueda y miniaturas oficiales.

## Distribución y despliegue

- `vendor/pdfjs/web` contiene el visor genérico completo y sin modificar de
  Mozilla, con licencia Apache y manifiesto de procedencia/SHA-256.
- `pdfjs-dist` aporta motor legacy, worker y recursos de la misma versión.
- `scripts/prepare-pdfjs.mjs` comprueba versiones y genera `public/pdfjs` antes
  de dev, tests y build. Los assets generados no se versionan.
- El Dockerfile existente copia `public` al runtime: incluye visor, worker,
  traducciones, fuentes, CMaps y WASM. No usa CDN ni servicios externos.
- El wrapper Next se carga con `ssr:false`. La API backend no cambia.
- Los estilos del contenedor se importan en `src/app/layout.tsx`, junto al resto
  del sistema visual. No depender de CSS dentro del chunk dinámico: su ausencia
  deja el iframe en 300×150 px y elimina el espacio de lectura.
- Para actualizar: fijar versión npm, sustituir distribución oficial y manifiesto,
  ejecutar prepare y repetir pruebas. No actualizar solo worker/motor ni sustituir
  el visor completo por `pdf_viewer.mjs`, que es una librería de componentes.

## Seguridad y ciclo de vida

El iframe carga HTML propio del mismo origen, no una URL privada en el visor
nativo. El padre transfiere bytes mediante ArrayBuffer y verifica origen, ventana
emisora y sesión. No envía cookies/tokens a servicios externos. La CSP limita
conexiones al propio origen; se desactiva scripting del PDF y modelos remotos.
Las cabeceras impiden que otro origen lo incruste.

La cola oficial prioriza páginas visibles y cercanas; no dibuja todos los canvas
al abrir. Cada canvas se limita a 2 MP móvil y 8 MP escritorio, dimensión máxima
4096, sin canvas extra de detalle. Precargar metadata para navegar/imprimir no
equivale a dibujar páginas. Cerrar/cambiar PDF destruye documento/tareas/worker
con `app.close()`, retira listeners, cancela impresión y revoca URLs temporales
del iframe. Las respuestas tardías no reemplazan un documento nuevo.

## Impresión

«Imprimir» invoca `PDFViewerApplication.triggerPrinting()` y el PDFPrintService
oficial. Nunca imprime la pantalla de FudIA. Conserva el render secuencial y el
diálogo del sistema; impresora/papel/escala pertenecen al usuario. No conecta
directamente mediante Bluetooth, USB o Wi-Fi. El ticket conserva 80 mm: elegir
papel adecuado en el controlador físico.

La ayuda móvil menciona «Todas las impresoras» si el sistema lo ofrece. Si falta
soporte o falla la preparación se mantiene «Descargar PDF». No se confirma que
una impresora haya impreso ni se garantiza compatibilidad universal. Imprimir
muchos folios consume más memoria que leerlos progresivamente.

## Comprobación reproducible

`npm test` incluye integridad/versiones, protocolo, límites de render, impresión
delegada/errores y limpieza. Los tests de ticket/caja conservan contenido/formato.

Tras `npm run pdfjs:prepare`, ejecutar
`node tests/helpers/pdf-viewer-preview.mjs`: sirve `localhost:5194` con el
componente real y PDFs ficticios (ticket, cierre largo, 80 páginas, PDF dañado).
No necesita sesión/API ni registra ventas/cobros. Es QA local, sin rutas de
producción.

Probar escritorio y 390×844, 390×400, 430×932: apertura, navegación, zoom/ancho,
búsqueda, rotación, pantalla completa, descarga, impresión, cierre/reapertura y
sustitución del Blob. Confirmar pie dentro del viewport y pocos canvas activos.

Antes de liberar, probar Chrome Android y Safari iOS físicos con los servicios
e impresoras del restaurante: «Todas las impresoras» si aparece, cancelación,
descarga alternativa, safe areas y documentos grandes con memoria limitada.
La emulación de tamaño no valida impresión física ni gestos reales.

### Resultado de esta implementación

Comprobado en navegador con los PDFs ficticios del componente real:
apertura de ticket, documento de 80 páginas, salto al folio 40/siguiente,
zoom al 100 % y aumento, búsqueda con una coincidencia, rotación, pantalla
completa/salida, sustitución del Blob y cierre (cero iframes restantes).
El documento de 80 páginas mantenía tres canvas al observarlo en móvil.
Se revisaron escritorio, 390×844, 430×932 y 390×400; en esta última vista
se midieron los botones Imprimir/Descargar: 44 px y dentro del viewport.
El archivo inválido muestra error explícito.
También se verificó el ticket real desde Ventas → detalle → Previsualizar:
modal de 480 px, iframe ocupando el cuerpo y barra sin desborde horizontal.
La carga global de estilos corrige el iframe por defecto de 300×150 px.

El botón Imprimir abrió «Preparando documento para impresión…» de PDF.js.
La automatización quedó bloqueada al entrar en impresión y no confirmó el
diálogo del sistema. El clic de descarga se ejecutó, pero el navegador
automatizado no entregó un evento/archivo verificable. No se consideran
validados la descarga final ni el resultado de impresión física. Repetirlos
manualmente en Chrome de escritorio, Chrome Android y Safari iPhone.
