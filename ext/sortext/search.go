// Iris - Decentralized Messaging Framework
// Copyright 2013 Peter Szilagyi. All rights reserved.
//
// Iris is dual licensed: you can redistribute it and/or modify it under the
// terms of the GNU General Public License as published by the Free Software
// Foundation, either version 3 of the License, or (at your option) any later
// version.
//
// The framework is distributed in the hope that it will be useful, but WITHOUT
// ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or
// FITNESS FOR A PARTICULAR PURPOSE.  See the GNU General Public License for
// more details.
//
// Alternatively, the Iris framework may be used in accordance with the terms
// and conditions contained in a signed written agreement between you and the
// author(s).
//
// Author: peterke@gmail.com (Peter Szilagyi)

package sortext

import (
	"math/big"
	"sort"
)

// SearchBigInts searches for x in a sorted slice of *big.Ints and returns the
// index as specified by Search. The return value is the index to insert x if x
// is not present (it could be len(a)).
// The slice must be sorted in ascending order.
func SearchBigInts(a []*big.Int, x *big.Int) int {
	return sort.Search(len(a), func(i int) bool { return a[i].Cmp(x) >= 0 })
}

// SearchBigRats searches for x in a sorted slice of *big.Rats and returns the
// index as specified by Search. The return value is the index to insert x if x
// is not present (it could be len(a)).
// The slice must be sorted in ascending order.
func SearchBigRats(a []*big.Rat, x *big.Rat) int {
	return sort.Search(len(a), func(i int) bool { return a[i].Cmp(x) >= 0 })
}

// Search returns the result of applying SearchBigInts to the receiver and x.
func (p BigIntSlice) Search(x *big.Int) int { return SearchBigInts(p, x) }

// Search returns the result of applying SearchBigRats to the receiver and x.
func (p BigRatSlice) Search(x *big.Rat) int { return SearchBigRats(p, x) }
