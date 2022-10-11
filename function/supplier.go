// Copyright © 2020, 2022 Yoshiki Shibata. All rights reserved.

package function

// Supplier represents a supplier of results
type Supplier[T any] func() T
