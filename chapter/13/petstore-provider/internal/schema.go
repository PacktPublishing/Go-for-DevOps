package petstore

import (
	"fmt"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

type PetType string

const (
	DogPetType     PetType = "dog"
	CatPetType     PetType = "cat"
	BirdPetType    PetType = "bird"
	ReptilePetType PetType = "reptile"
)

func getPetDataSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"pet_id": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"name": {
			Type:             schema.TypeString,
			Optional:         true,
			ValidateDiagFunc: validateName(),
		},
		"type": {
			Type:             schema.TypeString,
			Optional:         true,
			ValidateDiagFunc: validateType(),
		},
		"birthday": {
			Type:             schema.TypeString,
			Optional:         true,
			ValidateDiagFunc: validateBirthday(),
		},
		"pets": {
			Type:     schema.TypeList,
			Computed: true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"id": {
						Type:     schema.TypeString,
						Computed: true,
					},
					"name": {
						Type:     schema.TypeString,
						Computed: true,
					},
					"type": {
						Type:     schema.TypeString,
						Computed: true,
					},
					"birthday": {
						Type:     schema.TypeString,
						Computed: true,
					},
				},
			},
		},
	}
}

func getPetResourceSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"id": {
			Type:     schema.TypeString,
			Optional: true,
			Computed: true,
		},
		"name": {
			Type:             schema.TypeString,
			Required:         true,
			ValidateDiagFunc: validateName(),
		},
		"type": {
			Type:             schema.TypeString,
			Required:         true,
			ValidateDiagFunc: validateType(),
		},
		"birthday": {
			Type:             schema.TypeString,
			Required:         true,
			ValidateDiagFunc: validateBirthday(),
		},
	}
}

func validateName() schema.SchemaValidateDiagFunc {
	return validateDiagFunc(validation.All(validation.StringIsNotEmpty, validation.StringIsNotWhiteSpace))
}

func validateType() schema.SchemaValidateDiagFunc {
	return validateDiagFunc(validation.StringInSlice([]string{
		string(DogPetType),
		string(CatPetType),
		string(ReptilePetType),
		string(BirdPetType),
	}, true))
}

func validateBirthday() schema.SchemaValidateDiagFunc {
	return validateDiagFunc(validation.IsRFC3339Time)
}

func validateDiagFunc(validateFunc func(interface{}, string) ([]string, []error)) schema.SchemaValidateDiagFunc {
	return func(i interface{}, path cty.Path) diag.Diagnostics {
		warnings, errs := validateFunc(i, fmt.Sprintf("%+v", path))
		var diags diag.Diagnostics
		for _, warning := range warnings {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  warning,
			})
		}
		for _, err := range errs {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  err.Error(),
			})
		}
		return diags
	}
}
