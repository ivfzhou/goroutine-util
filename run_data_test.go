/*
 * Copyright (c) 2023 ivfzhou
 * goroutine-util is licensed under Mulan PSL v2.
 * You can use this software according to the terms and conditions of the Mulan PSL v2.
 * You may obtain a copy of Mulan PSL v2 at:
 *          http://license.coscl.org.cn/MulanPSL2
 * THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND,
 * EITHER EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT,
 * MERCHANTABILITY OR FIT FOR A PARTICULAR PURPOSE.
 * See the Mulan PSL v2 for more details.
 */

package goroutine_util_test

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"

	gu "gitee.com/ivfzhou/goroutine-util"
)

func TestRunData(t *testing.T) {
	ctx := context.Background()
	count := int32(0)
	err := gu.RunData(ctx, func(ctx context.Context, t int32) error {
		atomic.AddInt32(&count, t)
		return nil
	}, true, 1, 2, 3, 4)
	if err != nil {
		t.Error("concurrent: unexpected error", err)
	}
	if count != 10 {
		t.Error("concurrent: unexpected count", count)
	}

	expectedErr := errors.New("expected error")
	err = gu.RunData(ctx, func(ctx context.Context, t int32) error {
		if t == 3 {
			return expectedErr
		}
		return nil
	}, true, 1, 2, 3, 4)
	if err != expectedErr {
		t.Error("concurrent: unexpected error", err)
	}

	err = gu.RunData(ctx, func(ctx context.Context, t int32) error {
		if t == 3 {
			panic(t)
		}
		return nil
	}, true, 1, 2, 3, 4)
	if err == nil || !strings.Contains(err.Error(), "3") {
		t.Error("concurrent: unexpected error", err)
	}
}
