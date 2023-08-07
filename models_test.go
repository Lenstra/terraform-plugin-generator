package generator

import (
	"testing"

	"github.com/Lenstra/terraform-plugin-generator/tests/structs"
	"github.com/stretchr/testify/require"
)

func TestModels(t *testing.T) {
	objects := map[string]interface{}{
		"Config":     structs.Config{},
		"Coffee":     structs.Coffee{},
		"Ingredient": structs.Ingredient{},
	}
	err := GenerateModels("./tests/", "tests", objects, &GeneratorOptions{})
	require.NoError(t, err)
}
