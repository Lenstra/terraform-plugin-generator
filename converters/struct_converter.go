package converters

import (
	"fmt"
	"reflect"

	"github.com/Lenstra/terraform-plugin-generator/tags"
	"github.com/dave/jennifer/jen"
)

// StructConverter knows how to convert struct and *struct
type StructConverter struct{}

var _ AttributeConverter = &StructConverter{}

func (c *StructConverter) Check(typ reflect.Type) (bool, error) {
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	return typ.Kind() == reflect.Struct, nil
}

func (c *StructConverter) GetFrameworkType(converters *Converter, typ reflect.Type) (*jen.Statement, error) {
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	name, _, _, _, err := converters.GetNamesForType(typ)
	if err != nil {
		return nil, err
	}
	return jen.Op("*").Id(name), nil
}

func (c *StructConverter) Decode(converters *Converter, path, src, target *jen.Statement, typ reflect.Type) (*jen.Statement, error) {
	ref := jen.Op("*").Id("item")
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
		ref = jen.Id("item")
	}

	_, _, decodeFunctionName, _, err := converters.GetNamesForType(typ)
	if err != nil {
		return nil, err
	}

	return jen.If(src.Clone().Op("!=").Nil()).Block(
		jen.Var().Id("item").Op("*").Qual(typ.PkgPath(), typ.Name()),
		jen.Id("diags").Dot("Append").Call(jen.Id(decodeFunctionName).Call(path, jen.Add(src), jen.Op("&").Id("item")).Op("...")),
		jen.Line(),
		jen.If(jen.Id("diags").Dot("HasError").Call()).Block(
			jen.Return(jen.Id("diags")),
		).Line(),
		target.Op("=").Add(ref),
	), nil
}

func (c *StructConverter) Encode(converters *Converter, src, target *jen.Statement, typ reflect.Type) (*jen.Statement, error) {
	ptr := false
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
		ptr = true
	}

	_, _, _, encodingFuncName, err := converters.GetNamesForType(typ)
	if err != nil {
		return nil, err
	}

	op := jen.Empty()
	if !ptr {
		op = jen.Op("&")
	}

	return jen.Block(
		jen.List(jen.Id("data"), jen.Id("d")).Op(":=").Id(encodingFuncName).Call(op.Add(src)),
		jen.Id("diags").Dot("Append").Call(jen.Id("d").Op("...")),
		jen.If(jen.Id("diags").Dot("HasError").Call()).Block(
			jen.Return(jen.List(jen.Nil(), jen.Id("diags"))),
		),
		jen.If(jen.Id("data").Op("!=").Nil()).Block(
			target.Op("=").Id("data"),
		),
	), nil

}

func (c *StructConverter) GetSchema(converters *Converter, path string, info *tags.FieldInformation) (*jen.Statement, *jen.Statement, error) {
	typ := info.GoType
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}

	fields, err := converters.GetFields(path, typ)
	if err != nil {
		return nil, nil, err
	}

	attrs := []jen.Code{}
	blocks := []jen.Code{}
	for _, field := range fields {
		converter, err := converters.Get(field.GoType)
		if err != nil {
			return nil, nil, err
		}
		attr, b, err := converter.GetSchema(converters, field.Path, field)
		if err != nil {
			return nil, nil, err
		}
		if attr != nil {
			attrs = append(attrs, jen.Line().Lit(field.Name).Op(":").Add(attr))
		}
		if b != nil {
			blocks = append(blocks, jen.Line().Lit(field.Name).Op(":").Add(b))
		}
	}
	attrs = append(attrs, jen.Line())

	codes := []jen.Code{}
	codes = append(codes, jen.Id("Attributes").Op(":").Map(jen.String()).Qual(converters.SchemaImportPath(), "Attribute").Values(attrs...))
	if len(blocks) != 0 {
		if !info.Block {
			return nil, nil, fmt.Errorf("%#v: got blocks but this is an attribute", path)
		}
		blocks = append(blocks, jen.Line())
		codes = append(codes, jen.Id("Blocks").Op(":").Map(jen.String()).Qual(converters.SchemaImportPath(), "Block").Values(blocks...))
	}

	attrType := jen.Empty()

	result := jen.Op("&").Add(attrType).ValuesFunc(func(g *jen.Group) {
		if info.Optional && !info.Block {
			g.Line().Id("Optional").Op(":").True()
		}
		if info.Required && !info.Block {
			g.Line().Id("Required").Op(":").True()
		}
		if info.Computed && !info.Block {
			g.Line().Id("Computed").Op(":").True()
		}
		if info.Sensitive {
			g.Line().Id("Sensitive").Op(":").True()
		}
		if info.Description != "" {
			g.Line().Id("MarkdownDescription").Op(":").Lit(info.Description)
		}
		for _, code := range codes {
			g.Line().Add(code)
		}
		g.Line()
	})

	if info.Block {
		attrType.Qual(converters.SchemaImportPath(), "SingleNestedBlock")
		return nil, result, nil
	}

	attrType.Qual(converters.SchemaImportPath(), "SingleNestedAttribute")
	return result, nil, nil
}
