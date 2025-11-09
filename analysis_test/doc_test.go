// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package analysis_test

import (
	"fmt"

	"github.com/go-openapi/analysis" // This package
	"github.com/go-openapi/loads"    // Spec loading
)

func ExampleSpec() {
	// Example with spec file in this repo
	path := "../fixtures/flatten.yml"
	doc, err := loads.Spec(path) // Load spec from file
	if err == nil {
		an := analysis.New(doc.Spec()) // Analyze spec

		paths := an.AllPaths()
		fmt.Printf("This spec contains %d paths", len(paths))
	}
	// Output: This spec contains 2 paths
}

func ExampleFlatten() {
	// Example with spec file in this repo
	path := "../fixtures/flatten.yml"
	doc, err := loads.Spec(path) // Load spec from file
	if err == nil {
		an := analysis.New(doc.Spec()) // Analyze spec
		// flatten the specification in doc
		erf := analysis.Flatten(analysis.FlattenOpts{Spec: an, BasePath: path})
		if erf == nil {
			fmt.Printf("Specification doc flattened")
		}
		// .. the analyzed spec has been updated and may be now used with the reworked spec
	}
	// Output: Specification doc flattened
}
