// Package searchtest provides reusable contract tests for the search.Index port.
//
// Domain:  Discovery
// Concern: How do we verify that Index adapters satisfy the port contract?
//
// Any Index adapter can validate it satisfies the port contract by calling
// IndexContractTest in its own test file:
//
//	func TestContract(t *testing.T) {
//	    searchtest.IndexContractTest(t, func() search.Index {
//	        return myadapter.New()
//	    })
//	}
package searchtest
