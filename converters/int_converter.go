package converters

import (
	"fmt"
	"reflect"

	"github.com/Lenstra/terraform-plugin-generator/tags"
	"github.com/dave/jennifer/jen"
)

// IntConverter knows how to convert int, int8, int16, int32, int64, uint,
// uint8, uint16, uint32, uint64, *int, *int8, *int16, *int32, *int64, *uint,
// *uint8, *uint16, *uint32, *uint64
type IntConverter struct{}

var _ AttributeConverter = &IntConverter{}

func (c *IntConverter) Check(typ reflect.Type) (bool, error) {
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	switch typ.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32,
		reflect.Uint64:
		return true, nil
	default:
		return false, nil
	}
}

func (c *IntConverter) GetFrameworkType(_ *Converter, typ reflect.Type) (*jen.Statement, error) {
	return jen.Qual("github.com/hashicorp/terraform-plugin-framework/types", "Int64"), nil
}

func (c *IntConverter) Decode(converters *Converter, path, src, target *jen.Statement, typ reflect.Type) (*jen.Statement, error) {
	op := jen.Empty()
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
		op = jen.Op("&")
	}
	code := src.Clone().Dot("ValueInt64").Call()
	switch typ.Kind() {
	case reflect.Int:
		code = jen.Int().Call(code)
	case reflect.Int8:
		code = jen.Int8().Call(code)
	case reflect.Int16:
		code = jen.Int16().Call(code)
	case reflect.Int32:
		code = jen.Int32().Call(code)
	case reflect.Int64:
		break
	case reflect.Uint:
		code = jen.Uint().Call(code)
	case reflect.Uint8:
		code = jen.Uint8().Call(code)
	case reflect.Uint16:
		code = jen.Uint16().Call(code)
	case reflect.Uint32:
		code = jen.Uint32().Call(code)
	case reflect.Uint64:
		code = jen.Uint64().Call(code)
	default:
		return nil, fmt.Errorf("unexpected type %s", typ.Name())
	}

	return jen.If().Op("!").Add(src.Clone()).Dot("IsNull").Call().Block(
		jen.Id("i").Op(":=").Add(code),
		target.Op("=").Add(op).Id("i"),
	), nil
}

func (c *IntConverter) Encode(converters *Converter, src, target *jen.Statement, typ reflect.Type) (*jen.Statement, error) {
	ptr := false
	if typ.Kind() == reflect.Pointer {
		ptr = true
		typ = typ.Elem()
	}

	// Use framework functions if no convertion is needed
	if typ.Kind() == reflect.Int64 {
		method := "Int64Value"
		if ptr {
			method = "Int64PointerValue"
		}
		return target.Op("=").Qual("github.com/hashicorp/terraform-plugin-framework/types", method).Call(src), nil
	}

	value := jen.Empty()
	code := target.Op("=").Qual("github.com/hashicorp/terraform-plugin-framework/types", "Int64Value").Call(value)

	// Convert the src
	if ptr {
		value.Add(jen.Int64().Call(jen.Op("*").Add(src)))
		return jen.If(src.Clone().Op("!=").Nil()).Block(code), nil
	}

	value.Add(jen.Int64().Call(src))
	return code, nil
}

func (c *IntConverter) GetSchema(converters *Converter, path string, info *tags.FieldInformation) (*jen.Statement, *jen.Statement, error) {
	return basicSchema(converters.SchemaImportPath(), "Int64Attribute", info, nil)
}

func (c *IntConverter) GetType() *jen.Statement {
	return jen.Qual("github.com/hashicorp/terraform-plugin-framework/types", "Int64Type")
}
