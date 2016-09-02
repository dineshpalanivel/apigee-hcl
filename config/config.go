package config

import (
	"fmt"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/hcl/hcl/ast"
	"github.com/kevinswiber/apigee-hcl/config/hclerror"
	"github.com/kevinswiber/apigee-hcl/config/policy"
)

// Config is a container for holding the contents of an exported Apigee proxy bundle
type Config struct {
	Proxy           *Proxy
	ProxyEndpoints  []*ProxyEndpoint
	TargetEndpoints []*TargetEndpoint
	Policies        []interface{}
	Resources       map[string]string
}

// LoadConfigFromHCL converts an HCL ast.ObjectList into a Config object
func LoadConfigFromHCL(list *ast.ObjectList) (*Config, error) {
	var errors *multierror.Error

	var c Config

	if proxies := list.Filter("proxy"); len(proxies.Items) > 0 {
		result, err := loadProxyHCL(proxies)
		if err != nil {
			errors = multierror.Append(errors, err)
			return nil, errors
		}

		c.Proxy = result
	}

	if proxyEndpoints := list.Filter("proxy_endpoint"); len(proxyEndpoints.Items) > 0 {
		result, err := loadProxyEndpointsHCL(proxyEndpoints)
		if err != nil {
			errors = multierror.Append(errors, err)
			return nil, errors
		}

		c.ProxyEndpoints = result
	}

	if targetEndpoints := list.Filter("target_endpoint"); len(targetEndpoints.Items) > 0 {
		result, err := loadTargetEndpointsHCL(targetEndpoints)
		if err != nil {
			errors = multierror.Append(errors, err)
			return nil, errors
		}

		c.TargetEndpoints = result
	}

	if policies := list.Filter("policy"); len(policies.Items) > 0 {
		var ps []interface{}

		for _, item := range policies.Items {
			if len(item.Keys) < 2 ||
				item.Keys[0].Token.Value() == "" ||
				item.Keys[1].Token.Value() == "" {
				pos := item.Val.Pos()
				newError := hclerror.PosError{
					Pos: pos,
					Err: fmt.Errorf("policy requires a type and name"),
				}

				errors = multierror.Append(errors, &newError)
				continue
			}
			policyType := item.Keys[0].Token.Value().(string)

			if f, ok := PolicyList[policyType]; ok {
				p, err := f(item)
				if err != nil {
					errors = multierror.Append(errors, err)
				}

				switch p.(type) {
				case policy.ScriptPolicy:
					script := p.(policy.ScriptPolicy)
					if len(script.ResourceURL) > 0 && len(script.Content) > 0 {
						if c.Resources == nil {
							c.Resources = make(map[string]string)
						}
						c.Resources[script.ResourceURL] = script.Content
					}
				case policy.JavaScriptPolicy:
					script := p.(policy.JavaScriptPolicy)
					if len(script.ResourceURL) > 0 && len(script.Content) > 0 {
						if c.Resources == nil {
							c.Resources = make(map[string]string)
						}
						c.Resources[script.ResourceURL] = script.Content
					}
				}
				ps = append(ps, p)
			}
		}

		if errors != nil {
			return nil, errors
		}

		c.Policies = ps
	}
	return &c, nil
}
