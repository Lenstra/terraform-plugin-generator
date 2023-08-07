package generator

import (
	"fmt"
	"path/filepath"
	"reflect"
	"sort"

	"github.com/Lenstra/terraform-plugin-generator/converters"
	"github.com/Lenstra/terraform-plugin-generator/internal/helpers"
	"github.com/Lenstra/terraform-plugin-generator/internal/iterators"
	"github.com/Lenstra/terraform-plugin-generator/tags"
	. "github.com/dave/jennifer/jen" //lint:ignore ST1001 accept dot import
	"github.com/hashicorp/go-hclog"
	"golang.org/x/exp/slices"
)

type ModelGenerator struct {
	Path                string
	Package             string
	Objects             map[string]interface{}
	Logger              hclog.Logger
	GetFieldInformation tags.FieldInformationGetter
	AttributeConverters []converters.AttributeConverter
}

func (g *ModelGenerator) Render() error {
	if g.Logger == nil {
		g.Logger = hclog.Default()
	}
	if g.AttributeConverters == nil {
		g.AttributeConverters = converters.DefaultConverters
	}
	if g.GetFieldInformation == nil {
		g.GetFieldInformation = tags.GetFieldInformationFromTerraformTag
	}

	if g.Package == "" {
		return fmt.Errorf("missing package name")
	}

	userGiven := []string{}
	names := map[reflect.Type]string{}
	types := map[string]reflect.Type{}
	done := map[string]bool{}
	for key, obj := range g.Objects {
		names[reflect.TypeOf(obj)] = key
		types[key] = reflect.TypeOf(obj)

		if _, found := done[key]; found {
			return fmt.Errorf("%s has been given multiple time", key)
		}
		done[key] = false

		userGiven = append(userGiven, key)
	}

	sort.Strings(userGiven)

	converter := converters.NewConverter(g.AttributeConverters, &names, g.GetFieldInformation, "")

	queue := []reflect.Type{}
	for _, name := range userGiven {
		queue = append(queue, reflect.TypeOf(g.Objects[name]))
	}

	modelFile := NewFile(g.Package)
	modelFile.HeaderComment(headerComment)

	decodersFile := NewFile(g.Package)
	decodersFile.HeaderComment(headerComment)
	decodersFile.Type().Id("Getter").Interface(
		Id("Get").Params(Qual("context", "Context"), Interface()).Qual("github.com/hashicorp/terraform-plugin-framework/diag", "Diagnostics"),
	).Line()

	encodersFile := NewFile(g.Package)
	encodersFile.HeaderComment(headerComment)
	encodersFile.Type().Id("Setter").Interface(
		Id("Set").Params(Qual("context", "Context"), Interface()).Qual("github.com/hashicorp/terraform-plugin-framework/diag", "Diagnostics"),
	).Line()

	cases := []Code{}
	for _, name := range userGiven {
		typ := types[name]
		_, _, _, encodeFunctionName, err := converter.GetNamesForType(typ)
		if err != nil {
			return err
		}

		cases = append(cases, Case(Op("*").Qual(typ.PkgPath(), typ.Name())).Block(
			List(Id("converted"), Id("diags")).Op("=").Id(encodeFunctionName).Call(Id("o"))),
		)
	}

	encodersFile.Func().Id("Set").Params(Id("ctx").Qual("context", "Context"), Id("setter").Id("Setter"), Id("obj").Interface()).Qual("github.com/hashicorp/terraform-plugin-framework/diag", "Diagnostics").Block(
		Var().Id("diags").Qual("github.com/hashicorp/terraform-plugin-framework/diag", "Diagnostics"),
		Var().Id("converted").Interface(),
		Switch(Id("o").Op(":=").Id("obj").Assert(Id("type"))).BlockFunc(func(gr *Group) {
			for _, c := range cases {
				gr.Add(c)
			}
			gr.Default().Block(
				Id("diags").Dot("AddError").Call(Lit("unsupported object type"), Qual("fmt", "Sprintf").Call(Lit("%T is not supported in %s.Set(). Please report this issue to the provider developers."), Id("obj"), Lit(g.Package))),
				Return().Id("diags"),
			)
		}),
		Id("diags").Dot("Append").Call(Id("setter").Dot("Set").Call(Id("ctx"), Id("converted")).Op("...")),
		Return().Id("diags"),
	).Line()

	privateDecodeFunctions := Empty()

	for i := 0; i < len(queue); i++ {
		typ := queue[i]
		if typ.Kind() != reflect.Struct {
			return fmt.Errorf("this should not happen")
		}
		if helpers.Ignore(typ) {
			continue
		}
		// If we have a name for this type it means it has already been
		// handled
		if name, found := names[typ]; found && done[name] {
			continue
		}
		// It has not been treated already, let's find a name for this type
		name, _, _, _, err := converter.GetNamesForType(typ)
		if err != nil {
			return err
		}

		done[name] = true

		if slices.Contains(userGiven, name) {
			code, err := g.renderPublicDecodeFunction(converter, typ)
			if err != nil {
				return err
			}
			decodersFile.Add(*code...)
		}

		code, err := g.renderDecodeFunction(converter, typ)
		if err != nil {
			return err
		}
		privateDecodeFunctions.Add(*code...)

		code, err = g.renderEncodeFunction(converter, typ)
		if err != nil {
			return err
		}
		encodersFile.Add(*code...)

		code, todo, err := g.renderObject(converter, typ)
		queue = append(queue, todo...)
		if err != nil {
			return err
		}

		modelFile.Add(*code...)
	}

	if err := modelFile.Save(filepath.Join(g.Path, "models.go")); err != nil {
		return err
	}

	decodersFile.Add(privateDecodeFunctions)

	if err := decodersFile.Save(filepath.Join(g.Path, "decoders.go")); err != nil {
		return err
	}

	return encodersFile.Save(filepath.Join(g.Path, "encoders.go"))
}

func (g *ModelGenerator) renderObject(c *converters.Converter, typ reflect.Type) (*Statement, []reflect.Type, error) {
	fields, todo, err := iterators.IterateFields("", g.GetFieldInformation, typ)
	if err != nil {
		return nil, nil, err
	}

	var codes []Code
	for _, field := range fields {
		code, err := c.GetFrameworkType(field.GoType)
		if err != nil {
			return nil, nil, err
		}

		codes = append(
			codes,
			Id(field.GoName).Add(code).Tag(map[string]string{"tfsdk": field.Name}),
		)
	}

	name, _, _, _, err := c.GetNamesForType(typ)
	if err != nil {
		return nil, nil, err
	}
	return Type().Id(name).Struct(codes...).Line(), todo, nil
}

func (g *ModelGenerator) renderDecodeFunction(c *converters.Converter, typ reflect.Type) (*Statement, error) {
	fields, _, err := iterators.IterateFields("", g.GetFieldInformation, typ)
	if err != nil {
		return nil, err
	}

	name, ident, decodeFunctionName, _, err := c.GetNamesForType(typ)
	if err != nil {
		return nil, err
	}

	codes := []Code{}
	for _, field := range fields {
		code, err := c.Decode(
			Id("path").Dot("AtName").Call(Lit(field.Name)),
			Id("data").Dot(field.GoName),
			Id(ident).Add(field.Accessor),
			field.GoType,
		)
		if err != nil {
			return nil, err
		}

		codes = append(codes, code, Line())
	}

	codes = append(codes, Line(), Return(Id("diags")))

	return Func().Id(decodeFunctionName).Params(
		Id("path").Qual("github.com/hashicorp/terraform-plugin-framework/path", "Path"),
		Id("data").Id(name),
		Id(ident).Op("*").Qual(typ.PkgPath(), typ.Name()),
	).Parens(Id("diags").Qual("github.com/hashicorp/terraform-plugin-framework/diag", "Diagnostics")).Block(
		codes...,
	).Line().Line(), nil
}

func (g *ModelGenerator) renderPublicDecodeFunction(c *converters.Converter, typ reflect.Type) (*Statement, error) {
	_, name, decodeFunctionName, _, err := c.GetNamesForType(typ)
	if err != nil {
		return nil, err
	}

	return Func().Id(helpers.Public(decodeFunctionName)).Params(
		Id("ctx").Qual("context", "Context"),
		Id("getter").Id("Getter"),
		Id(name).Op("*").Qual(typ.PkgPath(), typ.Name()),
	).Qual("github.com/hashicorp/terraform-plugin-framework/diag", "Diagnostics").Block(
		Var().Id("data").Id(typ.Name()),
		Id("diags").Op(":=").Id("getter").Dot("Get").Params(Id("ctx"), Op("&").Id("data")),
		If(Id("diags").Dot("HasError").Call()).Block(
			Return(Id("diags")),
		),
		Line(),
		Id("diags").Dot("Append").Call(Id(decodeFunctionName).Call(
			Qual("github.com/hashicorp/terraform-plugin-framework/path", "Empty()"),
			Id("data"),
			Id(name),
		).Op("...")),
		Return(Id("diags")),
	).Line(), nil
}

func (g *ModelGenerator) renderEncodeFunction(c *converters.Converter, typ reflect.Type) (*Statement, error) {
	fields, _, err := iterators.IterateFields("", g.GetFieldInformation, typ)
	if err != nil {
		return nil, err
	}

	name, ident, _, encodeFunctionName, err := c.GetNamesForType(typ)
	if err != nil {
		return nil, err
	}

	codes := []Code{}
	for _, field := range fields {
		code, err := c.Encode(
			Id(ident).Add(field.Accessor),
			Id("res").Dot(field.GoName),
			field.GoType,
		)
		if err != nil {
			return nil, err
		}

		codes = append(codes, code)
	}

	return Func().Id(encodeFunctionName).Params(
		Id(ident).Op("*").Qual(typ.PkgPath(), typ.Name()),
	).Parens(List(Op("*").Id(name), Qual("github.com/hashicorp/terraform-plugin-framework/diag", "Diagnostics"))).BlockFunc(func(g *Group) {
		g.If(Id(ident).Op("==").Nil()).Block(
			Return().List(Nil(), Nil()),
		).Line()
		g.Var().Id("diags").Qual("github.com/hashicorp/terraform-plugin-framework/diag", "Diagnostics")
		g.Id("res").Op(":=").Id(name).Block()

		for _, code := range codes {
			g.Add(code)
		}

		g.Return().List(Op("&").Id("res"), Id("diags"))
	}).Line(), nil
}
