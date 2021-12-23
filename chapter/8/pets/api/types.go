package api

import "time"

type (
	// PetType is the type of pet, Dog, Cat, or Fish
	PetType string

	// Pet is a structure describing an available Pet
	Pet struct {
		// Name of the pet
		Name string `json:"name,omitempty"`
		// Type of the pet
		Type PetType `json:"type,omitempty"`
		// Birthday of the pet
		Birthday time.Time `json:"birthday,omitempty"`
	}
)

var (
	Dog  PetType = "dog"
	Cat  PetType = "cat"
	Fish PetType = "fish"
)
