package generator

import (
	"reflect"
	"testing"

	"github.com/Lenstra/terraform-plugin-generator/tests/structs"
	"github.com/dave/jennifer/jen"
	"github.com/stretchr/testify/require"
)

func TestModels(t *testing.T) {
	objects := map[string]interface{}{
		"Config":     structs.Config{},
		"Coffee":     structs.Coffee{},
		"Ingredient": structs.Ingredient{},
	}
	err := GenerateModels("./tests/", "tests", objects, &GeneratorOptions{
		GetFieldInformation: func(s string, typ reflect.Type, sf reflect.StructField) (*FieldInformation, error) {
			info, err := GetFieldInformationFromTerraformTag(s, typ, sf)
			if info == nil || err != nil {
				return info, err
			}
			info.Default = jen.Nil()
			return info, nil
		},
	})
	require.NoError(t, err)
}

func TestSchema(t *testing.T) {
	objects := map[string]interface{}{
		"Config":     structs.Config{},
		"Coffee":     structs.Coffee{},
		"Ingredient": structs.Ingredient{},
	}
	err := GenerateSchema(ResourceSchema, "./tests/", "tests", objects, &GeneratorOptions{
		GetFieldInformation: func(s string, typ reflect.Type, sf reflect.StructField) (*FieldInformation, error) {
			info, err := GetFieldInformationFromTerraformTag(s, typ, sf)
			if info == nil || err != nil {
				return info, err
			}
			info.Default = jen.Nil()
			info.Validators = jen.Nil()
			return info, nil
		},
	})
	require.NoError(t, err)
}
