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
	"sync"
	"sync/atomic"
	"testing"
	"time"

	gu "gitee.com/ivfzhou/goroutine-util"
)

func ExampleNewRunner() {
	type product struct{}
	ctx := context.Background()
	op := func(ctx context.Context, data *product) error {
		// 要处理的任务逻辑。
		return nil
	}

	add, wait := gu.NewRunner[*product](ctx, 12, op)

	// 将任务要处理的数据传递给任务处理逻辑 op。
	var projects []*product
	for _, v := range projects {
		if err := add(v, true); err != nil {
			// 发生错误可能会提前预知。
		}
	}

	// 等待所有数据处理完毕。
	if err := wait(true); err != nil {
		// 处理 op 返回的错误。
	}
}

func TestNewRunner(t *testing.T) {
	t.Run("正常运行", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			maxRoutines := 0
			if rand.Intn(2) == 1 {
				maxRoutines = 300
			}
			type job struct {
				x int32
			}
			result := int32(0)
			add, wait := gu.NewRunner(context.Background(), maxRoutines, func(ctx context.Context, t *job) error {
				time.Sleep(time.Millisecond * 100)
				atomic.AddInt32(&result, t.x)
				return nil
			})
			var err error
			expectedResult := int32(0)
			const jobCount = 800
			for i := int32(0); i < jobCount; i++ {
				n := rand.Int31n(100)
				addBlock := rand.Intn(2) == 1
				err = add(&job{n}, addBlock)
				expectedResult += n
				if err != nil {
					break
				}
			}
			if err != nil {
				t.Errorf("unexpected error: want nil, got %v", err)
			}
			if err = wait(rand.Intn(2) == 1); err != nil {
				t.Errorf("unexpected error: want nil, got %v", err)
			}
			if result != expectedResult {
				t.Errorf("unexpected result: want %v, got %v", expectedResult, result)
			}
		}
	})

	t.Run("发生错误", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			maxRoutines := 0
			if rand.Intn(2) == 1 {
				maxRoutines = 300
			}
			type job struct {
				x, y int32
			}
			expectedErr := errors.New("expected error")
			const jobCount = 800
			occurErrorIndex := rand.Int31n(jobCount / 2)
			result := int32(0)
			add, wait := gu.NewRunner(context.Background(), maxRoutines, func(ctx context.Context, t *job) error {
				time.Sleep(time.Millisecond * 100)
				if t.x == occurErrorIndex {
					return expectedErr
				}
				atomic.AddInt32(&result, t.y)
				return nil
			})
			var err error
			expectedResult := int32(0)
			for i := int32(0); i < jobCount; i++ {
				n := rand.Int31n(100)
				addBlock := rand.Intn(2) == 1
				err = add(&job{i, n}, addBlock)
				expectedResult += n
				if err != nil {
					break
				}
			}
			if err != nil && !errors.Is(err, expectedErr) {
				t.Errorf("unexpected error: want %v, got %v", expectedErr, err)
			}
			waitFast := rand.Intn(2) == 1
			if err = wait(waitFast); !errors.Is(err, expectedErr) {
				t.Errorf("unexpected error: want %v, got %v", expectedErr, err)
			}
			if result > expectedResult {
				t.Errorf("unexpected result: got %v, want %v", result, expectedResult)
			}
		}
	})

	t.Run("发生恐慌", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			maxRoutines := 0
			if rand.Intn(2) == 1 {
				maxRoutines = 300
			}
			type job struct {
				x, y int32
			}
			expectedErr := errors.New("expected error")
			const jobCount = 800
			occurPanicIndex := rand.Int31n(jobCount / 2)
			result := int32(0)
			add, wait := gu.NewRunner(context.Background(), maxRoutines, func(ctx context.Context, t *job) error {
				time.Sleep(time.Millisecond * 100)
				if t.x == occurPanicIndex {
					panic(expectedErr)
				}
				atomic.AddInt32(&result, t.y)
				return nil
			})
			var err error
			expectedCount := int32(0)
			for i := int32(0); i < jobCount; i++ {
				n := rand.Int31n(100)
				addBlock := rand.Intn(2) == 1
				err = add(&job{i, n}, addBlock)
				expectedCount += n
				if err != nil {
					break
				}
			}
			if err != nil && !errors.Is(err, expectedErr) {
				t.Errorf("unexpected error: want %v, got %v", expectedErr, err)
			}
			waitFast := rand.Intn(2) == 1
			if err = wait(waitFast); !errors.Is(err, expectedErr) {
				t.Errorf("unexpected error: want %v, got %v", expectedErr, err)
			}
			if result > expectedCount {
				t.Errorf("unexpected result: want %v, got %v", expectedCount, result)
			}
		}
	})

	t.Run("上下文终止", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			maxRoutines := 0
			if rand.Intn(2) == 1 {
				maxRoutines = 300
			}
			type job struct {
				x, y int32
			}
			expectedErr := errors.New("expected error")
			ctx, cancel := NewCtxCancelWithError()
			result := int32(0)
			const jobCount = 800
			cancelIndex := rand.Int31n(jobCount / 2)
			add, wait := gu.NewRunner(ctx, maxRoutines, func(ctx context.Context, t *job) error {
				time.Sleep(time.Millisecond * 100)
				if t.x == cancelIndex {
					cancel(expectedErr)
				}
				atomic.AddInt32(&result, t.y)
				return nil
			})
			var err error
			expectedResult := int32(0)
			for i := int32(0); i < jobCount; i++ {
				n := rand.Int31n(100)
				addBlock := rand.Intn(2) == 1
				err = add(&job{i, n}, addBlock)
				if err != nil {
					break
				}
				expectedResult += n
			}
			if err != nil && !errors.Is(err, expectedErr) {
				t.Errorf("unexpected error: want %v, got %v", expectedErr, err)
			}
			waitFast := rand.Intn(2) == 1
			if err = wait(waitFast); !errors.Is(err, expectedErr) {
				t.Errorf("unexpected error: want %v, got %v", expectedErr, err)
			}
			if result > expectedResult {
				t.Errorf("unexpected result: want %v, got %v", expectedResult, result)
			}
		}
	})

	t.Run("大量add、wait", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			maxRoutines := 0
			if rand.Intn(2) == 1 {
				maxRoutines = 300
			}
			type job struct {
				x int32
			}
			result := int32(0)
			const jobCount = 800
			add, wait := gu.NewRunner(context.Background(), maxRoutines, func(ctx context.Context, t *job) error {
				time.Sleep(time.Millisecond * 100)
				atomic.AddInt32(&result, t.x)
				return nil
			})
			wg := sync.WaitGroup{}
			expectedResult := int32(0)
			for i := int32(0); i < jobCount; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					n := rand.Int31n(100)
					_ = add(&job{n}, rand.Intn(2) == 1)
					atomic.AddInt32(&expectedResult, n)
				}()
			}
			wg.Wait()
			waitErrorMap := sync.Map{}
			for i := 0; i < jobCount; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					waitErrorMap.Store(i, wait(rand.Intn(2) == 1))
				}(i)
			}
			wg.Wait()
			err := wait(rand.Intn(2) == 1)
			if err != nil {
				t.Errorf("unexpected error: want nil, got %v", err)
			}
			if result != expectedResult {
				t.Errorf("unexpected result: want %v, got %v", expectedResult, result)
			}
			waitErrorMap.Range(func(_, v any) bool {
				if v != nil {
					t.Errorf("unexpected error: want nil, hot %v", err)
				}
				return true
			})
		}
	})

	t.Run("wait后add", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			maxRoutines := 0
			if rand.Intn(2) == 1 {
				maxRoutines = 300
			}
			type job struct{}
			add, wait := gu.NewRunner(context.Background(), maxRoutines, func(ctx context.Context, t *job) error {
				time.Sleep(time.Millisecond * 100)
				return nil
			})
			const jobCount = 1000
			callWaitIndex := rand.Int31n(jobCount / 2)
			wg := sync.WaitGroup{}
			for i := int32(0); i < jobCount; i++ {
				if i == callWaitIndex {
					go func() {
						_ = wait(rand.Int31n(2) == 1)
					}()
				}
				wg.Add(1)
				go func() {
					defer wg.Done()
					defer func() {
						p := recover()
						if p != nil && p != gu.ErrCallAddAfterWait {
							t.Errorf("unexpected recoverd value: want %v, got %v", gu.ErrCallAddAfterWait, p)
						}
					}()
					_ = add(&job{}, rand.Int31n(2) == 1)
				}()
			}
			wg.Wait()
		}
	})

	t.Run("长时间运行", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			maxRoutines := 0
			if rand.Intn(2) == 1 {
				maxRoutines = 300
			}
			type job struct {
				x, y int32
			}
			result := int32(0)
			const jobCount = 800
			sleepIndex := rand.Int31n(jobCount / 2)
			add, wait := gu.NewRunner(context.Background(), maxRoutines, func(ctx context.Context, t *job) error {
				if t.y == sleepIndex {
					time.Sleep(time.Second)
				}
				atomic.AddInt32(&result, t.x)
				return nil
			})
			var err error
			expectedResult := int32(0)
			for i := int32(0); i < jobCount; i++ {
				n := rand.Int31n(100)
				err = add(&job{n, i}, rand.Intn(2) == 1)
				expectedResult += n
				if err != nil {
					break
				}
			}
			if err != nil {
				t.Errorf("unexpected error: want nil, got %v", err)
				return
			}
			if err = wait(rand.Intn(2) == 1); err != nil {
				t.Errorf("unexpected error: want nil, got %v", err)
			}
			if result != expectedResult {
				t.Errorf("unexpected result: want %v, got %v", expectedResult, result)
			}
		}
	})
}
