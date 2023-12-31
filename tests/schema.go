/*
Code generated by github-terraform-generator; DO NOT EDIT.
Any modifications will be overwritten
*/

package tests

import schema "github.com/hashicorp/terraform-plugin-framework/resource/schema"

func coffeeSchema() schema.Schema {
	return schema.Schema{
		MarkdownDescription: "",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Optional:   true,
				Default:    nil,
				Validators: nil,
			},
			"name": schema.StringAttribute{
				Required:   true,
				Default:    nil,
				Validators: nil,
			},
			"teaser": schema.StringAttribute{
				Optional:   true,
				Default:    nil,
				Validators: nil,
			},
			"description": schema.StringAttribute{
				Optional:   true,
				Default:    nil,
				Validators: nil,
			},
			"image": schema.StringAttribute{
				Optional:   true,
				Default:    nil,
				Validators: nil,
			},
			"ingredients": &schema.ListNestedAttribute{
				Optional: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							Required:   true,
							Default:    nil,
							Validators: nil,
						},
						"float32": schema.Float64Attribute{
							Optional:   true,
							Default:    nil,
							Validators: nil,
						},
						"float64": schema.Float64Attribute{
							Optional:   true,
							Default:    nil,
							Validators: nil,
						},
					}},
			},
			"customer": &schema.SingleNestedAttribute{
				Optional:   true,
				Default:    nil,
				Validators: nil,
				Attributes: map[string]schema.Attribute{
					"id": schema.Int64Attribute{
						Optional:   true,
						Default:    nil,
						Validators: nil,
					},
					"name": schema.StringAttribute{
						Optional:   true,
						Default:    nil,
						Validators: nil,
					},
				},
			},
		},
		Blocks: map[string]schema.Block{},
	}
}

func configSchema() schema.Schema {
	return schema.Schema{
		MarkdownDescription: "",
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				Required:   true,
				Default:    nil,
				Validators: nil,
			},
			"bool": schema.BoolAttribute{
				Optional:   true,
				Default:    nil,
				Validators: nil,
			},
			"int": schema.Int64Attribute{
				Optional:   true,
				Default:    nil,
				Validators: nil,
			},
			"string": schema.StringAttribute{
				Optional:   true,
				Default:    nil,
				Validators: nil,
			},
		},
		Blocks: map[string]schema.Block{},
	}
}

func ingredientSchema() schema.Schema {
	return schema.Schema{
		MarkdownDescription: "",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Required:   true,
				Default:    nil,
				Validators: nil,
			},
			"float32": schema.Float64Attribute{
				Optional:   true,
				Default:    nil,
				Validators: nil,
			},
			"float64": schema.Float64Attribute{
				Optional:   true,
				Default:    nil,
				Validators: nil,
			},
		},
		Blocks: map[string]schema.Block{},
	}
}
