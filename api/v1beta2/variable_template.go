/*
 * variable_template.go
 *
 * This source file is part of the FoundationDB open source project
 *
 * Copyright 2018-2026 Apple Inc. and the FoundationDB project authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package v1beta2

import (
	"fmt"
	"regexp"
	"strings"

	monitorapi "github.com/apple/foundationdb/fdbkubernetesmonitor/api"
)

// variableReference matches $NAME and ${NAME}. Inside braces "-" and "." are allowed as well, so every valid
// Kubernetes environment variable name can be referenced, e.g. a NODE_LABEL_* variable of a label key with a "-".
var variableReference = regexp.MustCompile(
	`\$(?:([A-Za-z_][A-Za-z0-9_]*)|\{([-._A-Za-z][-._A-Za-z0-9]*)\})`,
)

// ParseVariableTemplate parses a value that mixes literal text with references to environment variables, written as
// $NAME or ${NAME}, e.g. "$REGION-$RACK", into literal and environment arguments for the fdb-kubernetes-monitor.
// A value without a "$" is a single literal. A value that starts with "$", contains no other "$" and doesn't start
// with "${" references a single variable named by the rest of the value, which is how such values were handled
// before templates were supported, e.g. "$NODE_LABEL_A-B" references the variable "NODE_LABEL_A-B". Every other "$"
// must start a valid reference, otherwise an error is returned.
func ParseVariableTemplate(value string) ([]monitorapi.Argument, error) {
	if !strings.Contains(value, "$") {
		return []monitorapi.Argument{
			{ArgumentType: monitorapi.LiteralArgumentType, Value: value},
		}, nil
	}

	name, hasPrefix := strings.CutPrefix(value, "$")
	if hasPrefix && name != "" && !strings.Contains(name, "$") && !strings.HasPrefix(name, "{") {
		return []monitorapi.Argument{
			{ArgumentType: monitorapi.EnvironmentArgumentType, Source: name},
		}, nil
	}

	// Every "$" starts a variable with at most one literal before it.
	arguments := make([]monitorapi.Argument, 0, 2*strings.Count(value, "$")+1)
	addLiteral := func(start int, end int) error {
		literal := value[start:end]
		if idx := strings.Index(literal, "$"); idx >= 0 {
			return fmt.Errorf(
				"invalid variable reference at position %d in %q, use $NAME or ${NAME}",
				start+idx,
				value,
			)
		}

		if literal != "" {
			arguments = append(arguments, monitorapi.Argument{
				ArgumentType: monitorapi.LiteralArgumentType,
				Value:        literal,
			})
		}

		return nil
	}

	position := 0
	for _, match := range variableReference.FindAllStringSubmatchIndex(value, -1) {
		err := addLiteral(position, match[0])
		if err != nil {
			return nil, err
		}

		// match[2:4] is the $NAME group and match[4:6] the ${NAME} group.
		start, end := match[2], match[3]
		if start < 0 {
			start, end = match[4], match[5]
		}

		arguments = append(arguments, monitorapi.Argument{
			ArgumentType: monitorapi.EnvironmentArgumentType,
			Source:       value[start:end],
		})
		position = match[1]
	}

	err := addLiteral(position, len(value))
	if err != nil {
		return nil, err
	}

	return arguments, nil
}
