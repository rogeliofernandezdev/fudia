from __future__ import annotations

from typing import Any

TOOL_SCHEMAS: dict[str, dict[str, Any]] = {
    "search_menu": {
        "type": "function",
        "name": "search_menu",
        "description": (
            "Busca productos reales por nombre, descripción o categoría. "
            "Usa query vacío para consultar la carta disponible."
        ),
        "parameters": {
            "type": "object",
            "properties": {"query": {"type": "string"}},
            "required": ["query"],
            "additionalProperties": False,
        },
        "strict": True,
    },
    "get_modifier_options": {
        "type": "function",
        "name": "get_modifier_options",
        "description": (
            "Obtiene grupos y opciones reales de modificadores para un producto."
        ),
        "parameters": {
            "type": "object",
            "properties": {"productId": {"type": "string"}},
            "required": ["productId"],
            "additionalProperties": False,
        },
        "strict": True,
    },
    "get_combo_options": {
        "type": "function",
        "name": "get_combo_options",
        "description": "Obtiene grupos y alternativas reales de un combo.",
        "parameters": {
            "type": "object",
            "properties": {"productId": {"type": "string"}},
            "required": ["productId"],
            "additionalProperties": False,
        },
        "strict": True,
    },
    "add_item": {
        "type": "function",
        "name": "add_item",
        "description": (
            "Agrega al carrito un producto real sin configuraciones pendientes."
        ),
        "parameters": {
            "type": "object",
            "properties": {
                "productId": {"type": "string"},
                "quantity": {"type": "number", "minimum": 0.01},
                "note": {"type": "string"},
            },
            "required": ["productId", "quantity", "note"],
            "additionalProperties": False,
        },
        "strict": True,
    },
    "add_modified_item": {
        "type": "function",
        "name": "add_modified_item",
        "description": "Agrega un producto con modificadores reales validados.",
        "parameters": {
            "type": "object",
            "properties": {
                "productId": {"type": "string"},
                "quantity": {"type": "number", "minimum": 0.01},
                "note": {"type": "string"},
                "modifiers": {
                    "type": "array",
                    "items": {
                        "type": "object",
                        "properties": {
                            "groupId": {"type": "string"},
                            "optionId": {"type": "string"},
                        },
                        "required": ["groupId", "optionId"],
                        "additionalProperties": False,
                    },
                },
            },
            "required": ["productId", "quantity", "note", "modifiers"],
            "additionalProperties": False,
        },
        "strict": True,
    },
    "add_combo_item": {
        "type": "function",
        "name": "add_combo_item",
        "description": "Agrega un combo usando selecciones reales validadas.",
        "parameters": {
            "type": "object",
            "properties": {
                "productId": {"type": "string"},
                "quantity": {"type": "number", "minimum": 0.01},
                "note": {"type": "string"},
                "selections": {
                    "type": "array",
                    "items": {
                        "type": "object",
                        "properties": {
                            "groupId": {"type": "string"},
                            "productId": {"type": "string"},
                        },
                        "required": ["groupId", "productId"],
                        "additionalProperties": False,
                    },
                },
            },
            "required": ["productId", "quantity", "note", "selections"],
            "additionalProperties": False,
        },
        "strict": True,
    },
    "remove_item": {
        "type": "function",
        "name": "remove_item",
        "description": "Retira un producto del carrito.",
        "parameters": {
            "type": "object",
            "properties": {"productId": {"type": "string"}},
            "required": ["productId"],
            "additionalProperties": False,
        },
        "strict": True,
    },
    "view_cart": {
        "type": "function",
        "name": "view_cart",
        "description": "Consulta el carrito temporal actual.",
        "parameters": {
            "type": "object",
            "properties": {},
            "additionalProperties": False,
        },
        "strict": True,
    },
    "prepare_confirmation": {
        "type": "function",
        "name": "prepare_confirmation",
        "description": "Prepara el resumen que el cliente debe confirmar.",
        "parameters": {
            "type": "object",
            "properties": {},
            "additionalProperties": False,
        },
        "strict": True,
    },
    "confirm_order": {
        "type": "function",
        "name": "confirm_order",
        "description": "Confirma un envío a cocina tras confirmación explícita.",
        "parameters": {
            "type": "object",
            "properties": {},
            "additionalProperties": False,
        },
        "strict": True,
    },
    "request_bill": {
        "type": "function",
        "name": "request_bill",
        "description": "Obtiene la cuenta real de la mesa desde foods-backend.",
        "parameters": {
            "type": "object",
            "properties": {},
            "additionalProperties": False,
        },
        "strict": True,
    },
    "request_human": {
        "type": "function",
        "name": "request_human",
        "description": "Solicita atención humana real para la mesa.",
        "parameters": {
            "type": "object",
            "properties": {
                "reason": {
                    "type": "string",
                    "description": (
                        "Motivo breve basado en lo dicho por el cliente."
                    ),
                }
            },
            "required": ["reason"],
            "additionalProperties": False,
        },
        "strict": True,
    },
    "get_delivery_options": {
        "type": "function",
        "name": "get_delivery_options",
        "description": (
            "Consulta si el local ofrece recojo y delivery, la dirección del "
            "local y los rangos de costo de envío por distancia."
        ),
        "parameters": {
            "type": "object",
            "properties": {},
            "additionalProperties": False,
        },
        "strict": True,
    },
    "set_fulfillment": {
        "type": "function",
        "name": "set_fulfillment",
        "description": (
            "Registra si el cliente recogerá el pedido en el local (pickup) "
            "o si se le enviará con el delivery del local (delivery)."
        ),
        "parameters": {
            "type": "object",
            "properties": {
                "type": {
                    "type": "string",
                    "enum": ["pickup", "delivery"],
                    "description": "pickup = recojo en local; delivery = envío.",
                }
            },
            "required": ["type"],
            "additionalProperties": False,
        },
        "strict": True,
    },
    "set_delivery_details": {
        "type": "function",
        "name": "set_delivery_details",
        "description": (
            "Guarda el nombre del cliente y, para delivery, la dirección "
            "escrita y la referencia. Usa texto vacío para no cambiar un dato."
        ),
        "parameters": {
            "type": "object",
            "properties": {
                "customerName": {
                    "type": "string",
                    "description": "Nombre de quien recibe o recoge.",
                },
                "address": {
                    "type": "string",
                    "description": "Dirección exacta escrita por el cliente.",
                },
                "reference": {
                    "type": "string",
                    "description": "Referencia para el repartidor.",
                },
            },
            "required": ["customerName", "address", "reference"],
            "additionalProperties": False,
        },
        "strict": True,
    },
    "check_order_status": {
        "type": "function",
        "name": "check_order_status",
        "description": "Consulta el estado real del último pedido confirmado.",
        "parameters": {
            "type": "object",
            "properties": {},
            "additionalProperties": False,
        },
        "strict": True,
    },
}

MENU_TOOL_SCHEMAS = [
    TOOL_SCHEMAS[name]
    for name in (
        "search_menu",
        "get_modifier_options",
        "get_combo_options",
        "get_delivery_options",
    )
]
ORDER_TOOL_SCHEMAS = [
    TOOL_SCHEMAS[name]
    for name in (
        "search_menu",
        "get_modifier_options",
        "get_combo_options",
        "add_item",
        "add_modified_item",
        "add_combo_item",
        "remove_item",
        "view_cart",
        "prepare_confirmation",
        "confirm_order",
        "get_delivery_options",
        "set_fulfillment",
        "set_delivery_details",
    )
]
SERVICE_TOOL_SCHEMAS = [
    TOOL_SCHEMAS[name]
    for name in (
        "request_bill",
        "request_human",
        "check_order_status",
    )
]
