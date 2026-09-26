UPDATE organization_modules
SET active=false,updated_at=now()
WHERE module_key IN (
  'call_center','carta_qr','kiosco','logistica','delivery','delivery_apps','repartidores',
  'crm','puntos','ofertas','personal','costos','bi','app_manager','facturacion','integraciones','whatsapp_bot'
)
  AND active;
