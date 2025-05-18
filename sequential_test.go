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
	"testing"

	gu "gitee.com/ivfzhou/goroutine-util"
)

func ExampleRunSequentially() {
	ctx := context.Background()
	first := func(context.Context) error { return nil }
	then := func(context.Context) error { return nil }
	last := func(context.Context) error { return nil }
	err := gu.RunSequentially(ctx, first, then, last)
	if err != nil {
		// 处理错误。
	}
}

func TestRunSequentially(t *testing.T) {
	t.Run("正常运行", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			expectedResult := 1
			err := gu.RunSequentially(context.Background(),
				func(context.Context) error {
					if expectedResult != 1 {
						t.Errorf("expected result: want 1, got %v", expectedResult)
						return nil
					}
					expectedResult++
					return nil
				},
				nil,
				func(context.Context) error {
					if expectedResult != 2 {
						t.Errorf("expected result: want 2, got %v", expectedResult)
						return nil
					}
					expectedResult++
					return nil
				},
			)
			if err != nil {
				t.Errorf("unexpected error: want nil, got %v", err)
			}
			if expectedResult != 3 {
				t.Errorf("unexpected result: want 3, got %v", expectedResult)
			}
		}
	})

	t.Run("发生错误", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			expectedResult := 0
			expectedErr := errors.New("expected error")
			err := gu.RunSequentially(context.Background(),
				func(context.Context) error {
					expectedResult++
					return nil
				},
				func(context.Context) error {
					expectedResult++
					return expectedErr
				},
				func(context.Context) error {
					expectedResult++
					return nil
				},
			)
			if err == nil || !errors.Is(err, expectedErr) {
				t.Errorf("unexpected error: want %v, got %v", expectedErr, err)
			}
			if expectedResult != 2 {
				t.Errorf("unexpected result: want 2, got %v", expectedResult)
			}
		}
	})

	t.Run("发生恐慌", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			expectedResult := 0
			expectedErr := errors.New("expected error")
			err := gu.RunSequentially(context.Background(),
				func(context.Context) error {
					expectedResult++
					return nil
				},
				func(context.Context) error {
					expectedResult++
					panic(expectedErr)
				},
				func(context.Context) error {
					expectedResult++
					return nil
				},
			)
			if err == nil || !errors.Is(err, expectedErr) {
				t.Errorf("unexpected error: want %v, got %v", expectedErr, err)
			}
			if expectedResult != 2 {
				t.Errorf("unexpected result: want 2, got %v", expectedResult)
			}
		}
	})

	t.Run("上下文终止", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			expectedErr := errors.New("expected error")
			ctx, cancel := NewCtxCancelWithError()
			expectedResult := 0
			err := gu.RunSequentially(ctx,
				func(context.Context) error {
					expectedResult++
					return nil
				},
				func(context.Context) error {
					expectedResult++
					cancel(expectedErr)
					return nil
				},
				func(context.Context) error {
					expectedResult++
					return nil
				},
			)
			if err == nil || !errors.Is(err, expectedErr) {
				t.Errorf("unexpected error: want %v, got %v", expectedErr, err)
			}
			if expectedResult != 2 {
				t.Errorf("unexpected result: want 2, got %v", expectedResult)
			}
		}
	})
}
