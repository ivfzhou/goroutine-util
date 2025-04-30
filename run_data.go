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

// RunData 将 jobs 传给 fn，并发运行 fn。若发生错误，就立刻终止运行。
//
// ctx：上下文。如果上下文终止了，则终止运行。
//
// fn：要运行处理 jobs 的函数。
//
// fastExit：发生错误就立刻返回，不等待所有协程全部退出。
//
// jobs：要处理的任务。
//
// 返回错误与 fn 返回的错误一致。
//
// 注意：fn 为空将触发恐慌。
func RunData[T any](ctx context.Context, fn func(context.Context, T) error, fastExit bool, jobs ...T) error {
	// 任务为空，直接返回。
	if len(jobs) <= 0 {
		return nil
	}

	// fn 不能为空。
	if fn == nil {
		panic("fn cannot be nil")
	}

	nilCtx := false
	if ctx == nil {
		nilCtx = true
		ctx = context.Background()
	}

	// 上下文终止了就直接返回。
	if !nilCtx {
		select {
		case <-ctx.Done():
			err := ctx.Err()
			return err
		default:
		}
	}

	innerCtx, cancel := context.WithCancel(ctx) // 创建上下文，以便发生错误时，通知终止运行。
	var err AtomicError                         // 持有错误信息。
	wg := &sync.WaitGroup{}
	fnWrapper := func(job T) {
		defer wg.Done()
		select {
		case <-innerCtx.Done():
			return
		default:
		}
		var fnErr error
		defer func() {
			if p := recover(); p != nil {
				fnErr = wrapperPanic(p)
			}
			if fnErr != nil {
				err.Set(fnErr)
				cancel()
			}
		}()
		if nilCtx {
			fnErr = fn(nil, job)
		} else {
			fnErr = fn(ctx, job)
		}
	}
	for _, v := range jobs {
		wg.Add(1)
		go fnWrapper(v)
	}

	// 运行完毕后释放资源。
	go func() {
		wg.Wait()
		cancel()
	}()

	select {
	case <-innerCtx.Done(): // 上下文终止了，可能时发生了错误。
		e := err.Get()
		if e == nil && !nilCtx { // 也可能是上层上下文终止了。
			err.Set(ctx.Err())
		}
		if fastExit {
			return err.Get()
		}
	}

	wg.Wait()
	return err.Get()
}
