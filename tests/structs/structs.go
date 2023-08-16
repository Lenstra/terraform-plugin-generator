package structs

type Config struct {
	Host           string         `terraform:"host,required"`
	PromotedBool   PromotedBool   `terraform:"-,promoted"`
	PromotedInt    PromotedInt    `terraform:"-,promoted"`
	PromotedString PromotedString `terraform:"-,promoted"`
}

type PromotedBool struct {
	Bool bool `terraform:"bool"`
}

type PromotedInt struct {
	Int int `terraform:"int"`
}

type PromotedString struct {
	String string `terraform:"string"`
}

type Coffee struct {
	ID          int          `terraform:"id"`
	Name        string       `terraform:"name,required"`
	Teaser      string       `terraform:"teaser"`
	Description string       `terraform:"description"`
	Image       string       `terraform:"image"`
	Ingredients []Ingredient `terraform:"ingredients"`
	Customer    *Customer    `terraform:"customer"`
}

type Ingredient struct {
	ID      int     `terraform:"id,required"`
	Float32 float32 `terraform:"float32"`
	Float64 float64 `terraform:"float64"`
}

type Customer struct {
	ID   int64  `terraform:"id"`
	Name string `terraform:"name"`
}
