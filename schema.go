package generator

import (
	"fmt"
	"path/filepath"
	"reflect"
	"sort"

	"github.com/Lenstra/terraform-plugin-generator/converters"
	"github.com/Lenstra/terraform-plugin-generator/internal/iterators"
	"github.com/Lenstra/terraform-plugin-generator/tags"
	"github.com/hashicorp/go-hclog"
	"github.com/stoewer/go-strcase"

	. "github.com/dave/jennifer/jen" //lint:ignore ST1001 accept dot import
)

type SchemaType string

var (
	ProviderSchema     SchemaType = "provider"
	DataSourceSchema   SchemaType = "datasource"
	ResourceSchema     SchemaType = "resource"
	ProviderMetaSchema SchemaType = "providermeta"
)

func (s SchemaType) importPath() string {
	switch s {
	case ProviderSchema:
		return "github.com/hashicorp/terraform-plugin-framework/provider/schema"
	case DataSourceSchema:
		return "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	case ResourceSchema:
		return "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	case ProviderMetaSchema:
		return "github.com/hashicorp/terraform-plugin-framework/provider/metaschema"
	}
	return ""
}

type SchemaGenerator struct {
	Type                SchemaType
	Path                string
	Package             string
	Objects             map[string]interface{}
	Logger              hclog.Logger
	GetFieldInformation tags.FieldInformationGetter
	AttributeConverters []converters.AttributeConverter
}

func (g *SchemaGenerator) Render() error {
	if g.AttributeConverters == nil {
		g.AttributeConverters = converters.DefaultConverters
	}
	if g.GetFieldInformation == nil {
		g.GetFieldInformation = tags.GetFieldInformationFromTerraformTag
	}
	importPath := g.Type.importPath()
	if importPath == "" {
		return fmt.Errorf("unexpected schema type %q", g.Type)
	}

	names := []string{}
	for k := range g.Objects {
		names = append(names, k)
	}

	m := map[reflect.Type]string{}
	converter := converters.NewConverter(g.AttributeConverters, &m, g.GetFieldInformation, importPath)

	sort.Strings(names)

	f := NewFile(g.Package)
	f.HeaderComment(headerComment)

	for _, name := range names {
		code, err := g.renderObject(converter, importPath, name, reflect.TypeOf(g.Objects[name]))
		if err != nil {
			return err
		}
		f.Add(code)

	}

	return f.Save(filepath.Join(g.Path, "schema.go"))
}

func (g *SchemaGenerator) renderObject(c *converters.Converter, importPath, name string, typ reflect.Type) (*Statement, error) {
	fields, _, err := iterators.IterateFields(name, g.GetFieldInformation, typ)
	if err != nil {
		return nil, err
	}

	attributes := []Code{}
	blocks := []Code{}
	for _, field := range fields {
		converter, err := c.Get(field.GoType)
		if err != nil {
			return nil, err
		}

		attr, b, err := converter.GetSchema(c, field.Path, field)
		if err != nil {
			return nil, err
		}
		if attr != nil {
			attributes = append(attributes, Lit(field.Name).Op(":").Add(attr))
		}
		if b != nil {
			blocks = append(blocks, Lit(field.Name).Op(":").Add(b))
		}
	}

	return Func().Id(strcase.LowerCamelCase(name)+"Schema").Params().Qual(importPath, "Schema").BlockFunc(func(g *Group) {
		g.Return().Qual(importPath, "Schema").Values(
			Line().Id("MarkdownDescription").Op(":").Lit(""),
			Line().Id("Attributes").Op(":").Map(String()).Qual(importPath, "Attribute").ValuesFunc(func(g *Group) {
				for _, code := range attributes {
					g.Line().Add(code)
				}
				g.Line()
			}),
			Line().Id("Blocks").Op(":").Map(String()).Qual(importPath, "Block").ValuesFunc(func(g *Group) {
				for _, code := range blocks {
					g.Line().Add(code)
				}
				g.Line()
			}),
			Line(),
		)
	}).Line(), nil
}
