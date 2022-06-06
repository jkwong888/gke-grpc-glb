package tenant

import (
	"fmt"
	"io/ioutil"
	"log"

	"gopkg.in/yaml.v3"
)

/* TODO: how to define a RANGE of tenants?  e.g. 0000-4999 is valid for this deployment */
type TenantConfig struct {
	AllowedTenants []string `yaml:"allowed_tenants"`
	DeniedTenants []string `yaml:"denied_tenants"`
}

func makeDefaultTenantConfig() (*TenantConfig) {
	defaultTenantConfig := TenantConfig{}
	defaultTenantConfig.AllowedTenants = []string{"*"}
	defaultTenantConfig.DeniedTenants = []string{}

	return &defaultTenantConfig
}

func LoadTenantConfig(configDir string) (*TenantConfig) {
	t := &TenantConfig{}

	yamlFile, err := ioutil.ReadFile(fmt.Sprintf("%v/tenant-config.yaml", configDir))
	if err != nil {
		log.Printf("Unable to load tenantConfig: %v, accept all tenants", err.Error())
		return makeDefaultTenantConfig()
	}

	err = yaml.Unmarshal(yamlFile, t)
	if err != nil {
		log.Printf("Could not parse tenantConfig: %v, accept all tenants", err.Error())
		return makeDefaultTenantConfig()
	}

	return t
}

func (t *TenantConfig) CheckTenantId(tenantIdToCheck string) bool {
	/* check if tenant is explicitly denied */
	for _, tenantId := range t.DeniedTenants {
		if tenantId == "*" {
			/* special case all tenants are denied -- but we may explicitly allow only this tenant so skip down to the allowed tenants */
			break
		}
		if tenantId == tenantIdToCheck {
			/* we've specifically denied this tenant, just return the error immediately */
			return false
		}
	}

	for _, tenantId := range t.AllowedTenants {
		if tenantId == "*" {
			/* special case all tenants are allowed */
			return true
		}

		if tenantId == tenantIdToCheck {
			/* we've specifically allowed this tenant, just return true */
			return true
		}
	}

	return false
}