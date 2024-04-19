/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef_test

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
)

func ExampleQNames() {

	product, order, customer := appdef.NewQName("test", "product"), appdef.NewQName("test", "order"), appdef.NewQName("test", "customer")

	// Create empty QNames
	qnames := appdef.QNames{}

	// Add some QNames
	qnames.Add(product, order, customer, product)

	// Iterate over QNames
	for _, qname := range qnames {
		fmt.Println(qname)
	}

	// Check is QNames contains some QName
	fmt.Println(qnames.Contains(product))
	fmt.Println(qnames.Contains(appdef.NewQName("test", "unknown")))

	// Find QName by name
	fmt.Println(qnames.Find(order))
	fmt.Println(qnames.Find(appdef.NewQName("test", "data")))

	// Print length and content
	fmt.Println(len(qnames), qnames)

	// Output:
	// test.customer
	// test.order
	// test.product
	// true
	// false
	// 1 true
	// 1 false
	// 3 [test.customer test.order test.product]
}
