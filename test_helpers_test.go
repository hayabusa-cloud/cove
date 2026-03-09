// ©Hayabusa Cloud Co., Ltd. 2026. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cove_test

import (
	"testing"

	"code.hybscloud.com/kont"
)

type ping struct{ kont.Phantom[int] }

func expectPanicMessage(t *testing.T, want string, f func()) {
	t.Helper()
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic")
		}
		if s, ok := r.(string); !ok || s != want {
			t.Fatalf("unexpected panic value: %v", r)
		}
	}()
	f()
}
