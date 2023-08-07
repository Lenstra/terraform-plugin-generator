package converters

import (
	"fmt"
	"reflect"

	"github.com/Lenstra/terraform-plugin-generator/tags"
	"github.com/dave/jennifer/jen"
)

type ListConverter struct{}

var _ AttributeConverter = &ListConverter{}

func (c *ListConverter) Check(typ reflect.Type) (bool, error) {
	return typ.Kind() == reflect.Slice, nil
}

func (c *ListConverter) GetFrameworkType(converters *Converter, typ reflect.Type) (*jen.Statement, error) {
	subType, err := converters.GetFrameworkType(typ.Elem())
	if err != nil {
		return nil, err
	}
	return jen.Index().Add(subType), nil
}

func (c *ListConverter) Decode(converters *Converter, path, src, target *jen.Statement, typ reflect.Type) (*jen.Statement, error) {
	code, err := converters.Decode(path.Dot("AtListIndex").Call(jen.Id("i")), jen.Id("data"), target.Clone().Index(jen.Id("i")), typ.Elem())
	if err != nil {
		return nil, err
	}

	typ = typ.Elem()
	op := jen.Empty()
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
		op = jen.Op("*")
	}

	codeType := jen.Index().Add(op).Qual(typ.PkgPath(), typ.Name())

	return jen.If(src.Clone().Op("!=").Nil()).Block(
		target.Clone().Op("=").Make(codeType, jen.Len(src.Clone())),
		jen.For(jen.List(jen.Id("i"), jen.Id("data")).Op(":=").Range().Add(src)).Block(
			code,
		),
	), nil
}

func (c *ListConverter) Encode(converters *Converter, src, target *jen.Statement, typ reflect.Type) (*jen.Statement, error) {
	frameworkType, err := converters.GetFrameworkType(typ)
	if err != nil {
		return nil, err
	}

	typ = typ.Elem()

	code, err := converters.Encode(jen.Id("attr"), target.Clone().Index(jen.Id("i")), typ)
	if err != nil {
		return nil, err
	}

	return jen.If(src.Clone().Op("!=").Nil()).Block(
		target.Clone().Op("=").Make(frameworkType, jen.Len(src)),
		jen.For(jen.List(jen.Id("i"), jen.Id("attr")).Op(":=").Range().Add(src)).Block(
			code,
		),
	), nil
}

func (c *ListConverter) GetSchema(converters *Converter, path string, info *tags.FieldInformation) (*jen.Statement, *jen.Statement, error) {
	typ := info.GoType.Elem()
	converter, err := converters.Get(typ)
	if err != nil {
		return nil, nil, err
	}
	if simpleConverter, ok := converter.(SimpleAttributeConverter); ok {
		return jen.Qual(converters.SchemaImportPath(), "ListAttribute").ValuesFunc(func(g *jen.Group) {
			g.Line().Id("ElementType").Op(":").Add(simpleConverter.GetType())
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
			g.Line()
		}), nil, nil
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
	innerType := jen.Empty()

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
		g.Line().Id("NestedObject").Op(":").Add(innerType).ValuesFunc(func(g *jen.Group) {
			for _, code := range codes {
				g.Line().Add(code)
			}
		})
		g.Line()
	})

	if info.Block {
		attrType.Qual(converters.SchemaImportPath(), "ListNestedBlock")
		innerType.Qual(converters.SchemaImportPath(), "NestedBlockObject")
		return nil, result, nil
	}

	attrType.Qual(converters.SchemaImportPath(), "ListNestedAttribute")
	innerType.Qual(converters.SchemaImportPath(), "NestedAttributeObject")
	return result, nil, nil
}
