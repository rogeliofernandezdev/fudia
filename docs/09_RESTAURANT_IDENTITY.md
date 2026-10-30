# Identidad del restaurante y carta virtual

## Alcance

«Mi restaurante» es el nombre visible del módulo de Empresa existente. Conserva
la entidad `organizations`, sus datos legales, permisos, suscripción y rutas.
No crea otra empresa ni cambia la autenticación de cookie/token opaco.

- Compartidos por restaurante: nombre comercial existente, logo, fecha de
  fundación y carta virtual (imágenes ordenadas, hasta 20 páginas).
- Por local: dirección, horario, zona horaria, fiscalidad y fecha de apertura.
- Las fechas permiten consultar aniversarios; esta entrega no envía campañas
  automáticas ni mensajes de marketing.

El wizard existente incorpora identidad y carta en el paso Restaurante y apertura
en Primer local. Primero confirma el alta transaccional; después sube imágenes
directamente a Storage. Si falla esta segunda fase, conserva el restaurante
creado y permite completar sus imágenes sin repetir el alta. Una recarga pierde
los archivos locales; los datos ya creados se editan después en Mi restaurante.

## Archivos y seguridad

El backend autoriza metadata JPEG/PNG/WebP, 1–5 MiB y hasta 4000 × 4000 px.
PostgreSQL genera el identificador; una política POST V4 limita tamaño y MIME
en Storage. Los bytes no pasan por fudia-api. Persistimos referencias a imágenes,
nunca URLs firmadas absolutas. Las URLs de lectura son privadas y temporales.

`restaurants/{organizationId}/branding/{imageId}/original.{ext}` se diferencia
del espacio de productos. `purpose` distingue `restaurant_logo` y `virtual_menu`.
El API comprueba empresa, permiso, subida confirmada, propósito y autor de la
carga; las imágenes ya publicadas pueden ser mantenidas por otro administrador.
El orden final se guarda atómicamente y la modificación se audita. El acceso de
Plataforma exige administrador de plataforma real y un local activo del destino.

Se reutiliza la reserva, confirmación, Eventarc, lease, fencing y limpieza del
procesador de imágenes. Estados: PROCESSING, READY y FAILED. No se espera a READY
para terminar el formulario. La interfaz muestra preview local/original durante
PROCESSING y consulta su evolución; los fallos permiten reemplazar la imagen.

| Propósito | Variantes necesarias para READY |
| --- | --- |
| Producto | thumbnail.webp 200, medium.webp 600 |
| Logo | thumbnail.webp 200, medium.webp 600, logo.png 600 con transparencia |
| Carta virtual | thumbnail.webp 200, medium.webp 2400, delivery.jpg 2400 |

Las medidas son el lado máximo, sin ampliar. La carta necesita más resolución
que una fotografía de plato para conservar legibles sus textos. JPEG de entrega
y PNG del logo permiten enviarlos como imágenes de WhatsApp. No cambia el perfil
global del número del bot para cada restaurante.

## Consumidores

- Tickets/precuentas de 80 mm e informe de caja: logo proporcional; su ausencia
  no impide cobrar, generar o imprimir. Se conserva contenido y formato del PDF.
- Concierge: el backend resuelve identidad a partir del QR y devuelve URLs recién
  firmadas. Bienvenida: logo y páginas; consulta de carta completa: páginas en
  orden. Solo se envían imágenes READY. Si no hay carta, conserva la carta textual.
  Precios, disponibilidad y confirmación siguen dependiendo del catálogo del backend.
- Redis persiste respuesta e imágenes con la sesión; el outbox entrega texto y
  páginas con índice de avance y referencia Meta independiente. Un reintento
  continúa en la página pendiente sin volver a ejecutar acciones comerciales.

## Despliegue

1. Aplicar migración `000096_restaurant_profile` en foods-backend.
2. Desplegar fudia-image-processor con los nuevos perfiles antes de permitir cargas.
3. Desplegar API, Concierge y frontend compatibles.
4. Mantener CORS del bucket para los dominios frontend. Eventarc debe cubrir todos
   los objetos originales del bucket, incluyendo `branding/`, no solo `products/`.
5. Mantener los permisos existentes: API firma/sube/lee; worker procesa y actualiza
   el contrato SQL `product_images`. No necesita acceso a organizaciones ni a su
   galería. El permiso SELECT existente de toda product_images incluye purpose.
6. La limpieza exitosa por Custom-Time se conserva a dos días y no elimina FAILED.

No se han creado ni modificado recursos de GCP mediante esta implementación.
IAM, CORS, entrega Eventarc y envío real de Meta requieren comprobación desplegada.
