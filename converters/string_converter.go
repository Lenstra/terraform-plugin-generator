package converters

import (
	"fmt"
	"reflect"
	"time"

	"github.com/Lenstra/terraform-plugin-generator/tags"
	"github.com/dave/jennifer/jen"
)

// StringConverter knows how to convert:
//   - string, *string and all of the type aliased to string
//   - []byte
//   - time.Time, *time.Time, time.Duration, *time.Duration
type StringConverter struct{}

var _ AttributeConverter = &StringConverter{}

type stringValueType int

const (
	invalidStringType stringValueType = iota
	byteType          stringValueType = iota
	stringType        stringValueType = iota
	timeType          stringValueType = iota
)

func getStringType(typ reflect.Type) stringValueType {
	if typ.String() == "[]uint8" {
		return byteType
	}
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	if typ == reflect.TypeOf(time.Duration(0)) || typ == reflect.TypeOf(time.Time{}) {
		return timeType
	}
	if typ.Kind() == reflect.String {
		return stringType
	}
	return invalidStringType
}

func (c *StringConverter) Check(typ reflect.Type) (bool, error) {
	return getStringType(typ) != invalidStringType, nil
}

func (c *StringConverter) GetFrameworkType(_ *Converter, typ reflect.Type) (*jen.Statement, error) {
	return jen.Qual("github.com/hashicorp/terraform-plugin-framework/types", "String"), nil
}

func (c *StringConverter) Decode(converters *Converter, path, src, target *jen.Statement, typ reflect.Type) (*jen.Statement, error) {
	switch getStringType(typ) {
	case byteType:
		return decodeBytes(converters, path, src, target, typ)
	case stringType:
		return decodeString(converters, path, src, target, typ)
	case timeType:
		return decodeTime(converters, path, src, target, typ)
	}
	return nil, fmt.Errorf("invalid string type")
}

func decodeString(converters *Converter, path, src, target *jen.Statement, typ reflect.Type) (*jen.Statement, error) {
	method := "ValueString"
	if typ.Kind() == reflect.Pointer {
		method = "ValueStringPointer"
		typ = typ.Elem()
	}

	value := src.Clone().Dot(method).Call()
	if typ != reflect.TypeOf("") {
		// Taking care of the aliases
		value = jen.Qual(typ.PkgPath(), typ.Name()).Call(value)
	}

	return jen.If().Op("!").Add(src.Clone()).Dot("IsNull").Call().Block(
		target.Op("=").Add(value),
	), nil
}

func decodeBytes(converters *Converter, path, src, target *jen.Statement, typ reflect.Type) (*jen.Statement, error) {
	return jen.If().Op("!").Add(src.Clone()).Dot("IsNull").Call().Block(
		target.Op("=").Index().Byte().Call(src.Clone().Dot("ValueString").Call()),
	), nil
}

func decodeTime(converters *Converter, path, src, target *jen.Statement, typ reflect.Type) (*jen.Statement, error) {
	op := jen.Empty()
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
		op = jen.Op("&")
	}

	ident := "dur"
	parseFunc := jen.Qual("time", "ParseDuration").Call(src.Clone().Dot("ValueString").Call())
	errMessage := "failed to parse duration"
	if typ == reflect.TypeOf(time.Time{}) {
		ident = "t"
		parseFunc = jen.Qual("time", "Parse").Call(jen.Qual("time", "RFC3339"), src.Clone().Dot("ValueString").Call())
		errMessage = "failed to parse time string"
	}

	return jen.If().Op("!").Add(src.Clone()).Dot("IsNull").Call().Block(
		jen.List(jen.Id(ident), jen.Id("err")).Op(":=").Add(parseFunc),
		jen.If(jen.Id("err").Op("!=").Nil()).Block(
			jen.Id("diags").Dot("Append").Call(
				jen.Qual("github.com/hashicorp/terraform-plugin-framework/diag", "NewAttributeErrorDiagnostic").Call(
					path,
					jen.Lit(errMessage),
					jen.Id("err").Dot("Error").Call(),
				),
			),
		),
		target.Op("=").Add(op).Id(ident),
	), nil
}

func (c *StringConverter) Encode(converters *Converter, src, target *jen.Statement, typ reflect.Type) (*jen.Statement, error) {
	switch getStringType(typ) {
	case byteType:
		return encodeBytes(converters, src, target, typ)
	case stringType:
		return encodeString(converters, src, target, typ)
	case timeType:
		return encodeTime(converters, src, target, typ)
	}
	return nil, fmt.Errorf("invalid string type")
}

func encodeBytes(converters *Converter, src, target *jen.Statement, typ reflect.Type) (*jen.Statement, error) {
	return encodeString(converters, src, target, typ)
}

func encodeString(converters *Converter, src, target *jen.Statement, typ reflect.Type) (*jen.Statement, error) {
	ptr := false
	value := src.Clone()
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
		ptr = true
		value = jen.Op("*").Add(src)
	}

	// If it is a raw string we use the standard functions to do the conversion
	if typ == reflect.TypeOf("") {
		method := "StringValue"
		if ptr {
			method = "StringPointerValue"
		}
		return target.Op("=").Qual("github.com/hashicorp/terraform-plugin-framework/types", method).Call(src), nil
	}

	// Taking care of the aliases
	value = jen.String().Call(value)

	code := target.Op("=").Qual("github.com/hashicorp/terraform-plugin-framework/types", "StringValue").Call(value)

	if ptr {
		return jen.If(src.Clone().Op("!=").Nil()).Block(code), nil
	}

	return code, nil
}

func encodeTime(converters *Converter, src, target *jen.Statement, typ reflect.Type) (*jen.Statement, error) {
	ptr := false
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
		ptr = true
	}

	var code *jen.Statement
	if typ == reflect.TypeOf(time.Time{}) {
		code = target.Op("=").Qual("github.com/hashicorp/terraform-plugin-framework/types", "StringValue").Call(src.Clone().Dot("Format").Call(jen.Qual("time", "RFC3339")))
	} else {
		code = target.Op("=").Qual("github.com/hashicorp/terraform-plugin-framework/types", "StringValue").Call(src.Clone().Dot("String").Call())
	}

	if ptr {
		return jen.If(src.Clone().Op("!=").Nil()).Block(code), nil
	}

	return code, nil
}

func (c *StringConverter) GetSchema(converters *Converter, path string, info *tags.FieldInformation) (*jen.Statement, *jen.Statement, error) {
	return basicSchema(converters.SchemaImportPath(), "StringAttribute", info, nil)
}

func (c *StringConverter) GetType() *jen.Statement {
	return jen.Qual("github.com/hashicorp/terraform-plugin-framework/types", "StringType")
}
