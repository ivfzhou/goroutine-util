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
	"sync"
	"testing"
	"time"

	gu "gitee.com/ivfzhou/goroutine-util"
)

func TestRunConcurrentlyEmpty(t *testing.T) {
	// 没有 fn 可运行时返回空 wait。
	if wait := gu.RunConcurrently(context.Background()); wait != nil {
		t.Error("unexpected wait: want nil, got non-nil")
	}
}

func TestRunConcurrentlyCanceledCtx(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	wait := gu.RunConcurrently(ctx, func(context.Context) error { return nil })
	if wait == nil {
		t.Fatal("unexpected wait: want non-nil, got nil")
	}
	if err := wait(false); !errors.Is(err, context.Canceled) {
		t.Errorf("unexpected error: want %v, got %v", context.Canceled, err)
	}
}

func TestRunSequentiallyEmpty(t *testing.T) {
	if err := gu.RunSequentially(context.Background()); err != nil {
		t.Errorf("unexpected error: want nil, got %v", err)
	}
	if err := gu.RunSequentially(context.TODO()); err != nil {
		t.Errorf("unexpected error: want nil, got %v", err)
	}
	if err := gu.RunSequentially(context.Background(), nil, nil); err != nil {
		t.Errorf("unexpected error: want nil, got %v", err)
	}
}

func TestRunSequentiallyCanceledCtx(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := gu.RunSequentially(ctx, func(context.Context) error { return nil }); !errors.Is(err, context.Canceled) {
		t.Errorf("unexpected error: want %v, got %v", context.Canceled, err)
	}
}

func TestRunDataEmptyJobs(t *testing.T) {
	// 没有任务，即使 fn 为空也不触发恐慌。
	if err := gu.RunData[int](context.Background(), nil, false); err != nil {
		t.Errorf("unexpected error: want nil, got %v", err)
	}
}

func TestRunDataNilFn(t *testing.T) {
	defer func() {
		if p := recover(); p == nil {
			t.Error("unexpected result: want panic, got nil")
		}
	}()
	_ = gu.RunData(context.Background(), nil, false, 1)
}

func TestRunDataCanceledCtx(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := gu.RunData(ctx, func(context.Context, int) error { return nil }, false, 1)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("unexpected error: want %v, got %v", context.Canceled, err)
	}
}

func TestNewRunnerNilFn(t *testing.T) {
	defer func() {
		if p := recover(); p == nil {
			t.Error("unexpected result: want panic, got nil")
		}
	}()
	_, _ = gu.NewRunner[int](context.Background(), 0, nil)
}

func TestNewRunnerCanceledCtx(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	add, wait := gu.NewRunner(ctx, 0, func(context.Context, int) error { return nil })
	if err := add(1, false); !errors.Is(err, context.Canceled) {
		t.Errorf("unexpected add error: want %v, got %v", context.Canceled, err)
	}
	if err := wait(false); !errors.Is(err, context.Canceled) {
		t.Errorf("unexpected wait error: want %v, got %v", context.Canceled, err)
	}
}

func TestListenChanEmpty(t *testing.T) {
	ch := gu.ListenChan[int]()
	v, ok := <-ch
	if ok {
		t.Errorf("unexpected result: want false, got %v, value %v", ok, v)
	}
}

func TestListenChanNilOnly(t *testing.T) {
	ch := gu.ListenChan[int](nil, nil)
	v, ok := <-ch
	if ok {
		t.Errorf("unexpected result: want false, got %v, value %v", ok, v)
	}
}

func TestRunPeriodicallyNegative(t *testing.T) {
	defer func() {
		if p := recover(); p == nil {
			t.Error("unexpected result: want panic, got nil")
		}
	}()
	_ = gu.RunPeriodically(-time.Second)
}

func TestRunPeriodicallyZero(t *testing.T) {
	run := gu.RunPeriodically(0)
	called := false
	run(func() { called = true })
	if !called {
		t.Error("unexpected result: want true, got false")
	}
}

func TestRunPipelineEmpty(t *testing.T) {
	// 没有 steps。
	successCh, errCh := gu.RunPipeline(context.Background(), []int{1, 2}, false)
	if _, ok := <-successCh; ok {
		t.Errorf("unexpected successCh: want closed, got %v", ok)
	}
	if _, ok := <-errCh; ok {
		t.Errorf("unexpected errCh: want closed, got %v", ok)
	}

	// 没有 jobs。
	successCh, errCh = gu.RunPipeline(context.Background(), nil, false, func(context.Context, int) error { return nil })
	if _, ok := <-successCh; ok {
		t.Errorf("unexpected successCh: want closed, got %v", ok)
	}
	if _, ok := <-errCh; ok {
		t.Errorf("unexpected errCh: want closed, got %v", ok)
	}
}

func TestRunPipelineCanceledCtx(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	successCh, errCh := gu.RunPipeline(ctx, []int{1, 2}, false, func(context.Context, int) error { return nil })
	if _, ok := <-successCh; ok {
		t.Errorf("unexpected successCh: want closed, got %v", ok)
	}
	if _, ok := <-errCh; ok {
		t.Errorf("unexpected errCh: want closed, got %v", ok)
	}
}

func TestNewPipelineRunnerEmptySteps(t *testing.T) {
	push, successCh, endPush := gu.NewPipelineRunner[int](context.Background())
	if push(1) {
		t.Error("unexpected push: want false, got true")
	}
	if _, ok := <-successCh; ok {
		t.Errorf("unexpected successCh: want closed, got %v", ok)
	}
	endPush() // 不应恐慌。
}

func TestNewPipelineRunnerCanceledCtx(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	push, successCh, _ := gu.NewPipelineRunner(ctx, func(context.Context, int) bool { return true })
	if push(1) {
		t.Error("unexpected push: want false, got true")
	}
	if _, ok := <-successCh; ok {
		t.Errorf("unexpected successCh: want closed, got %v", ok)
	}
}

func TestQueueConcurrentGetAndClose(t *testing.T) {
	// 并发调用 GetFromChan 与 Close，不应发生数据竞争。
	for range 200 {
		q := &gu.Queue[int]{}
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			q.Push(1)
			q.Push(2)
			q.Close()
		}()
		go func() {
			defer wg.Done()
			for range q.GetFromChan() {
			}
		}()
		wg.Wait()
	}
}

func TestQueuePushAfterCloseConcurrent(t *testing.T) {
	// Push 与 Close 并发时，Push 要么成功要么返回 false，但不能崩溃。
	for range 200 {
		q := &gu.Queue[int]{}
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			for range 100 {
				q.Push(1)
			}
			q.Close()
		}()
		go func() {
			defer wg.Done()
			for range q.GetFromChan() {
			}
		}()
		wg.Wait()
	}
}
