package generator

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/dave/jennifer/jen"
)

type FieldInformation struct {
	Name        string
	Path        string
	Optional    bool
	Required    bool
	Computed    bool
	Sensitive   bool
	Description string
	Block       bool
	Default     *jen.Statement

	Promoted bool
	Parent   *FieldInformation

	// Go data
	goName   string
	goType   reflect.Type
	accessor *jen.Statement
}

type FieldInformationGetter func(string, reflect.StructField) (*FieldInformation, error)

func GetFieldInformationFromTerraformTag(_ string, field reflect.StructField) (*FieldInformation, error) {
	tag, ok := field.Tag.Lookup("terraform")
	if !ok {
		return nil, nil
	}

	modifiers := map[string]struct{}{}
	values := strings.Split(tag, ",")
	name, values := values[0], values[1:]

	result := &FieldInformation{
		Name:     name,
		goName:   field.Name,
		goType:   field.Type,
		accessor: jen.Dot(field.Name),
	}

	for _, v := range values {
		switch v {
		case "sensitive":
			if _, found := modifiers["sensitive"]; found {
				return nil, fmt.Errorf("sensitive modifier given multiple time")
			}
			modifiers["sensitive"] = struct{}{}
			result.Sensitive = true
		case "promoted":
			if _, found := modifiers["promoted"]; found {
				return nil, fmt.Errorf("promoted modifier given multiple time")
			}
			modifiers["promoted"] = struct{}{}
			if result.Name != "-" {
				return nil, fmt.Errorf(`the name must be "-" when a field is promoted`)
			}
			result.Promoted = true
			result.Name = ""
		case "optional":
			if _, found := modifiers["optional"]; found {
				return nil, fmt.Errorf("optional modifier given multiple time")
			}
			modifiers["optional"] = struct{}{}
			result.Optional = true
		case "required":
			if _, found := modifiers["required"]; found {
				return nil, fmt.Errorf("required modifier given multiple time")
			}
			modifiers["required"] = struct{}{}
			result.Required = true
		case "computed":
			if _, found := modifiers["computed"]; found {
				return nil, fmt.Errorf("computed modifier given multiple time")
			}
			modifiers["computed"] = struct{}{}
			result.Computed = true
		case "block":
			if _, found := modifiers["block"]; found {
				return nil, fmt.Errorf("block modifier given multiple time")
			}
			modifiers["block"] = struct{}{}
			result.Block = true
		default:
			return nil, fmt.Errorf("unknown modifier %q", v)
		}
	}

	if !result.Required && !result.Computed {
		result.Optional = true
	}

	return result, nil
}
