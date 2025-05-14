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
	"sync"
)

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
	userCtx := ctx
	if ctx == nil {
		nilCtx = true
		ctx = context.Background()
	} else { // 上下文已终止，返回函数。
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

		go func() {
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
				if !nilCtx && !err.HasSet() {
					select {
					case <-ctx.Done():
						err.Set(ctx.Err()) // 顶层上下文终止。
					default:
					}
				}
			default:
				fnErr = f(userCtx)
			}
		}()
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
				if !nilCtx && !err.HasSet() {
					select {
					case <-ctx.Done():
						err.Set(ctx.Err()) // 可能是顶层上下文终止了。
					default:
					}
				}
				err.Set(nil) // 记上设置标记。
				return err.Get()
			}
		}

		wg.Wait()
		return err.Get()
	}
}
