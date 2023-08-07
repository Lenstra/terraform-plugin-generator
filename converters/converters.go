package converters

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/Lenstra/terraform-plugin-generator/internal/helpers"
	"github.com/Lenstra/terraform-plugin-generator/internal/iterators"
	"github.com/Lenstra/terraform-plugin-generator/tags"
	"github.com/dave/jennifer/jen"
	"github.com/stoewer/go-strcase"
)

func basicSchema(importPath, name string, info *tags.FieldInformation, attributes []jen.Code) (*jen.Statement, *jen.Statement, error) {
	return jen.Qual(importPath, name).ValuesFunc(func(g *jen.Group) {
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
		for _, code := range attributes {
			g.Line().Add(code)
		}
		g.Line()
	}), nil, nil
}

var DefaultConverters = []AttributeConverter{
	&MapInterfaceConverter{},
	&BoolConverter{},
	&StringConverter{},
	&IntConverter{},
	&ListConverter{},
	&MapConverter{},
	&StructConverter{},
}

type AttributeConverter interface {
	Check(reflect.Type) (bool, error)
	GetFrameworkType(*Converter, reflect.Type) (*jen.Statement, error)
	Decode(*Converter, *jen.Statement, *jen.Statement, *jen.Statement, reflect.Type) (*jen.Statement, error)
	Encode(*Converter, *jen.Statement, *jen.Statement, reflect.Type) (*jen.Statement, error)
	GetSchema(*Converter, string, *tags.FieldInformation) (*jen.Statement, *jen.Statement, error)
}

type SimpleAttributeConverter interface {
	GetType() *jen.Statement
}

type NoConverterFoundError struct {
	typ reflect.Type
}

func (e *NoConverterFoundError) Error() string {
	if e.typ == nil {
		return "no converter registered"
	}
	return fmt.Sprintf("no converter found for %s", e.typ.String())
}

type Converter struct {
	attributeConverters []AttributeConverter
	names               *map[reflect.Type]string
	userGivenType       map[reflect.Type]struct{}
	getFieldInformation tags.FieldInformationGetter
	schemaImportPath    string
}

func NewConverter(attributeConverters []AttributeConverter, names *map[reflect.Type]string, getFieldInformation tags.FieldInformationGetter, schemaImportPath string) *Converter {
	c := &Converter{
		attributeConverters: attributeConverters,
		names:               names,
		userGivenType:       map[reflect.Type]struct{}{},
		getFieldInformation: getFieldInformation,
		schemaImportPath:    schemaImportPath,
	}

	// We keep track of the types given by the user so that we can return the
	// appropriate case for the encoding function
	for k := range *names {
		c.userGivenType[k] = struct{}{}
	}

	return c
}

func (c *Converter) Get(typ reflect.Type) (AttributeConverter, error) {
	if c == nil {
		return nil, &NoConverterFoundError{}
	}
	for _, converter := range c.attributeConverters {
		ok, err := converter.Check(typ)
		if err != nil {
			return nil, err
		}
		if ok {
			return converter, nil
		}
	}

	return nil, &NoConverterFoundError{typ}
}

func (c *Converter) GetFrameworkType(typ reflect.Type) (*jen.Statement, error) {
	converter, err := c.Get(typ)
	if err != nil {
		return nil, err
	}
	stmt, err := converter.GetFrameworkType(c, typ)
	return validate("GetFrameworkType()", converter, typ, stmt, err)
}

func (c *Converter) Decode(path, src, dest *jen.Statement, typ reflect.Type) (*jen.Statement, error) {
	converter, err := c.Get(typ)
	if err != nil {
		return nil, err
	}
	stmt, err := converter.Decode(c, path, src, dest, typ)
	return validate("Decode()", converter, typ, stmt, err)
}

func (c *Converter) Encode(src, dest *jen.Statement, typ reflect.Type) (*jen.Statement, error) {
	converter, err := c.Get(typ)
	if err != nil {
		return nil, err
	}
	stmt, err := converter.Encode(c, src, dest, typ)
	return validate("Encode()", converter, typ, stmt, err)
}

func (c *Converter) GetNamesForType(typ reflect.Type) (string, string, string, string, error) {
	name := (*c.names)[typ]
	if name == "" {
		name = typ.Name()
	}
	for t, v := range *c.names {

		if name == v && t != typ {
			// If there is a conflict we try to prefix the name with the package

			name = typ.PkgPath() + name
			name = helpers.Public(name[strings.LastIndex(name, "/")+1:])

			for t, v := range *c.names {
				// If there is still a conflict we bail out now
				if name == v && t != typ {
					return "", "", "", "", fmt.Errorf("conflict with %s", name)
				}
			}
		}
	}
	(*c.names)[typ] = name

	encodingFuncName := "encode" + name
	if _, found := c.userGivenType[typ]; found {
		encodingFuncName = helpers.Public(encodingFuncName)
	}

	return name, strcase.LowerCamelCase(name), "decode" + name, encodingFuncName, nil
}

func (c *Converter) GetFields(path string, typ reflect.Type) ([]*tags.FieldInformation, error) {
	fields, _, err := iterators.IterateFields(path, c.getFieldInformation, typ)
	if err != nil {
		return nil, err
	}
	return fields, nil
}

func (c *Converter) SchemaImportPath() string {
	return c.schemaImportPath
}

func validate(name string, c AttributeConverter, typ reflect.Type, stmt *jen.Statement, err error) (*jen.Statement, error) {
	if err != nil {
		return nil, err
	}
	if stmt == nil {
		return nil, fmt.Errorf("no code received from %T for %s in %s", c, typ.String(), name)
	}
	return stmt, nil
}
