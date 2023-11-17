// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

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
