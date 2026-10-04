package nexora

import (
	"fmt"
	"strings"
)

func orderRegistrations(registrations []Registration) ([]Registration, error) {
	if err := validateComposition(registrations); err != nil {
		return nil, fmt.Errorf("invalid application compisition: %w", err)
	}

	count := len(registrations)
	if count == 0 {
		return []Registration{}, nil
	}

	// Map each component name to its registration index
	indices := make(map[string]int, count)
	for i, registration := range registrations {
		indices[registration.Name] = i
	}

	indegree := make([]int, count)

	dependents := make([][]int, count)

	for i, registration := range registrations {
		indegree[i] = len(registration.DependsOn)

		for _, dependencyName := range registration.DependsOn {
			dependencyIndex, exists := indices[dependencyName]

			if !exists {
				return nil, fmt.Errorf(
					"component %q depends on unknown component %q",
					registration.Name,
					dependencyName,
				)
			}

			if dependencyIndex == i {
				return nil, fmt.Errorf(
					"component %q cannot depend on itself",
					registration.Name,
				)
			}

			dependents[dependencyIndex] = append(
				dependents[dependencyIndex],
				i,
			)
		}
	}

	ordered := make([]Registration, 0, count)
	processed := make([]bool, count)

	for len(ordered) < count {
		next := -1

		for i := range registrations {
			if !processed[i] && indegree[i] == 0 {
				next = i
				break
			}
		}

		if next == -1 {
			unresolved := make([]string, 0, count)

			for i, registration := range registrations {
				if !processed[i] {
					unresolved = append(unresolved, registration.Name)
				}
			}

			return nil, fmt.Errorf(
				"dependency cycle detected or unresolved dependency chain involving components: %s",
				strings.Join(unresolved, ", "),
			)
		}

		processed[next] = true
		ordered = append(ordered, registrations[next])

		for _, dependentIndex := range dependents[next] {
			indegree[dependentIndex]--
		}
	}

	return ordered, nil
}
