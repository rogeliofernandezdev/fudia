from decimal import Decimal

import pytest

from src.domain.models import (
    ComboDetail,
    ComboGroup,
    ComboOption,
    MenuItem,
    MenuResponse,
    TableContext,
)
from src.services.carta import Carta, build_carta
from src.services.ports import FudiaError
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
    combo: bool = False,
) -> MenuItem:
    return MenuItem(
        productId=name.lower(),
        name=name,
        price=Decimal(price),
        categoryName=category,
        recommended=recommended,
        status=status,
        isCombo=combo,
    )


def carta(*items: MenuItem) -> Carta:
    return Carta(currency_symbol="S/", items=list(items))


def option(name: str, surcharge: str = "0", available: bool = True) -> ComboOption:
    return ComboOption(
        productId=name.lower(),
        name=name,
        surcharge=Decimal(surcharge),
        available=available,
    )


class ComboFudia:
    def __init__(self, fail: bool = False) -> None:
        self.fail = fail

    async def get_combo(self, token: str, product_id: str) -> ComboDetail:
        if self.fail:
            raise FudiaError(409, "combo_unavailable", "Sin alternativas.")
        return ComboDetail(
            id=product_id,
            name="Menú",
            price=Decimal("12"),
            groups=[
                ComboGroup(
                    id="g1",
                    name="Entrada",
                    required=True,
                    minSelections=1,
                    maxSelections=1,
                    options=[option("Causa Rellena")],
                ),
                ComboGroup(
                    id="g2",
                    name="Segundo",
                    required=True,
                    minSelections=1,
                    maxSelections=1,
                    options=[
                        option("Aji de gallina"),
                        option("Lomo Saltado", "3"),
                        option("Seco", available=False),
                    ],
                ),
            ],
        )


LA_PANCHITA = MenuResponse(
    currencySymbol="S/",
    items=[
        item("Causa Rellena", "Entradas", price="20"),
        item("Ceviche", "Marino", price="25"),
        item("Aji de gallina", "Segundos", price="15"),
        item("Lomo Saltado", "Segundos", price="15"),
        item("Menú", "Combos", price="12", combo=True),
    ],
)


def test_greeting_does_not_repeat_the_table_word() -> None:
    reply = welcome_message(table("MESA 7"), None)

    assert reply == "🍽️ Bienvenido a La Panchita\n📍 MESA 7\n\n¿Qué deseas pedir?"
    assert "📍 Mesa 12\n" in welcome_message(table("12"), None)


def test_recommended_go_first_and_categories_keep_backend_order() -> None:
    reply = welcome_message(
        table(),
        carta(
            item("Lomo saltado", recommended=True, price="32.5"),
            item("Ají de gallina"),
            item("Chicha morada", "Bebidas", price="8"),
            item("Arroz con pollo"),
        ),
    )

    assert reply == (
        "🍽️ Bienvenido a La Panchita\n📍 MESA 7\n\n"
        "Recomendados de la casa:\n- Lomo saltado: S/ 32.50\n\n"
        "Fondos:\n- Ají de gallina: S/ 28.00\n- Arroz con pollo: S/ 28.00\n\n"
        "Bebidas:\n- Chicha morada: S/ 8.00\n\n"
        "¿Qué deseas pedir?"
    )
    assert "*" not in reply


@pytest.mark.asyncio
async def test_combo_shows_what_it_includes_and_its_dishes_are_not_repeated() -> None:
    full = await build_carta(ComboFudia(), "a" * 32, LA_PANCHITA)

    reply = welcome_message(table(), full)

    assert reply == (
        "🍽️ Bienvenido a La Panchita\n📍 MESA 7\n\n"
        "Menú (S/ 12.00):\n"
        "- Entrada: Causa Rellena\n"
        "- Segundo: Aji de gallina o Lomo Saltado (+S/ 3.00)\n\n"
        "Marino:\n- Ceviche: S/ 25.00\n\n"
        "¿Qué deseas pedir?"
    )
    assert full.included == {"causa rellena", "aji de gallina", "lomo saltado"}


@pytest.mark.asyncio
async def test_combo_without_detail_stays_a_plain_dish() -> None:
    full = await build_carta(ComboFudia(fail=True), "a" * 32, LA_PANCHITA)

    reply = welcome_message(table(), full)

    assert "Combos:\n- Menú: S/ 12.00" in reply
    assert "Segundos:\n- Aji de gallina: S/ 15.00" in reply
    assert full.included == set()


@pytest.mark.asyncio
async def test_sold_out_dishes_are_left_out_and_uncategorized_go_to_otros() -> None:
    menu = MenuResponse(
        currencySymbol="S/",
        items=[
            item("Ceviche", "Marino", status="sold_out"),
            item("Causa", status="low"),
            item("Turrón", None),
        ],
    )

    reply = welcome_message(table(), await build_carta(ComboFudia(), "t", menu))

    assert reply == (
        "🍽️ Bienvenido a La Panchita\n📍 MESA 7\n\n"
        "Fondos:\n- Causa: S/ 28.00\n\n"
        "Otros:\n- Turrón: S/ 28.00\n\n"
        "¿Qué deseas pedir?"
    )
    assert "Ceviche" not in reply and "gotado" not in reply


@pytest.mark.asyncio
async def test_sold_out_combo_is_left_out_and_its_dishes_stay_in_their_categories() -> None:
    menu = MenuResponse(
        currencySymbol="S/",
        items=[
            item("Causa Rellena", "Entradas", price="20"),
            item("Lomo Saltado", "Segundos", price="15"),
            item("Menú", "Combos", price="12", combo=True, status="sold_out"),
        ],
    )

    reply = welcome_message(table(), await build_carta(ComboFudia(), "t", menu))

    assert "Entradas:\n- Causa Rellena: S/ 20.00" in reply
    assert "Segundos:\n- Lomo Saltado: S/ 15.00" in reply
    assert "Menú" not in reply


@pytest.mark.asyncio
async def test_nothing_available_says_there_are_no_dishes_today() -> None:
    menu = MenuResponse(
        currencySymbol="S/",
        items=[
            item("Causa Rellena", "Entradas", status="sold_out"),
            item("Ceviche", "Marino", status="sold_out"),
            item("Menú", "Combos", combo=True, status="sold_out"),
        ],
    )

    reply = welcome_message(table(), await build_carta(ComboFudia(), "t", menu))

    assert reply == (
        "🍽️ Bienvenido a La Panchita\n📍 MESA 7\n\n"
        "Hoy no tenemos platos disponibles.\n\n"
        "Si necesitas ayuda, puedo avisar al personal del local."
    )
    assert welcome_message(table(), carta()) == reply


def test_unreadable_carta_sends_only_the_greeting() -> None:
    reply = welcome_message(table(), None)

    assert reply == "🍽️ Bienvenido a La Panchita\n📍 MESA 7\n\n¿Qué deseas pedir?"


def test_long_carta_fits_whatsapp_and_counts_the_rest() -> None:
    reply = welcome_message(
        table(),
        carta(*(item(f"Plato especial de la casa número {n}") for n in range(200))),
    )

    assert len(reply) <= MAX_REPLY_CHARS
    shown = reply.count("\n- ")
    assert 0 < shown < 200
    assert f"Hay {200 - shown} opciones más." in reply
    assert reply.endswith("¿Qué deseas pedir?")
