package gbp

// ProductFamily categorises products.
type ProductFamily string

const (
	FamilySavings ProductFamily = "Savings"
	FamilyLending ProductFamily = "Lending"
)

// Product is a named composition of features with default parameters.
type Product struct {
	ID       string
	Name     string
	Family   ProductFamily
	Features []Feature
	Defaults map[string]string // e.g. {"annual_rate": "0.015"}
}
