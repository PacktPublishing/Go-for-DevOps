terraform {
  required_providers {
    petstore = {
      version = "0.1.0"
      source  = "example.com/gofordevops/petstore"
    }
  }
}

provider "petstore" {
  host = "127.0.0.1:6742"
}

resource "petstore_pet" "thor" {
  name     = "Thor"
  type     = "dog"
  birthday = "2021-04-01T00:00:00Z"
}

resource "petstore_pet" "tron" {
  name     = "Tron"
  type     = "cat"
  birthday = "2020-06-25T00:00:00Z"
}

variable "pet_name" {
  type    = string
  default = "Thor"
}

data "petstore_pet" "all" {
  depends_on = [petstore_pet.thor, petstore_pet.tron]
}

# Returns all pets
output "all_pets" {
  value = data.petstore_pet.all
}

# Only returns Thor by name
output "thor" {
  value = {
    for pet in data.petstore_pet.all.pets :
    pet.id => pet
    if pet.name == var.pet_name
  }
}
