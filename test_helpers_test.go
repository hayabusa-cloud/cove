// ©Hayabusa Cloud Co., Ltd. 2026. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cove_test

import (
	"testing"

	"code.hybscloud.com/kont"
)

type ping struct{ kont.Phantom[int] }

func expectPanic(t *testing.T, f func()) (got any) {
	t.Helper()
	defer func() {
		got = recover()
		if got == nil {
			t.Fatal("expected panic")
		}
	}()
	f()
	return nil
}

func expectPanicMessage(t *testing.T, want string, f func()) {
	t.Helper()
	if got := expectPanic(t, f); got != want {
		t.Fatalf("unexpected panic value: %v", got)
	}
}
