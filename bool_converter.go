package generator

import (
	"reflect"

	"github.com/dave/jennifer/jen"
)

// BoolConverter knows how to convert bool and *bool
type BoolConverter struct{}

var _ AttributeConverter = &BoolConverter{}

func (c *BoolConverter) Check(typ reflect.Type) (bool, error) {
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	return typ.Kind() == reflect.Bool, nil
}

func (c *BoolConverter) GetFrameworkType(_ *Converter, typ reflect.Type) (*jen.Statement, error) {
	return jen.Qual("github.com/hashicorp/terraform-plugin-framework/types", "Bool"), nil
}

func (c *BoolConverter) Decode(converters *Converter, field *FieldInformation, path, src, target *jen.Statement, typ reflect.Type) (*jen.Statement, error) {
	method := "ValueBool"
	if typ.Kind() == reflect.Pointer {
		method = "ValueBoolPointer"
	}

	return decode(src, target.Op("=").Add(src.Clone()).Dot(method).Call())
}

func (c *BoolConverter) Encode(converters *Converter, field *FieldInformation, src, target *jen.Statement, typ reflect.Type) (*jen.Statement, error) {
	method := "BoolValue"
	if typ.Kind() == reflect.Pointer {
		method = "BoolPointerValue"
	}
	return target.Op("=").Qual("github.com/hashicorp/terraform-plugin-framework/types", method).Call(src), nil
}

func (c *BoolConverter) GetSchema(converters *Converter, path string, info *FieldInformation) (*jen.Statement, *jen.Statement, error) {
	return basicSchema(converters.SchemaImportPath(), "BoolAttribute", info, nil)
}

func (c *BoolConverter) GetType() *jen.Statement {
	return jen.Qual("github.com/hashicorp/terraform-plugin-framework/types", "BoolType")
}
