from decimal import Decimal

from src.domain.models import MenuItem, MenuResponse, TableContext
from src.services.welcome import MAX_REPLY_CHARS, welcome_message


def table(name: str = "MESA 7") -> TableContext:
    return TableContext(
        name=name,
        seats=4,
        zone="Salón",
        organizationName="La Panchita",
        locationName="Miraflores",
        conciergeEnabled=True,
    )


def item(
    name: str,
    category: str | None = "Fondos",
    *,
    recommended: bool = False,
    status: str = "available",
    price: str = "28",
) -> MenuItem:
    return MenuItem(
        productId=name.lower(),
        name=name,
        price=Decimal(price),
        categoryName=category,
        recommended=recommended,
        status=status,
    )


def test_greeting_does_not_repeat_the_table_word() -> None:
    reply = welcome_message(table("MESA 7"), None)

    assert reply == (
        "¡Bienvenido a La Panchita! 👋\nEstás en la MESA 7.\n\n¿Qué deseas pedir?"
    )
    assert "Estás en la Mesa 12." in welcome_message(table("12"), None)


def test_recommended_go_first_and_categories_keep_backend_order() -> None:
    menu = MenuResponse(
        currencySymbol="S/",
        items=[
            item("Lomo saltado", recommended=True, price="32.5"),
            item("Ají de gallina"),
            item("Chicha morada", "Bebidas", price="8"),
            item("Arroz con pollo"),
        ],
    )

    reply = welcome_message(table(), menu)

    assert reply == (
        "¡Bienvenido a La Panchita! 👋\nEstás en la MESA 7.\n\n"
        "*⭐ Recomendados de la casa*\n• Lomo saltado · S/ 32.50\n\n"
        "*Fondos*\n• Ají de gallina · S/ 28.00\n• Arroz con pollo · S/ 28.00\n\n"
        "*Bebidas*\n• Chicha morada · S/ 8.00\n\n"
        "¿Qué deseas pedir?"
    )


def test_unavailable_dishes_are_hidden_and_uncategorized_go_to_otros() -> None:
    menu = MenuResponse(
        currencySymbol="S/",
        items=[
            item("Ceviche", status="out_of_stock"),
            item("Causa", status="low"),
            item("Turrón", None),
        ],
    )

    reply = welcome_message(table(), menu)

    assert "Ceviche" not in reply
    assert "• Causa · S/ 28.00" in reply
    assert "*Otros*\n• Turrón" in reply


def test_empty_menu_sends_only_the_greeting() -> None:
    reply = welcome_message(table(), MenuResponse(currencySymbol="S/"))

    assert reply.endswith("Estás en la MESA 7.\n\n¿Qué deseas pedir?")


def test_long_menu_fits_whatsapp_and_counts_the_rest() -> None:
    menu = MenuResponse(
        currencySymbol="S/",
        items=[item(f"Plato especial de la casa número {n}") for n in range(200)],
    )

    reply = welcome_message(table(), menu)

    assert len(reply) <= MAX_REPLY_CHARS
    shown = reply.count("\n• ")
    assert 0 < shown < 200
    assert f"…y {200 - shown} platos más." in reply
    assert reply.endswith("¿Qué deseas pedir?")
