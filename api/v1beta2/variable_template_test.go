/*
 * variable_template_test.go
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
	monitorapi "github.com/apple/foundationdb/fdbkubernetesmonitor/api"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func literalArgument(value string) monitorapi.Argument {
	return monitorapi.Argument{ArgumentType: monitorapi.LiteralArgumentType, Value: value}
}

func environmentArgument(source string) monitorapi.Argument {
	return monitorapi.Argument{ArgumentType: monitorapi.EnvironmentArgumentType, Source: source}
}

var _ = Describe("ParseVariableTemplate", func() {
	DescribeTable("parsing a valid value",
		func(value string, expected []monitorapi.Argument) {
			Expect(ParseVariableTemplate(value)).To(Equal(expected))
		},
		Entry("a literal",
			"rack-1",
			[]monitorapi.Argument{literalArgument("rack-1")}),
		Entry("an empty value",
			"",
			[]monitorapi.Argument{literalArgument("")}),
		Entry("a single variable",
			"$RACK",
			[]monitorapi.Argument{environmentArgument("RACK")}),
		Entry("a single variable keeps the whole name, as before templates were supported",
			"$NODE_LABEL_FAILURE-DOMAIN_BETA_KUBERNETES_IO_ZONE",
			[]monitorapi.Argument{
				environmentArgument("NODE_LABEL_FAILURE-DOMAIN_BETA_KUBERNETES_IO_ZONE"),
			}),
		Entry("a single variable in braces",
			"${RACK}",
			[]monitorapi.Argument{environmentArgument("RACK")}),
		Entry("multiple variables",
			"$NODE_LABEL_TOPOLOGY_KUBERNETES_IO_REGION-$NODE_LABEL_INFRA_X_COM_RACK",
			[]monitorapi.Argument{
				environmentArgument("NODE_LABEL_TOPOLOGY_KUBERNETES_IO_REGION"),
				literalArgument("-"),
				environmentArgument("NODE_LABEL_INFRA_X_COM_RACK"),
			}),
		Entry("a literal prefix",
			"pdxa-$RACK",
			[]monitorapi.Argument{literalArgument("pdxa-"), environmentArgument("RACK")}),
		Entry("braces separate a variable from following identifier characters",
			"${REGION}_rack$RACK.x",
			[]monitorapi.Argument{
				environmentArgument("REGION"),
				literalArgument("_rack"),
				environmentArgument("RACK"),
				literalArgument(".x"),
			}),
		Entry("braces allow \"-\" and \".\" in the variable name",
			"${NODE_LABEL_FAILURE-DOMAIN.X}/$RACK",
			[]monitorapi.Argument{
				environmentArgument("NODE_LABEL_FAILURE-DOMAIN.X"),
				literalArgument("/"),
				environmentArgument("RACK"),
			}),
	)

	DescribeTable("parsing a malformed value",
		func(value string) {
			_, err := ParseVariableTemplate(value)
			Expect(err).To(HaveOccurred())
		},
		Entry("a bare $", "$"),
		Entry("a trailing $", "rack-$"),
		Entry("a $ followed by a non-identifier character", "$REGION-$-rack"),
		Entry("a doubled $", "$$RACK"),
		Entry("an unterminated ${", "${RACK"),
		Entry("an unterminated ${ followed by a variable", "${REGION-$RACK"),
		Entry("an empty ${}", "rack-${}"),
		Entry("a variable name starting with a digit", "rack-${1RACK}"),
	)
})
