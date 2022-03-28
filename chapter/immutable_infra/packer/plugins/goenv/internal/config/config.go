package config

//go:generate packer-sdc mapstructure-to-hcl2 -type Provisioner

// Provisioner is our provisioner configuration.
type Provisioner struct {
	Version string
}

// Default inputs default values.
func (p *Provisioner) Defaults() {
	if p.Version == "" {
		p.Version = "latest"
	}
}
