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
	"os"
	"sync"
	"sync/atomic"
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
		return func(T) bool { return false }, ch, func() {}
	}

	userCtx := ctx
	if ctx == nil {
		ctx = context.Background()
	} else {
		select {
		case <-ctx.Done():
			ch := make(chan T)
			close(ch)
			return func(T) bool { return false }, ch, func() {}
		default:
		}
	}

	pushLock := sync.RWMutex{}           // 避免 push 和 endPush 同时调用，导致数据冲突。
	endPushFlag := int32(0)              // 标识已经调用了 endPush。
	pushChan := make(chan T, 1)          // 推送数据通道。
	closePushChanWg := &sync.WaitGroup{} // 等待所有数据都推送进队列。

	endPush = func() {
		pushLock.Lock()
		defer pushLock.Unlock()
		if atomic.CompareAndSwapInt32(&endPushFlag, 0, 1) {
			go func() {
				closePushChanWg.Wait()
				close(pushChan)
			}()
		}
	}

	push = func(t T) bool {
		pushLock.RLock()
		defer pushLock.RUnlock()
		if atomic.LoadInt32(&endPushFlag) > 0 {
			return false
		}

		select {
		case pushChan <- t:
			return true
		default:
		}

		closePushChanWg.Add(1)
		go func() {
			defer closePushChanWg.Done()
			pushChan <- t
		}()

		return true
	}

	stepWrapperFn := func(ctx context.Context, job T, step func(context.Context, T) bool) (b bool) {
		defer func() {
			if p := recover(); p != nil {
				_, _ = fmt.Fprintf(os.Stderr, "%v", wrapperPanic(p))
				b = false
			}
		}()
		b = step(ctx, job)
		return
	}

	startStep := func(step func(context.Context, T) bool, receiveChan <-chan T) chan T { // 开启一个 step 协程。
		sendChan := make(chan T, 1)

		go func() {
			wg := &sync.WaitGroup{}
			defer func() {
				wg.Wait()
				close(sendChan)
			}()

			for v := range receiveChan {
				wg.Add(1)
				go func(job T) {
					defer wg.Done()
					if !stepWrapperFn(userCtx, job, step) {
						return
					}
					select {
					case <-ctx.Done():
					case sendChan <- job:
					}
				}(v)
			}
		}()

		return sendChan
	}

	// 开启所有 step 协程。
	var nextChan = pushChan
	for i := range steps {
		nextChan = startStep(steps[i], nextChan)
	}

	return push, nextChan, endPush
}
