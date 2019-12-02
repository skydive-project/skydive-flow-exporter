/*
 * Copyright (C) 2019 IBM, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy ofthe License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specificlanguage governing permissions and
 * limitations under the License.
 *
 */

package pkg

import (
	"bytes"
	"encoding/json"
	"strings"
	"text/template"
	"time"

	cache "github.com/pmylund/go-cache"
	"github.com/skydive-project/skydive/api/client"
	"github.com/skydive-project/skydive/contrib/exporters/core"
	"github.com/skydive-project/skydive/logging"
	"github.com/spf13/viper"
)

// extendGremlin is used for transforming extensions using gremlin expressions
// Syntax supported is:
//	VAR_NAME=<gremlin expression with substitution strings>
// Substitution strings are defined according to golang template usage using {{ and }}
// For example:
//	AA_Name=G.V().Has('RoutingTables.Src','{{.Network.A}}').Values('Host')
// The substitution string refers to fields that exist in the data provided by the particular transform
// If needed, be sure to put quotes around the substitution results
// Use only single quotes in the gremlin expression
// Need to define `Extend` field as a map[string]interface{} subfield of `transform`
// Structure in yml file is as follows:
//   transform:
//     type: secadvisor
//     secadvisor:
//       extend:
//         - VAR_NAME1=<gremlin expression with substitution strings>
//         - VAR_NAME2=<gremlin expression with substitution strings>
type extendGremlin struct {
	gremlinClient    *client.GremlinQueryHelper
	gremlinExprCache *cache.Cache
	gremlinTemplates []*template.Template
	newVars          []string
	nTemplates       int
}

// NewExtendGremlin creates an extendGremlin structure to handle the defined data extensions
func NewExtendGremlin(cfg *viper.Viper) *extendGremlin {
	gremlinClient := client.NewGremlinQueryHelper(core.CfgAuthOpts(cfg))
	extendStrings := cfg.GetStringSlice(core.CfgRoot + "transform.secadvisor.extend")
	nTemplates := 0
	gremlinTemplates := make([]*template.Template, len(extendStrings))
	newVars := make([]string, len(extendStrings))
	// parse the extension expressions and save them for future use
	for _, ee := range extendStrings {
		logging.GetLogger().Debugf("extend gremlin expression: %s", ee)
		substrings := strings.SplitN(ee, "=", 2)
		vv, err := template.New("extend_template").Parse(substrings[1])
		if err != nil {
			logging.GetLogger().Errorf("NewExtendGremlin: template error: %s, template string: %s", err, substrings[1])
			continue
		}
		gremlinTemplates[nTemplates] = vv
		newVars[nTemplates] = substrings[0]
		nTemplates++
	}
	return &extendGremlin{
		gremlinClient:    gremlinClient,
		gremlinExprCache: cache.New(10*time.Minute, 10*time.Minute),
		gremlinTemplates: gremlinTemplates,
		newVars:          newVars,
		nTemplates:       nTemplates,
	}
}

// Extend: main function called to perform the extension substitutions
// we expect the data of a single flow per call
func (e *extendGremlin) Extend(in *SecurityAdvisorFlow) {
	for i := 0; i < e.nTemplates; i++ {
		var tpl bytes.Buffer
		// apply the gremlin expression
		err := e.gremlinTemplates[i].Execute(&tpl, in)
		if err != nil {
			continue
		}
		result := tpl.String()
		// check if result is already in cache; if not, perform gremlin request
		result2, ok := e.gremlinExprCache.Get(result)
		if !ok {
			var result4 []string
			// need to perform the gremlin query
			// cache an empty string if conversion fails for any reason
			result2 = ""
			result3, err := e.gremlinClient.Query(result)
			if err != nil {
				logging.GetLogger().Errorf("Extend: gremlin query error; err = %s, query = %s", err, result)
			} else if len(result3) == 0 {
				logging.GetLogger().Errorf("Extend: gremlin empty query; result = %s", result)
			} else if err = json.Unmarshal(result3, &result4); err != nil {
				logging.GetLogger().Errorf("Extend: unmarshal error; err = %s", err)
			} else if len(result4) > 0 {
				result2 = result4[0]
			}
			// add new field in map with the desired key name
			e.gremlinExprCache.Set(result, result2, cache.DefaultExpiration)
		}
		in.Extend[e.newVars[i]] = result2
	}
}
