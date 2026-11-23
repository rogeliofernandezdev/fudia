-- «Fudia Concierge» deja de ser una pantalla: el restaurante no configura nada
-- del canal (lo administra la plataforma por país). El módulo `whatsapp_bot`
-- sigue habilitando los pedidos desde el QR; solo se limpia la casilla de menú.
UPDATE roles
SET menu_access=array_remove(menu_access,'whatsapp_bot')
WHERE 'whatsapp_bot'=ANY(menu_access);
