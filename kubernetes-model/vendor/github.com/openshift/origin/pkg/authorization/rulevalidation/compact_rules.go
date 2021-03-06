/**
 * Copyright (C) 2015 Red Hat, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *         http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package rulevalidation

import (
	"reflect"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"

	authorizationapi "github.com/openshift/origin/pkg/authorization/apis/authorization"
)

// CompactRules combines rules that contain a single APIGroup/Resource, differ only by verb, and contain no other attributes.
// this is a fast check, and works well with the decomposed "missing rules" list from a Covers check.
func CompactRules(rules []authorizationapi.PolicyRule) ([]authorizationapi.PolicyRule, error) {
	compacted := make([]authorizationapi.PolicyRule, 0, len(rules))

	simpleRules := map[schema.GroupResource]*authorizationapi.PolicyRule{}
	for _, rule := range rules {
		if resource, isSimple := isSimpleResourceRule(&rule); isSimple {
			if existingRule, ok := simpleRules[resource]; ok {
				// Add the new verbs to the existing simple resource rule
				if existingRule.Verbs == nil {
					existingRule.Verbs = sets.NewString()
				}
				existingRule.Verbs.Insert(rule.Verbs.List()...)
			} else {
				// Copy the rule to accumulate matching simple resource rules into
				ruleCopy := rule.DeepCopy()
				simpleRules[resource] = ruleCopy
			}
		} else {
			compacted = append(compacted, rule)
		}
	}

	// Once we've consolidated the simple resource rules, add them to the compacted list
	for _, simpleRule := range simpleRules {
		compacted = append(compacted, *simpleRule)
	}

	return compacted, nil
}

// isSimpleResourceRule returns true if the given rule contains verbs, a single resource, a single API group, and no other values
func isSimpleResourceRule(rule *authorizationapi.PolicyRule) (schema.GroupResource, bool) {
	resource := schema.GroupResource{}

	// If we have "complex" rule attributes, return early without allocations or expensive comparisons
	if len(rule.ResourceNames) > 0 || len(rule.NonResourceURLs) > 0 || rule.AttributeRestrictions != nil {
		return resource, false
	}
	// If we have multiple api groups or resources, return early
	if len(rule.APIGroups) != 1 || len(rule.Resources) != 1 {
		return resource, false
	}

	// Test if this rule only contains APIGroups/Resources/Verbs
	simpleRule := &authorizationapi.PolicyRule{APIGroups: rule.APIGroups, Resources: rule.Resources, Verbs: rule.Verbs}
	if !reflect.DeepEqual(simpleRule, rule) {
		return resource, false
	}
	resource = schema.GroupResource{Group: rule.APIGroups[0], Resource: rule.Resources.List()[0]}
	return resource, true
}
