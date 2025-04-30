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

// NewPipelineRunner 创建流水线工作模型。
//
// ctx：上下文。当上下文终止时，将终止流水线运行，push 将返回假。
//
// steps：任务 T 依次在 steps 中运行。step 返回真表示将 T 传递给下一个 step 处理，否则不传递。
//
// push：向 step 中传递一个 T 处理，返回真表示添加成功。
//
// successCh：成功运行完所有 step 的 T 将从该通道发出。
//
// endPush：表示不再有 T 需要处理，push 将返回假。
func NewPipelineRunner[T any](ctx context.Context, steps ...func(context.Context, T) bool) (
	push func(T) bool, successCh <-chan T, endPush func()) {

	// 没有 step，则直接返回。
	if len(steps) <= 0 {
		ch := make(chan T)
		close(ch)
		return func(t T) bool { return false }, ch, func() {}
	}

	stepWrapper := func(ctx context.Context, f func(context.Context, T) bool, dataChan <-chan T) <-chan T {
		ch := make(chan T, cap(dataChan))

		go func() {
			wg := &sync.WaitGroup{}
			defer func() {
				wg.Wait()
				close(ch)
			}()

			if ctx == nil {
				for t := range dataChan {
					wg.Add(1)
					go func(t T) {
						defer wg.Done()
						if f(ctx, t) {
							ch <- t
						}
					}(t)
				}
				return
			}

			for {
				select {
				case <-ctx.Done():
					return
				case t, ok := <-dataChan:
					if !ok {
						return
					}
					wg.Add(1)
					go func(t T) {
						defer wg.Done()
						select {
						case <-ctx.Done():
							return
						default:
						}
						if f(ctx, t) {
							ch <- t
						}
					}(t)
				}
			}
		}()

		return ch
	}

	jobQueue := &Queue[T]{}
	nextCh := jobQueue.GetFromChan()
	for i := range steps {
		nextCh = stepWrapper(ctx, steps[i], nextCh)
	}

	return jobQueue.Push, nextCh, jobQueue.Close
}
