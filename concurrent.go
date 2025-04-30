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
	"context"
	"fmt"
	"sync"
	"sync/atomic"
)

// AtomicError 原子性读取和设置错误信息。
type AtomicError struct {
	err atomic.Value
}

// RunConcurrently 并发运行 fn，一旦有错误发生，终止运行。
//
// ctx：上下文，终止时，将导致 fn 终止运行，并在调用 wait 时返回 ctx.Err()。
//
// fn：要并发运行的函数。触发的恐慌将被恢复，并在调用 wait 时，以错误形式返回。
//
// wait：等待所有 fn 运行完毕。阻塞调用者协程，若 fastExit 为真，则一旦 fn 中发生了错误，立刻返回该错误。
//
// 注意：fn 是空将触发恐慌。
func RunConcurrently(ctx context.Context, fn ...func(context.Context) error) (wait func(fastExit bool) error) {
	// 没有 fn 可运行，返回函数。
	if len(fn) <= 0 {
		return nil
	}

	nilCtx := false
	if ctx == nil {
		nilCtx = true
		ctx = context.Background()
	}

	// 上下文已终止，返回函数。
	if !nilCtx {
		select {
		case <-ctx.Done():
			err := ctx.Err()
			return func(bool) error { return err }
		default:
		}
	}

	var err AtomicError // 持有错误信息。
	innerCtx, cancel := context.WithCancel(ctx)

	// 并发运行所有 fn。
	wg := &sync.WaitGroup{}
	wg.Add(len(fn))
	for _, f := range fn {
		if f == nil {
			panic("fn cannot be nil")
		}
		go func(f func(context.Context) error) {
			var fnErr error
			defer func() {
				if p := recover(); p != nil {
					fnErr = wrapperPanic(p)
				}
				if fnErr != nil {
					err.Set(fnErr) // 先设置 err。
					cancel()       // 再通知上下文终止。
				}
				wg.Done()
			}()
			select {
			case <-innerCtx.Done(): // 发生了错误，任务终止了，或者是顶层上下文终止了。
				err.Set(ctx.Err()) // 顶层上下文终止。
			default:
				if nilCtx {
					fnErr = f(nil)
				} else {
					fnErr = f(ctx)
				}
			}
		}(f)
	}

	// 所有任务运行完毕，释放上下文资源。
	go func() {
		wg.Wait()
		cancel()
	}()

	return func(fastExit bool) error {
		if fastExit {
			select {
			case <-innerCtx.Done(): // 上下文终止了就立刻返回。
				// 可能是顶层上下文终止了。
				e := err.Get()
				if e == nil {
					return ctx.Err()
				}
				return e
			}
		}

		wg.Wait()
		return err.Get()
	}
}

// Set 设置错误信息，除非 err 是空。返回真表示设置成功。
func (e *AtomicError) Set(err error) bool {
	if err == nil {
		return false
	}
	return e.err.CompareAndSwap(nil, err)
}

// Get 获取内部错误信息。
func (e *AtomicError) Get() error {
	err, _ := e.err.Load().(error)
	return err
}

// 将恐慌信息以 error 形式返回。并继承恐慌中的 error。
func wrapperPanic(p any) error {
	pe, ok := p.(error)
	if ok {
		return fmt.Errorf("panic: %w [recovered]\n%s\n", pe, getStackCallers())
	}
	return fmt.Errorf("panic: %v [recovered]\n%s\n", p, getStackCallers())
}
