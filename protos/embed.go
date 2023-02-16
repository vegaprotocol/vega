// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package protos

import (
	"embed"
	"strings"

	"gopkg.in/yaml.v2"
)

// We are going to use the path defined in the rules to ensure that only valid paths
// are recorded by the metrics. Rather than keeping a separate list in code that needs
// to be updated. We can embed the bindings files as the paths will need to be maintained
// here anyway.

//go:embed sources/vega/grpc-rest-bindings.yml sources/data-node/grpc-rest-bindings.yml
var FS embed.FS

type Rule struct {
	Selector string  `yaml:"selector"`
	Post     *string `yaml:"post"`
	Get      *string `yaml:"get"`
	Body     *string `yaml:"body"`
}

type Rules struct {
	Rules []Rule `yaml:"rules"`
}

type Bindings struct {
	HTTP Rules `yaml:"http"`
}

func CoreBindings() (*Bindings, error) {
	b, err := FS.ReadFile("sources/vega/grpc-rest-bindings.yml")
	if err != nil {
		return nil, err
	}

	var bindings Bindings
	err = yaml.Unmarshal(b, &bindings)
	if err != nil {
		return nil, err
	}

	return &bindings, nil
}

func DataNodeBindings() (*Bindings, error) {
	b, err := FS.ReadFile("sources/data-node/grpc-rest-bindings.yml")
	if err != nil {
		return nil, err
	}

	var bindings Bindings
	err = yaml.Unmarshal(b, &bindings)
	if err != nil {
		return nil, err
	}

	return &bindings, nil
}

func (b *Bindings) HasRoute(method, path string) bool {
	method = strings.ToLower(method)
	for _, rule := range b.HTTP.Rules {
		var route string
		switch method {
		case "get":
			if rule.Get == nil {
				continue
			}
			route = *rule.Get
		case "post":
			if rule.Post == nil {
				continue
			}
			route = *rule.Post
		default:
			return false
		}

		if strings.Contains(route, "/{") {
			route = strings.Split(route, "/{")[0]
		}

		if strings.Contains(path, route) {
			return true
		}
	}

	return false
}
