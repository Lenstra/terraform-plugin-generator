package generator

import (
	"fmt"
	"reflect"

	"github.com/dave/jennifer/jen"
)

// FloatConverter knows how to convert float32, float64, *float32 and *float64
type FloatConverter struct{}

var _ AttributeConverter = &FloatConverter{}

func (c *FloatConverter) Check(typ reflect.Type) (bool, error) {
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	switch typ.Kind() {
	case reflect.Float32, reflect.Float64:
		return true, nil
	default:
		return false, nil
	}
}

func (c *FloatConverter) GetFrameworkType(_ *Converter, typ reflect.Type) (*jen.Statement, error) {
	return jen.Qual("github.com/hashicorp/terraform-plugin-framework/types", "Float64"), nil
}

func (c *FloatConverter) Decode(converters *Converter, field *FieldInformation, path, src, target *jen.Statement, typ reflect.Type) (*jen.Statement, error) {
	op := jen.Empty()
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
		op = jen.Op("&")
	}
	code := src.Clone().Dot("ValueFloat64").Call()
	switch typ.Kind() {
	case reflect.Float32:
		code = jen.Float32().Call(code)
	case reflect.Float64:
		break
	default:
		return nil, fmt.Errorf("unexpected type %s", typ.Name())
	}

	return decode(src, jen.Id("i").Op(":=").Add(code).Line().Add(target.Op("=").Add(op).Id("i")))
}

func (c *FloatConverter) Encode(converters *Converter, field *FieldInformation, src, target *jen.Statement, typ reflect.Type) (*jen.Statement, error) {
	ptr := false
	if typ.Kind() == reflect.Pointer {
		ptr = true
		typ = typ.Elem()
	}

	// Use framework functions if no convertion is needed
	if typ.Kind() == reflect.Int64 {
		method := "Float64Value"
		if ptr {
			method = "Float64PointerValue"
		}
		return target.Op("=").Qual("github.com/hashicorp/terraform-plugin-framework/types", method).Call(src), nil
	}

	value := jen.Empty()
	code := target.Op("=").Qual("github.com/hashicorp/terraform-plugin-framework/types", "Float64Value").Call(value)

	// Convert the src
	if ptr {
		value.Add(jen.Float64().Call(jen.Op("*").Add(src)))
		return jen.If(src.Clone().Op("!=").Nil()).Block(code), nil
	}

	value.Add(jen.Float64().Call(src))
	return code, nil
}

func (c *FloatConverter) GetSchema(converters *Converter, path string, info *FieldInformation) (*jen.Statement, *jen.Statement, error) {
	return basicSchema(converters.SchemaImportPath(), "Float64Attribute", info, nil)
}

func (c *FloatConverter) GetType() *jen.Statement {
	return jen.Qual("github.com/hashicorp/terraform-plugin-framework/types", "Float64Type")
}
