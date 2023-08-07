package converters

import (
	"reflect"

	"github.com/Lenstra/terraform-plugin-generator/tags"
	"github.com/dave/jennifer/jen"
)

// MapInterfaceConverter knows how to convert map[string]interface{}
type MapInterfaceConverter struct{}

var _ AttributeConverter = &MapInterfaceConverter{}

func (c *MapInterfaceConverter) Check(typ reflect.Type) (bool, error) {
	switch reflect.Zero(typ).Interface().(type) {
	case map[string]interface{}:
		return true, nil
	}
	return false, nil
}

func (c *MapInterfaceConverter) GetFrameworkType(_ *Converter, typ reflect.Type) (*jen.Statement, error) {
	return jen.Qual("github.com/hashicorp/terraform-plugin-framework/types", "String"), nil
}

func (c *MapInterfaceConverter) Decode(converters *Converter, path, src, target *jen.Statement, typ reflect.Type) (*jen.Statement, error) {
	return jen.If().Op("!").Add(src.Clone()).Dot("IsNull").Call().Block(
		jen.Var().Id("m").Map(jen.String()).Interface(),
		jen.If(jen.Id("err").Op(":=").Qual("encoding/json", "Unmarshal").Call(jen.Index().Byte().Call(src.Clone().Dot("ValueString").Call()), jen.Op("&").Id("m")), jen.Id("err").Op("!=").Nil()).Block(
			jen.Id("diags").Dot("AddAttributeError").Call(path, jen.Lit("failed to unmarshal json"), jen.Id("err").Dot("Error").Call()),
			jen.Return(),
		),
		target.Op("=").Id("m"),
	), nil
}

func (c *MapInterfaceConverter) Encode(converters *Converter, src, target *jen.Statement, typ reflect.Type) (*jen.Statement, error) {
	return jen.If(src.Clone().Op("!=").Nil()).Block(
		jen.List(jen.Id("data"), jen.Id("err")).Op(":=").Qual("encoding/json", "Marshal").Call(src.Clone()),
		jen.If(jen.Id("err").Op("!=").Nil()).Block(
			jen.Panic(jen.Id("err")),
		),
		target.Op("=").Qual("github.com/hashicorp/terraform-plugin-framework/types", "StringValue").Call(jen.String().Call(jen.Id("data"))),
	), nil
}

func (c *MapInterfaceConverter) GetSchema(converters *Converter, path string, info *tags.FieldInformation) (*jen.Statement, *jen.Statement, error) {
	return basicSchema(converters.SchemaImportPath(), "StringAttribute", info, nil)
}

func (c *MapInterfaceConverter) GetType() *jen.Statement {
	return jen.Qual("github.com/hashicorp/terraform-plugin-framework/types", "StringType")
}
