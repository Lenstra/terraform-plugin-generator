package generator

import (
	"fmt"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	. "github.com/dave/jennifer/jen" //lint:ignore ST1001 accept dot import
	"golang.org/x/exp/slices"
)

func GenerateModels(path, pkg string, objects map[string]interface{}, opts *GeneratorOptions) error {
	opts = opts.validate()

	userGiven := []string{}
	names := map[reflect.Type]string{}
	types := map[string]reflect.Type{}
	done := map[string]bool{}
	for key, obj := range objects {
		names[reflect.TypeOf(obj)] = key
		types[key] = reflect.TypeOf(obj)

		if _, found := done[key]; found {
			return fmt.Errorf("%s has been given multiple time", key)
		}
		done[key] = false

		userGiven = append(userGiven, key)
	}

	sort.Strings(userGiven)

	converter := NewConverter(opts.AttributeConverters, &names, opts.GetFieldInformation, "")

	queue := []reflect.Type{}
	for _, name := range userGiven {
		queue = append(queue, reflect.TypeOf(objects[name]))
	}

	cases := []Code{}
	for _, name := range userGiven {
		typ := types[name]
		_, _, decodeFunctionName, _, err := converter.GetNamesForType(typ)
		if err != nil {
			return err
		}
		publicName := strings.ToUpper(decodeFunctionName[:1]) + decodeFunctionName[1:]

		cases = append(cases, Case(Op("**").Qual(typ.PkgPath(), typ.Name())).Block(
			Return().Id(publicName).Call(Id("ctx"), Id("getter"), Id("o"))),
		)
	}

	modelFile := newFile(pkg)
	decodersFile := newFile(pkg)
	decodersFile.Type().Id("Getter").Interface(
		Id("Get").Params(Qual("context", "Context"), Interface()).Qual("github.com/hashicorp/terraform-plugin-framework/diag", "Diagnostics"),
	).Line()

	decodersFile.Func().Id("Decode").Index(Id("Target").UnionFunc(func(g *Group) {
		for _, name := range userGiven {
			typ := types[name]
			g.Op("**").Qual(typ.PkgPath(), typ.Name())
		}
	})).Params(Id("ctx").Qual("context", "Context"), Id("getter").Id("Getter"), Id("obj").Id("Target")).Qual("github.com/hashicorp/terraform-plugin-framework/diag", "Diagnostics").Block(
		Switch(Id("o").Op(":=").Any().Call(Id("obj")).Assert(Id("type"))).BlockFunc(func(gr *Group) {
			for _, c := range cases {
				gr.Add(c)
			}
			gr.Default().Block(
				Var().Id("diags").Qual("github.com/hashicorp/terraform-plugin-framework/diag", "Diagnostics"),
				Id("diags").Dot("AddError").Call(Lit("unsupported object type"), Qual("fmt", "Sprintf").Call(Lit("%T is not supported in %s.Set(). Please report this issue to the provider developers."), Id("obj"), Lit(pkg))),
				Return().Id("diags"),
			)
		}),
	).Line()

	encodersFile := newFile(pkg)
	encodersFile.Type().Id("Setter").Interface(
		Id("Set").Params(Qual("context", "Context"), Interface()).Qual("github.com/hashicorp/terraform-plugin-framework/diag", "Diagnostics"),
	).Line()

	cases = []Code{}
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

	encodersFile.Func().Id("Set").Index(Id("Model").UnionFunc(func(g *Group) {
		for _, name := range userGiven {
			typ := types[name]
			g.Op("*").Qual(typ.PkgPath(), typ.Name())
		}
	})).Params(Id("ctx").Qual("context", "Context"), Id("setter").Id("Setter"), Id("obj").Id("Model")).Qual("github.com/hashicorp/terraform-plugin-framework/diag", "Diagnostics").Block(
		Var().Id("diags").Qual("github.com/hashicorp/terraform-plugin-framework/diag", "Diagnostics"),
		Var().Id("converted").Interface(),
		Switch(Id("o").Op(":=").Any().Call(Id("obj")).Assert(Id("type"))).BlockFunc(func(gr *Group) {
			for _, c := range cases {
				gr.Add(c)
			}
			gr.Default().Block(
				Id("diags").Dot("AddError").Call(Lit("unsupported object type"), Qual("fmt", "Sprintf").Call(Lit("%T is not supported in %s.Set(). Please report this issue to the provider developers."), Id("obj"), Lit(pkg))),
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
		if typ == reflect.TypeOf(time.Time{}) {
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
			code, err := renderPublicDecodeFunction(converter, typ)
			if err != nil {
				return err
			}
			decodersFile.Add(*code...)
		}

		code, err := renderDecodeFunction(converter, opts, typ)
		if err != nil {
			return err
		}
		privateDecodeFunctions.Add(*code...)

		code, err = renderEncodeFunction(converter, opts, typ)
		if err != nil {
			return err
		}
		encodersFile.Add(*code...)

		code, todo, err := renderObject(converter, opts, typ)
		queue = append(queue, todo...)
		if err != nil {
			return err
		}

		modelFile.Add(*code...)
	}

	if err := modelFile.Save(filepath.Join(path, "models.go")); err != nil {
		return err
	}

	decodersFile.Add(privateDecodeFunctions)

	if err := decodersFile.Save(filepath.Join(path, "decoders.go")); err != nil {
		return err
	}

	return encodersFile.Save(filepath.Join(path, "encoders.go"))
}

func renderObject(c *Converter, opts *GeneratorOptions, typ reflect.Type) (*Statement, []reflect.Type, error) {
	fields, todo, err := iterateFields("", opts.GetFieldInformation, typ)
	if err != nil {
		return nil, nil, err
	}

	var codes []Code
	for _, field := range fields {
		code, err := c.GetFrameworkType(field.goType)
		if err != nil {
			return nil, nil, err
		}

		codes = append(
			codes,
			Id(field.goName).Add(code).Tag(map[string]string{"tfsdk": field.Name}),
		)
	}

	name, _, _, _, err := c.GetNamesForType(typ)
	if err != nil {
		return nil, nil, err
	}
	return Type().Id(name).Struct(codes...).Line(), todo, nil
}

func renderDecodeFunction(c *Converter, opts *GeneratorOptions, typ reflect.Type) (*Statement, error) {
	fields, _, err := iterateFields("", opts.GetFieldInformation, typ)
	if err != nil {
		return nil, err
	}

	name, ident, decodeFunctionName, _, err := c.GetNamesForType(typ)
	if err != nil {
		return nil, err
	}

	codes := []Code{}
	for _, field := range fields {
		target := Id("target")
		if field.Parent != nil {
			target.Add(field.Parent.accessor.Clone())
		}
		code, err := c.Decode(
			field,
			Id("path").Dot("AtName").Call(Lit(field.Name)),
			Id("data").Dot(field.goName),
			target.Add(field.accessor),
			field.goType,
		)
		if err != nil {
			return nil, err
		}

		codes = append(codes, code, Line())
	}

	codes = append(codes, Line(), Return(Id("diags")))

	return Func().Id(decodeFunctionName).Params(
		Id("path").Qual("github.com/hashicorp/terraform-plugin-framework/path", "Path"),
		Id("data").Op("*").Id(name),
		Id(ident).Op("**").Qual(typ.PkgPath(), typ.Name()),
	).Parens(Id("diags").Qual("github.com/hashicorp/terraform-plugin-framework/diag", "Diagnostics")).BlockFunc(func(g *Group) {
		g.If(Id("data").Op("==").Nil()).Block(
			Return().Nil(),
		).Line()
		g.Id("target").Op(":=").Op("&").Qual(typ.PkgPath(), typ.Name()).Block()
		g.If(Op("*").Id(ident).Op("==").Nil()).Block(
			Op("*").Id(ident).Op("=").Id("target"),
		).Else().Block(
			Id("target").Op("=").Op("*").Id(ident),
		).Line()

		for _, code := range codes {
			g.Add(code)
		}
	}).Line().Line(), nil
}

func renderPublicDecodeFunction(c *Converter, typ reflect.Type) (*Statement, error) {
	_, name, decodeFunctionName, _, err := c.GetNamesForType(typ)
	if err != nil {
		return nil, err
	}
	publicName := strings.ToUpper(decodeFunctionName[:1]) + decodeFunctionName[1:]

	return Func().Id(publicName).Params(
		Id("ctx").Qual("context", "Context"),
		Id("getter").Id("Getter"),
		Id(name).Op("**").Qual(typ.PkgPath(), typ.Name()),
	).Qual("github.com/hashicorp/terraform-plugin-framework/diag", "Diagnostics").Block(
		Var().Id("data").Op("*").Id(typ.Name()),
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

func renderEncodeFunction(c *Converter, opts *GeneratorOptions, typ reflect.Type) (*Statement, error) {
	fields, _, err := iterateFields("", opts.GetFieldInformation, typ)
	if err != nil {
		return nil, err
	}

	name, ident, _, encodeFunctionName, err := c.GetNamesForType(typ)
	if err != nil {
		return nil, err
	}

	codes := []Code{}
	for _, field := range fields {
		target := Id(ident)
		if field.Parent != nil {
			target = target.Add(field.Parent.accessor.Clone())
		}
		code, err := c.Encode(
			field,
			target.Add(field.accessor),
			Id("res").Dot(field.goName),
			field.goType,
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
