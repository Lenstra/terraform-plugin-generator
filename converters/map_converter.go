package converters

import (
	"reflect"

	"github.com/Lenstra/terraform-plugin-generator/tags"
	"github.com/dave/jennifer/jen"
)

type MapConverter struct{}

var _ AttributeConverter = &MapConverter{}

func (c *MapConverter) Check(typ reflect.Type) (bool, error) {
	return typ.Kind() == reflect.Map && typ.Key().Kind() == reflect.String, nil
}

func (c *MapConverter) GetFrameworkType(converters *Converter, typ reflect.Type) (*jen.Statement, error) {
	subType, err := converters.GetFrameworkType(typ.Elem())
	if err != nil {
		return nil, err
	}
	return jen.Map(jen.String()).Add(subType), nil
}

func (c *MapConverter) Decode(converters *Converter, path, src, target *jen.Statement, typ reflect.Type) (*jen.Statement, error) {
	code, err := converters.Decode(path.Dot("AtMapKey").Call(jen.Id("key")), jen.Id("data"), target.Clone().Index(jen.Id("key")), typ.Elem())
	if err != nil {
		return nil, err
	}

	typ = typ.Elem()
	op := jen.Empty()
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
		op = op.Op("*")
	}
	if typ.Kind() == reflect.Slice {
		typ = typ.Elem()
		op = op.Index()
	}
	codeType := jen.Map(jen.String()).Add(op).Qual(typ.PkgPath(), typ.Name()).Block()

	return jen.If(src.Clone().Op("!=").Nil()).Block(
		target.Clone().Op("=").Add(codeType),
		jen.For(jen.List(jen.Id("key"), jen.Id("data")).Op(":=").Range().Add(src.Clone())).Block(
			code,
		),
	), nil
}

func (c *MapConverter) Encode(converters *Converter, src, target *jen.Statement, typ reflect.Type) (*jen.Statement, error) {
	frameworkType, err := converters.GetFrameworkType(typ)
	if err != nil {
		return nil, err
	}
	typ = typ.Elem()

	code, err := converters.Encode(jen.Id("v"), target.Clone().Index(jen.Id("k")), typ)
	if err != nil {
		return nil, err
	}

	return jen.If(src.Clone().Op("!=").Nil()).Block(
		target.Clone().Op("=").Add(frameworkType).Block(),
		jen.For(jen.List(jen.Id("k"), jen.Id("v")).Op(":=").Range().Add(src.Clone())).Block(
			code,
		),
	), nil
}

func (c *MapConverter) GetSchema(converters *Converter, path string, info *tags.FieldInformation) (*jen.Statement, *jen.Statement, error) {
	typ := info.GoType.Elem()
	slice := false
	if typ.Kind() == reflect.Slice {
		slice = true
		typ = typ.Elem()
	}

	converter, err := converters.Get(typ)
	if err != nil {
		return nil, nil, err
	}
	if simpleConverter, ok := converter.(SimpleAttributeConverter); ok {
		inner := simpleConverter.GetType()
		if slice {
			inner = jen.Qual("github.com/hashicorp/terraform-plugin-framework/types", "ListType").Values(
				jen.Line().Id("ElemType").Op(":").Add(inner),
				jen.Line(),
			)
		}

		return jen.Qual(converters.SchemaImportPath(), "MapAttribute").ValuesFunc(func(g *jen.Group) {
			g.Line().Id("ElementType").Op(":").Add(inner)
			if info.Optional {
				g.Line().Id("Optional").Op(":").True()
			}
			if info.Required {
				g.Line().Id("Required").Op(":").True()
			}
			if info.Computed {
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
	codes := []jen.Code{}
	for _, field := range fields {
		converter, err := converters.Get(field.GoType)
		if err != nil {
			return nil, nil, err
		}

		attr, _, err := converter.GetSchema(converters, field.Path, field)
		if err != nil {
			return nil, nil, err
		}
		if attr != nil {
			codes = append(codes, jen.Lit(field.Name).Op(":").Add(attr))
		}
	}

	return jen.Op("&").Qual(converters.SchemaImportPath(), "ListNestedAttribute").ValuesFunc(func(g *jen.Group) {
		if info.Optional {
			g.Line().Id("Optional").Op(":").True()
		}
		if info.Required {
			g.Line().Id("Required").Op(":").True()
		}
		if info.Computed {
			g.Line().Id("Computed").Op(":").True()
		}
		if info.Sensitive {
			g.Line().Id("Sensitive").Op(":").True()
		}
		if info.Description != "" {
			g.Line().Id("MarkdownDescription").Op(":").Lit(info.Description)
		}
		g.Line().Id("NestedObject").Op(":").Qual(converters.SchemaImportPath(), "NestedAttributeObject").Values(
			jen.Line().Id("Attributes").Op(":").Map(jen.String()).Qual(converters.SchemaImportPath(), "Attribute").ValuesFunc(func(g *jen.Group) {
				for _, code := range codes {
					g.Line().Add(code)
				}
				g.Line()
			}),
			jen.Line(),
		)
		g.Line()
	}), nil, nil
}
