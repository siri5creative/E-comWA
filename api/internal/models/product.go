package models

// Money amounts are represented as int64 whole Rupiah — IDR has no
// subunit in everyday use, and every example amount in the PRD/API docs is
// a whole number (e.g. 85000), so this avoids floating point entirely.

type Category struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type ProductVariant struct {
	ID            string  `json:"id"`
	VariantName   string  `json:"variant_name"`
	SKU           *string `json:"sku"`
	Price         int64   `json:"price"`
	StockQuantity int32   `json:"stock_quantity"`
}

type Product struct {
	ID            string           `json:"id"`
	Name          string           `json:"name"`
	Slug          string           `json:"slug"`
	Description   *string          `json:"description"`
	CoverImageURL *string          `json:"cover_image_url"`
	Category      *Category        `json:"category"`
	Variants      []ProductVariant `json:"variants"`
}
