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
	t.Run("正常运行", testNewRunnerNormal)

	t.Run("发生错误", testNewRunnerOccurError)

	t.Run("发生恐慌", testNewRunnerOccurPanic)

	t.Run("上下文终止", testNewRunnerCtxCancel)

	t.Run("大量add/wait", testNewRunnerManyAddWait)

	t.Run("wait后add", testNewRunnerAddAfterWait)

	t.Run("长时间运行", testNewRunnerLongTime)
}

func testNewRunnerNormal(t *testing.T) {
	for i := 0; i < 100; i++ {
		max := 0
		if rand.Intn(2) == 1 {
			max = 300
		}
		type data struct {
			x int32
		}
		ctx := context.Background()
		result := int32(0)
		add, wait := gu.NewRunner(ctx, max, func(ctx context.Context, t *data) error {
			time.Sleep(time.Millisecond * 100)
			atomic.AddInt32(&result, t.x)
			return nil
		})

		var err error
		expectedResult := int32(0)
		const count = 1000
		for i := int32(0); i < count; i++ {
			n := rand.Int31n(100)
			addBlock := rand.Intn(2) == 1
			err = add(&data{n}, addBlock)
			expectedResult += n
			if err != nil {
				break
			}
		}
		if err != nil {
			t.Errorf("unexpected error: want nil, got %v", err)
		}

		waitFast := rand.Intn(2) == 1
		if err = wait(waitFast); err != nil {
			t.Errorf("unexpected error: want nil, got %v", err)
		}
		if result != expectedResult {
			t.Errorf("unexpected result: want %v, got %v", expectedResult, result)
		}
	}
}

func testNewRunnerOccurError(t *testing.T) {
	for i := 0; i < 100; i++ {
		max := 0
		if rand.Intn(2) == 1 {
			max = 300
		}
		type data struct {
			x, y int32
		}
		ctx := context.Background()
		expectedErr := errors.New("expected error")
		const count = 1000
		occurErrorIndex := rand.Int31n(count / 2)
		result := int32(0)
		add, wait := gu.NewRunner(ctx, max, func(ctx context.Context, t *data) error {
			time.Sleep(time.Millisecond * 100)
			if t.x == occurErrorIndex {
				return expectedErr
			}
			atomic.AddInt32(&result, t.y)
			return nil
		})

		var err error
		expectedResult := int32(0)
		for i := int32(0); i < count; i++ {
			n := rand.Int31n(100)
			addBlock := rand.Intn(2) == 1
			err = add(&data{i, n}, addBlock)
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
}

func testNewRunnerOccurPanic(t *testing.T) {
	for i := 0; i < 100; i++ {
		max := 0
		if rand.Intn(2) == 1 {
			max = 300
		}
		type data struct {
			x, y int32
		}
		ctx := context.Background()
		expectedErr := errors.New("expected error")
		const count = 1000
		ocuurPanicIndex := rand.Int31n(count / 2)
		result := int32(0)
		add, wait := gu.NewRunner(ctx, max, func(ctx context.Context, t *data) error {
			time.Sleep(time.Millisecond * 100)
			if t.x == ocuurPanicIndex {
				panic(expectedErr)
			}
			atomic.AddInt32(&result, t.y)
			return nil
		})

		var err error
		expectedCount := int32(0)
		for i := int32(0); i < count; i++ {
			n := rand.Int31n(100)
			addBlock := rand.Intn(2) == 1
			err = add(&data{i, n}, addBlock)
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
}

func testNewRunnerLongTime(t *testing.T) {
	for i := 0; i < 10; i++ {
		max := 0
		if rand.Intn(2) == 1 {
			max = 300
		}
		type data struct {
			x, y int32
		}
		ctx := context.Background()
		result := int32(0)
		const count = 1000
		sleepIndex := rand.Int31n(count / 2)
		add, wait := gu.NewRunner(ctx, max, func(ctx context.Context, t *data) error {
			if t.y == sleepIndex {
				time.Sleep(time.Second * 10)
			} else {
				time.Sleep(time.Millisecond * 100)
			}
			atomic.AddInt32(&result, t.x)
			return nil
		})

		var err error
		expectedResult := int32(0)
		for i := int32(0); i < count; i++ {
			n := rand.Int31n(100)
			err = add(&data{n, i}, rand.Intn(2) == 1)
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
}

func testNewRunnerManyAddWait(t *testing.T) {
	for i := 0; i < 100; i++ {
		max := 0
		if rand.Intn(2) == 1 {
			max = 300
		}
		type data struct {
			x int32
		}
		ctx := context.Background()
		result := int32(0)
		const count = 1000
		add, wait := gu.NewRunner(ctx, max, func(ctx context.Context, t *data) error {
			time.Sleep(time.Millisecond * 100)
			atomic.AddInt32(&result, t.x)
			return nil
		})

		wg := sync.WaitGroup{}
		expectedResult := int32(0)
		for i := int32(0); i < count; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				n := rand.Int31n(100)
				_ = add(&data{n}, rand.Intn(2) == 1)
				atomic.AddInt32(&expectedResult, n)
			}()
		}
		wg.Wait()

		waitErrorMap := sync.Map{}
		for i := 0; i < count; i++ {
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
}

func testNewRunnerCtxCancel(t *testing.T) {
	for i := 0; i < 100; i++ {
		max := 0
		if rand.Intn(2) == 1 {
			max = 300
		}
		type data struct {
			x, y int32
		}
		expectedErr := errors.New("expected error")
		ctx, cancel := newCtxCancelWithError()
		result := int32(0)
		const count = 1000
		cancelIndex := rand.Int31n(count / 2)
		add, wait := gu.NewRunner(ctx, max, func(ctx context.Context, t *data) error {
			time.Sleep(time.Millisecond * 100)
			if t.x == cancelIndex {
				cancel(expectedErr)
			}
			atomic.AddInt32(&result, t.y)
			return nil
		})

		var err error
		expectedResult := int32(0)
		for i := int32(0); i < count; i++ {
			n := rand.Int31n(100)
			addBlock := rand.Intn(2) == 1
			err = add(&data{i, n}, addBlock)
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
}

func testNewRunnerAddAfterWait(t *testing.T) {
	for i := 0; i < 100; i++ {
		max := 0
		if rand.Intn(2) == 1 {
			max = 300
		}
		type data struct{}
		ctx := context.Background()
		add, wait := gu.NewRunner(ctx, max, func(ctx context.Context, t *data) error {
			time.Sleep(time.Millisecond * 100)
			return nil
		})

		const count = 1000
		callWaitIndex := rand.Int31n(count / 2)
		wg := sync.WaitGroup{}
		for i := int32(0); i < count; i++ {
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
				_ = add(&data{}, rand.Int31n(2) == 1)
			}()
		}
		wg.Wait()
	}
}
