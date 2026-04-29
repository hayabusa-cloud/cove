// ©Hayabusa Cloud Co., Ltd. 2026. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cove

// Ambient constrains ambient context type parameters.
type Ambient interface{}

// World constrains Kripke world type parameters.
//
// It is intentionally the same structural constraint as Ambient: Kripke worlds
// in cove are ambient contexts viewed through preorder and forcing evidence, not
// a separate runtime state plane.
type World interface{}

// Focus constrains type parameters that represent focused values.
type Focus interface{}
