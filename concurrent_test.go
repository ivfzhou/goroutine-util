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

type ctxCancelWithError struct {
	ctx context.Context
	err gu.AtomicError
}

func TestRunConcurrently(t *testing.T) {
	t.Run("正常运行", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			ctx := context.Background()

			const count = 1000
			expectedResult := int32(0)
			fns := make([]func(ctx context.Context) error, count)
			for i := 0; i < count; i++ {
				fns[i] = func(ctx context.Context) error {
					time.Sleep(time.Millisecond * 100)
					atomic.AddInt32(&expectedResult, 1)
					return nil
				}
			}

			fastExit := rand.Intn(2) == 0
			err := gu.RunConcurrently(ctx, fns...)(fastExit)
			if err != nil {
				t.Errorf("unexpected error: want nil, got %v", err)
			}
			if result := atomic.LoadInt32(&expectedResult); result != count {
				t.Errorf("unexpected result: want %v, got %v", count, result)
			}
		}
	})

	t.Run("发生错误", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			ctx := context.Background()
			expectedErr := errors.New("expected error")

			const count = 1000
			expectedResult := int32(0)
			occurErrorIndex := rand.Intn(count)
			fns := make([]func(ctx context.Context) error, count)
			for i := 0; i < count; i++ {
				index := i
				fns[i] = func(ctx context.Context) error {
					if index == occurErrorIndex {
						return expectedErr
					}
					time.Sleep(time.Millisecond * 100)
					atomic.AddInt32(&expectedResult, 1)
					return nil
				}
			}

			fastExit := rand.Intn(2) == 0
			err := gu.RunConcurrently(ctx, fns...)(fastExit)
			if err == nil || !errors.Is(err, expectedErr) {
				t.Errorf("unexpected error: want %v, got %v", expectedErr, err)
			}
			if result := atomic.LoadInt32(&expectedResult); result >= count {
				t.Errorf("unexpected result: want <=%v, got %v", count, result)
			}
		}
	})

	t.Run("发生恐慌", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			ctx := context.Background()
			expectedErr := errors.New("expected error")

			const count = 1000
			expectedResult := int32(0)
			panicIndex := rand.Intn(count)
			fns := make([]func(ctx context.Context) error, count)
			for i := 0; i < count; i++ {
				index := i
				fns[i] = func(ctx context.Context) error {
					if index == panicIndex {
						panic(expectedErr)
					}
					time.Sleep(time.Millisecond * 100)
					atomic.AddInt32(&expectedResult, 1)
					return nil
				}
			}

			fastExit := rand.Intn(2) == 0
			err := gu.RunConcurrently(ctx, fns...)(fastExit)
			if err == nil || !errors.Is(err, expectedErr) {
				t.Errorf("unexpected error: want %v, got %v", expectedErr, err)
			}
			if result := atomic.LoadInt32(&expectedResult); result >= count {
				t.Errorf("unexpected result: want <=%v, got %v", count, result)
			}
		}
	})

	t.Run("上下文终止", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			ctx, cancel := newCtxCancelWithError()
			expectedErr := errors.New("expected error")

			const count = 1000
			expectedResult := int32(0)
			cancelIndex := rand.Intn(count / 2)
			fns := make([]func(ctx context.Context) error, count)
			for i := 0; i < count; i++ {
				index := i
				fns[i] = func(ctx context.Context) error {
					if index == cancelIndex {
						cancel(expectedErr)
					}
					time.Sleep(time.Millisecond * 100)
					atomic.AddInt32(&expectedResult, 1)
					return nil
				}
			}

			fastExit := rand.Intn(2) == 0
			err := gu.RunConcurrently(ctx, fns...)(fastExit)
			if err == nil || !errors.Is(err, expectedErr) {
				t.Errorf("unexpected error: want %v, got %v", expectedErr, err)
			}
			if result := atomic.LoadInt32(&expectedResult); result >= count {
				t.Errorf("unexpected result: want <=%v, got %v", count, result)
			}
		}
	})
}

func newCtxCancelWithError() (context.Context, context.CancelCauseFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	c := &ctxCancelWithError{ctx: ctx}
	return c, func(cause error) {
		c.err.Set(cause)
		cancel()
	}
}

func (c *ctxCancelWithError) Deadline() (deadline time.Time, ok bool) {
	return c.ctx.Deadline()
}

func (c *ctxCancelWithError) Done() <-chan struct{} {
	return c.ctx.Done()
}

func (c *ctxCancelWithError) Err() error {
	return c.err.Get()
}

func (c *ctxCancelWithError) Value(key any) any {
	return c.ctx.Value(key)
}
