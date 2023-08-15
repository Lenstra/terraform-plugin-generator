package generator

import (
	"fmt"
	"reflect"
)

func iterateFields(path string, infoGetter FieldInformationGetter, typ reflect.Type) ([]*FieldInformation, []reflect.Type, error) {
	// We accept to get a Struct or a pointer to a struct to simplify the code in
	// the converters
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}

	type obj struct {
		Type reflect.Type
		Tag  *FieldInformation
	}
	queue := []obj{{Type: typ, Tag: nil}}

	fields := []*FieldInformation{}
	todo := []reflect.Type{}

	for i := 0; i < len(queue); i++ {
		o := queue[i]
		typ := o.Type

		if typ.Kind() != reflect.Struct {
			return nil, nil, fmt.Errorf("expected Struct, got %s", typ.String())
		}

		for i := 0; i < typ.NumField(); i++ {
			field := typ.Field(i)
			tag, err := infoGetter(path, field)
			if err != nil {
				return nil, nil, err
			}
			if tag == nil {
				continue
			}

			tag.Parent = o.Tag

			if tag.Promoted {
				if tag.Parent != nil {
					return nil, nil, fmt.Errorf("multiple attribute levels have been promoted")
				}
				if field.Type.Kind() == reflect.Pointer {
					field.Type = field.Type.Elem()
				}
				queue = append(queue, obj{Type: field.Type, Tag: tag})
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
