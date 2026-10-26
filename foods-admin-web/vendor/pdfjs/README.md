# Mozilla PDF.js: visor oficial completo

`web/` conserva sin cambios el visor genérico legacy de Mozilla, sus imágenes
y traducciones. No es solo la clase `PDFViewer` de `pdfjs-dist/web`.
Origen, versión y SHA-256 del ZIP están en `manifest.json`; licencia Apache-2.0
en `LICENSE`. No se incluye el PDF de ejemplo ni los source maps.

`scripts/prepare-pdfjs.mjs` copia este visor y el motor, worker, fuentes, CMaps,
perfiles ICC y WASM de **la misma versión fijada de `pdfjs-dist`** a
`public/pdfjs/`. Añade únicamente los adaptadores propios de FudIA al HTML
generado, sin editar los archivos originales de Mozilla. Falla si las versiones
no coinciden. `predev`, `prebuild` y `pretest` lo ejecutan; Docker/Cloud Run
incluyen el directorio generado mediante el COPY de `public` existente.

Para actualizar: descargar el ZIP legacy de una release oficial, comprobar su
hash, reemplazar `web/` y la licencia, fijar esa misma versión exacta en npm y
el manifiesto, y repetir pruebas funcionales y de impresión en dispositivos.
No basta con actualizar solamente el worker o el paquete npm.
