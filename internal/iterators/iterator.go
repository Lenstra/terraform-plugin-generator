package iterators

import (
	"fmt"
	"reflect"

	"github.com/Lenstra/terraform-plugin-generator/tags"
	"github.com/dave/jennifer/jen"
)

func IterateFields(path string, infoGetter tags.FieldInformationGetter, typ reflect.Type) ([]*tags.FieldInformation, []reflect.Type, error) {
	// We accept to get a Struct or a pointer to a struct to simplify the code in
	// the converters
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct {
		return nil, nil, fmt.Errorf("expected Struct, got %s", typ.String())
	}

	type obj struct {
		Type     reflect.Type
		Accessor *jen.Statement
	}
	queue := []obj{{Type: typ, Accessor: jen.Empty()}}

	fields := []*tags.FieldInformation{}
	todo := []reflect.Type{}

	for i := 0; i < len(queue); i++ {
		o := queue[i]
		typ := o.Type

		for i := 0; i < typ.NumField(); i++ {
			field := typ.Field(i)
			tag, err := infoGetter(path, field)
			if err != nil {
				return nil, nil, err
			}
			if tag == nil {
				continue
			}

			tag.Accessor = o.Accessor.Clone().Add(tag.Accessor)

			if tag.Promoted {
				queue = append(queue, obj{Type: field.Type, Accessor: tag.Accessor})
				continue
			}

			fieldType := field.Type
			for fieldType.Kind() == reflect.Pointer || fieldType.Kind() == reflect.Slice || fieldType.Kind() == reflect.Map {
				fieldType = fieldType.Elem()
			}
			if fieldType.Kind() == reflect.Struct {
				todo = append(todo, fieldType)
			}

			tag.Path = path + "." + tag.Name

			fields = append(fields, tag)
		}
	}

	return fields, todo, nil
}
