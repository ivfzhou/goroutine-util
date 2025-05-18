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
	"math/rand"
	"sync/atomic"
	"testing"
	"time"

	gu "gitee.com/ivfzhou/goroutine-util"
)

func ExampleRunConcurrently() {
	ctx := context.Background()

	work1 := func(ctx context.Context) error {
		// 编写你的业务逻辑。
		return nil
	}

	work2 := func(ctx context.Context) error {
		// 编写你的业务逻辑。
		return nil
	}

	err := gu.RunConcurrently(ctx, work1, work2)(true)
	if err != nil {
		// 处理发生的错误。
		return
	}
}

func TestRunConcurrently(t *testing.T) {
	t.Run("正常运行", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			const fnCount = 1000
			expectedResult := int32(0)
			fns := make([]func(context.Context) error, fnCount)
			for i := 0; i < fnCount; i++ {
				fns[i] = func(context.Context) error {
					time.Sleep(time.Millisecond * 10)
					atomic.AddInt32(&expectedResult, 1)
					return nil
				}
			}
			fastExit := rand.Intn(2) == 1
			ctx := context.Background()
			if rand.Intn(2) == 1 {
				ctx = nil
			}
			err := gu.RunConcurrently(ctx, fns...)(fastExit)
			if err != nil {
				t.Errorf("unexpected error: want nil, got %v", err)
			}
			if result := atomic.LoadInt32(&expectedResult); result != fnCount {
				t.Errorf("unexpected result: want %v, got %v", fnCount, result)
			}
		}
	})

	t.Run("发生错误", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			expectedErr := errors.New("expected error")
			const fnCount = 1000
			expectedResult := int32(0)
			occurErrorIndex := rand.Intn(fnCount)
			fns := make([]func(context.Context) error, fnCount)
			for i := 0; i < fnCount; i++ {
				index := i
				fns[i] = func(context.Context) error {
					if index == occurErrorIndex {
						return expectedErr
					}
					time.Sleep(time.Millisecond * 10)
					atomic.AddInt32(&expectedResult, 1)
					return nil
				}
			}
			fastExit := rand.Intn(2) == 1
			err := gu.RunConcurrently(context.Background(), fns...)(fastExit)
			if !errors.Is(err, expectedErr) {
				t.Errorf("unexpected error: want %v, got %v", expectedErr, err)
			}
			if result := atomic.LoadInt32(&expectedResult); int(result) >= fnCount {
				t.Errorf("unexpected result: want < %v, got %v", fnCount, result)
			}
		}
	})

	t.Run("发生恐慌", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			expectedErr := errors.New("expected error")
			const fnCount = 1000
			expectedResult := int32(0)
			panicIndex := rand.Intn(fnCount)
			fns := make([]func(context.Context) error, fnCount)
			for i := 0; i < fnCount; i++ {
				index := i
				fns[i] = func(context.Context) error {
					if index == panicIndex {
						panic(expectedErr)
					}
					time.Sleep(time.Millisecond * 10)
					atomic.AddInt32(&expectedResult, 1)
					return nil
				}
			}
			fastExit := rand.Intn(2) == 1
			err := gu.RunConcurrently(context.Background(), fns...)(fastExit)
			if !errors.Is(err, expectedErr) {
				t.Errorf("unexpected error: want %v, got %v", expectedErr, err)
			}
			if result := atomic.LoadInt32(&expectedResult); result >= fnCount {
				t.Errorf("unexpected result: want < %v, got %v", fnCount, result)
			}
		}
	})

	t.Run("上下文终止", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			ctx, cancel := NewCtxCancelWithError()
			expectedErr := errors.New("expected error")
			const fnCount = 1000
			expectedResult := int32(0)
			cancelIndex := rand.Intn(fnCount / 2)
			fns := make([]func(context.Context) error, fnCount)
			for i := 0; i < fnCount; i++ {
				index := i
				fns[i] = func(context.Context) error {
					if index == cancelIndex {
						cancel(expectedErr)
					}
					time.Sleep(time.Millisecond * 10)
					atomic.AddInt32(&expectedResult, 1)
					return nil
				}
			}
			fastExit := rand.Intn(2) == 1
			err := gu.RunConcurrently(ctx, fns...)(fastExit)
			if !errors.Is(err, expectedErr) {
				t.Errorf("unexpected error: want %v, got %v", expectedErr, err)
			}
			if result := atomic.LoadInt32(&expectedResult); result >= fnCount {
				t.Errorf("unexpected result: want < %v, got %v", fnCount, result)
			}
		}
	})

	t.Run("大量函数", func(t *testing.T) {
		var fnCount = 1024*1024*(rand.Intn(5)+1) + 10
		expectedResult := int32(0)
		fns := make([]func(context.Context) error, fnCount)
		for i := 0; i < fnCount; i++ {
			fns[i] = func(context.Context) error {
				atomic.AddInt32(&expectedResult, 1)
				return nil
			}
		}
		fastExit := rand.Intn(2) == 1
		err := gu.RunConcurrently(context.Background(), fns...)(fastExit)
		if err != nil {
			t.Errorf("unexpected error: want nil, got %v", err)
		}
		if result := atomic.LoadInt32(&expectedResult); int(result) != fnCount {
			t.Errorf("unexpected result: want %v, got %v", fnCount, result)
		}
	})
}
