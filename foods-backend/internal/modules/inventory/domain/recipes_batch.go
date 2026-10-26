package domain

// QuantityProduct es el producto bloqueado al aplicar el consumo de un pedido.
type QuantityProduct struct {
	Name    string
	Control string
}

// ActiveRecipe es la receta activa de un producto y su rendimiento.
type ActiveRecipe struct {
	ID    string
	Yield float64
}
