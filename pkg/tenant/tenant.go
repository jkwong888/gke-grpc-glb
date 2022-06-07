package tenant

import (
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	"gopkg.in/yaml.v3"
)

/* TODO: how to define a RANGE of tenants?  e.g. 0000-4999 is valid for this deployment */
type TenantConfig struct {
	AllowedTenants 	[]TenantMatch `yaml:"allowed_tenants" json:"allowed_tenants"`
	DeniedTenants 	[]TenantMatch `yaml:"denied_tenants" json:"denied_tenants"`
}

type TenantMatch struct {
	RangeMatch 	*[]TenantRangeMatch `yaml:"range,omitempty" json:"range,omitempty"`
	PrefixMatch *[]string 			`yaml:"prefix,omitempty" json:"prefix,omitempty"`
	ExactMatch 	*[]string 			`yaml:"exactMatch,omitempty" json:"exactMatch,omitempty"`
}

type TenantRangeMatch struct {
	Start string `yaml:"start" json:"start"`
	End   string `yaml:"end" json:"end"`
}

func makeDefaultTenantConfig() (*TenantConfig) {
	defaultTenantConfig := TenantConfig{}
	defaultTenantConfig.AllowedTenants = make([]TenantMatch, 1)
	defaultTenantConfig.AllowedTenants[0].ExactMatch = &[]string{"*"}
	defaultTenantConfig.DeniedTenants = []TenantMatch{}

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

func tenantMatches(tenantIdToCheck string, tm TenantMatch) bool {
	if tm.ExactMatch != nil {
		for _, t := range *tm.ExactMatch {
			if tenantIdToCheck == t {
				return true
			}
			
			// special case: "*"
			if t == "*" {
				return true
			}
		}

		return false
	}


	if tm.PrefixMatch != nil {
		for _, t := range *tm.PrefixMatch {
			if strings.HasPrefix(tenantIdToCheck, t) {
				return true
			}
		}

		return false
	}

	if tm.RangeMatch != nil {
		for _, t := range *tm.RangeMatch {
			if tenantIdToCheck < t.Start {
				continue
			}
			if tenantIdToCheck > t.End {
				continue	
			}

			// string is within range here
			return true
		}

	}

	// how did we get here?  nothing is defined
	return false
}

func (t *TenantConfig) CheckTenantId(tenantIdToCheck string) bool {
	// check if tenant is explicitly denied 
	for _, tenantMatch := range t.DeniedTenants {
		if tenantMatches(tenantIdToCheck, tenantMatch) {
			// if the tenant matches a deny rule, see if it's allowed later
			break
		}
	}

	for _, tenantMatch := range t.AllowedTenants {
		if tenantMatches(tenantIdToCheck, tenantMatch) {
			return true
		}

	}

	return false
}