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

package goroutine_util

import (
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

func TestWrapError(t *testing.T) {
	// 空错误返回空。
	if err := WrapError(nil); err != nil {
		t.Errorf("unexpected error: want nil, got %v", err)
	}

	// 非空错误被包裹，且可被 Unwrap 解开。
	sentinel := errors.New("sentinel")
	wrapped := WrapError(sentinel)
	if wrapped == nil {
		t.Fatal("unexpected error: want non-nil, got nil")
	}
	if !errors.Is(wrapped, sentinel) {
		t.Errorf("unexpected error: want %v, got %v", sentinel, wrapped)
	}

	// 包裹结果应是 *Error 类型。
	if _, ok := wrapped.(*Error); !ok {
		t.Errorf("unexpected type: want *Error, got %T", wrapped)
	}
}

func TestErrorMethods(t *testing.T) {
	sentinel := errors.New("boom")
	e := &Error{err: sentinel}
	if got := e.Error(); got != "boom" {
		t.Errorf("unexpected Error(): want boom, got %v", got)
	}
	if got := e.String(); got != "boom" {
		t.Errorf("unexpected String(): want boom, got %v", got)
	}
	if got := e.Unwrap(); got != sentinel {
		t.Errorf("unexpected Unwrap(): want %v, got %v", sentinel, got)
	}

	// 内部错误为空时返回空字符串与空错误。
	empty := &Error{}
	if got := empty.Error(); got != "" {
		t.Errorf("unexpected Error(): want empty, got %v", got)
	}
	if got := empty.String(); got != "" {
		t.Errorf("unexpected String(): want empty, got %v", got)
	}
	if got := empty.Unwrap(); got != nil {
		t.Errorf("unexpected Unwrap(): want nil, got %v", got)
	}
}

func TestAtomicError(t *testing.T) {
	var e AtomicError

	// 初始未设置。
	if e.HasSet() {
		t.Error("unexpected HasSet(): want false, got true")
	}
	if e.Get() != nil {
		t.Errorf("unexpected Get(): want nil, got %v", e.Get())
	}

	// 第一次设置成功，后续设置失败。
	sentinel := errors.New("sentinel")
	if !e.Set(sentinel) {
		t.Error("unexpected Set(): want true, got false")
	}
	if !e.HasSet() {
		t.Error("unexpected HasSet(): want true, got false")
	}
	if !errors.Is(e.Get(), sentinel) {
		t.Errorf("unexpected Get(): want %v, got %v", sentinel, e.Get())
	}
	if e.Set(errors.New("second")) {
		t.Error("unexpected Set(): want false, got true")
	}
	if !errors.Is(e.Get(), sentinel) {
		t.Errorf("unexpected Get(): want first error %v, got %v", sentinel, e.Get())
	}

	// Set(nil) 只打标记，Get 仍返回空。
	var e2 AtomicError
	if !e2.Set(nil) {
		t.Error("unexpected Set(nil): want true, got false")
	}
	if !e2.HasSet() {
		t.Error("unexpected HasSet(): want true, got false")
	}
	if e2.Get() != nil {
		t.Errorf("unexpected Get(): want nil, got %v", e2.Get())
	}
}

func TestAtomicErrorConcurrentSet(t *testing.T) {
	for range 100 {
		var e AtomicError
		sentinel := errors.New("sentinel")
		const goroutines = 100
		var winners int32
		var wg sync.WaitGroup
		for range goroutines {
			wg.Go(func() {
				if e.Set(sentinel) {
					atomic.AddInt32(&winners, 1)
				}
			})
		}
		wg.Wait()
		if winners != 1 {
			t.Errorf("unexpected winners: want 1, got %v", winners)
		}
		if !errors.Is(e.Get(), sentinel) {
			t.Errorf("unexpected Get(): want %v, got %v", sentinel, e.Get())
		}
	}
}

func TestWrapperPanic(t *testing.T) {
	// 恐慌值为 error 时，能通过 errors.Is 继承。
	sentinel := errors.New("sentinel")
	err := wrapperPanic(sentinel)
	if !errors.Is(err, sentinel) {
		t.Errorf("unexpected error: want %v, got %v", sentinel, err)
	}
	if !strings.Contains(err.Error(), "[recovered]") {
		t.Errorf("unexpected error: want contain [recovered], got %v", err)
	}
	if !strings.Contains(err.Error(), "panic:") {
		t.Errorf("unexpected error: want contain panic:, got %v", err)
	}

	// 恐慌值非 error 时，其字符串形式被包含在错误中。
	err = wrapperPanic("boom")
	if !strings.Contains(err.Error(), "boom") {
		t.Errorf("unexpected error: want contain boom, got %v", err)
	}
	if !strings.Contains(err.Error(), "[recovered]") {
		t.Errorf("unexpected error: want contain [recovered], got %v", err)
	}
}

func TestGetStackCallers(t *testing.T) {
	s := getStackCallers()
	if s == "" {
		t.Fatal("unexpected stack: want non-empty, got empty")
	}
	if !strings.Contains(s, ".go:") {
		t.Errorf("unexpected stack: want contain file:line, got %v", s)
	}
}
