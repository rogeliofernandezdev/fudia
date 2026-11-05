UPDATE roles SET menu_access = array_remove(array_remove(array_remove(array_remove(array_remove(
  menu_access,'ventas'),'comprobantes'),'salon'),'mi_restaurante'),'medios_pago')
WHERE menu_access && ARRAY['ventas','comprobantes','salon','mi_restaurante','medios_pago'];
