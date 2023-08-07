# terraform-plugin-generator

`terraform-plugin-generator` is a library that makes it possible to generate
[Terraform Plugin Framework](https://developer.hashicorp.com/terraform/plugin/framework)
models and schemas based on Go types.

Using `terraform-plugin-generator` you can reuse the types of your API Go client
by adding tag annotations to your structs definition:

```go
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
}

type Ingredient struct {
    ID int `terraform:"id,required"`
}
```

Once the Go code of your client is properly types you can generate the schemas
and models for your Terraform automatically:

```go
package main

import (
	"fmt"
	"log"
	"reflect"
	"strings"

	"github.com/YourProject/api"
	generator "github.com/Lenstra/github-terraform-generator"
)

func main() {
	generators := []generator.Generator{
		&generator.ModelGenerator{
			Package: "models",
			Path:    "./internal/models",
			Objects: map[string]interface{}{
				"Coffee": api.Coffee{},
			},
			Logger: logger,
		},
		&generator.SchemaGenerator{
			Type:                generator.DataSourceSchema,
			Package:             "datasource",
			Path:                "./internal/datasource",
			Logger:              logger,
			Objects: map[string]interface{}{
                "coffee": api.Coffee{},
			},
		},
		&generator.SchemaGenerator{
			Type:                generator.ResourceSchema,
			Package:             "resource",
			Path:                "./internal/resource",
			Objects: map[string]interface{}{
				"coffee": api.Coffee{},
			},
		},
		&generator.SchemaGenerator{
			Type:    generator.ProviderSchema,
			Package: "provider",
			Path:    "./internal/provider",
			Logger:  logger,
			Objects: map[string]interface{}{
				"config": api.Config{},
			},
		},
	}

	for _, generator := range generators {
		if err := generator.Render(); err != nil {
			log.Fatal(err.Error())
		}
	}
}
```

and the files `./internal/models/models.go`, `./internal/models/encoders.go`,
`./internal/models/decoders.go`, `./internal/datasource/schema.go`,
`./internal/resource/schema.go`, `./internal/provider/schema.go` will be generated
with code ready to be used in your Terraform provider.
