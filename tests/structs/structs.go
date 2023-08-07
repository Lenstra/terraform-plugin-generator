package structs

type Config struct {
	Host     string `terraform:"host,required"`
	Username string `terraform:"username"`
	Password string `terraform:"password,sensitive"`
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
	ID int `terraform:"id,required"`
}

type Customer struct {
	ID   int64  `terraform:"id"`
	Name string `terraform:"name"`
}
