package database

import "regexp"

var uuidText = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// noMatchUUID nunca lo genera gen_random_uuid() (versión 4), así que una
// comparación con él no devuelve filas.
const noMatchUUID = "00000000-0000-0000-0000-000000000000"

// UUIDArg prepara un identificador recibido como texto para compararlo con una
// columna uuid (`col=$n::uuid`) y no con `col::text=$n`: convertir la columna a
// texto impide usar su índice y obliga a recorrer la tabla. Un valor vacío llega
// como NULL, de modo que los filtros opcionales se escriben
// `($n::uuid IS NULL OR col=$n::uuid)`; uno con formato inválido se compara con un
// uuid que no existe, igual que antes no coincidía ningún texto, en vez de
// provocar un error de conversión.
func UUIDArg(value string) any {
	if value == "" {
		return nil
	}
	if !uuidText.MatchString(value) {
		return noMatchUUID
	}
	return value
}
