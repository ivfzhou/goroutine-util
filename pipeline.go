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

var RunPipelineCacheJobs = 10

// RunPipeline 将每个 jobs 依次递给 steps 函数处理。一旦某个 step 发生错误或恐慌，就结束处理，返回错误。
// 一个 job 最多在一个 step 中运行一次，且一个 job 一定是依次序递给 steps，前一个 step 处理完毕才会给下一个 step 处理。
// 每个 step 并发运行 jobs。
//
// ctx：上下文，上下文终止时，将终止所有 steps 运行，并在 errCh 中返回 ctx.Err()。
//
// stopWhenErr：当 step 发生错误时，是否继续处理 job。当为假时，只会停止将 job 往下一个 step 传递，不会终止运行。
//
// steps：处理 jobs 的函数。返回错误意味着终止运行。
//
// successCh：经所有 steps 处理成功 job 将从该通道发出。
//
// errCh：运行中的错误从该通道发出。
//
// 注意：若 steps 中含有空元素将会恐慌。
func RunPipeline[T any](ctx context.Context, jobs []T, stopWhenErr bool, steps ...func(context.Context, T) error) (
	successCh <-chan T, errCh <-chan error) {

	if len(steps) <= 0 || len(jobs) <= 0 { // 检测 step 和 jobs 不为空。
		ech := make(chan error)
		sch := make(chan T)
		close(sch)
		close(ech)
		return sch, ech
	}

	// 开启上下文，以便发生错误时，通知终止协程。
	userCtx := ctx
	if ctx == nil {
		ctx = context.Background()
	} else {
		select {
		case <-ctx.Done():
			ech := make(chan error)
			sch := make(chan T)
			close(sch)
			close(ech)
			return sch, ech
		default:
		}
	}

	innerCtx, cancel := context.WithCancel(ctx)
	wg := &sync.WaitGroup{}
	errChan := make(chan error, 1)
	errWg := &sync.WaitGroup{}
	errSendOnce := sync.Once{}

	var sendErrFn func(error)
	if stopWhenErr {
		sendErrFn = func(err error) {
			cancel()
			errSendOnce.Do(func() {
				errChan <- err
				close(errChan)
			})
		}
	} else {
		sendErrFn = func(err error) {
			select {
			case errChan <- err:
			default:
				errWg.Add(1)
				go func() { errChan <- err; errWg.Done() }()
			}
		}
	}

	stepWrapperFn := func(ctx context.Context, t T, step func(context.Context, T) error) (err error) {
		defer func() {
			if p := recover(); p != nil {
				err = wrapperPanic(p)
			}
		}()
		err = step(ctx, t)
		return
	}

	startStep := func(step func(context.Context, T) error, dataChan <-chan T) chan T {
		stepSuccessCh := make(chan T, 1)
		limiter := make(chan struct{}, RunPipelineCacheJobs)
		innerWg := &sync.WaitGroup{}
		wg.Add(1)
		go func() {
			defer func() {
				innerWg.Wait()
				close(stepSuccessCh)
				close(limiter)
				wg.Done()
			}()

			for v := range dataChan {
				limiter <- struct{}{}
				innerWg.Add(1)
				go func(job T) {
					defer func() {
						<-limiter
						innerWg.Done()
					}()
					if userCtx != nil {
						select {
						case <-innerCtx.Done():
							select {
							case <-userCtx.Done():
								sendErrFn(userCtx.Err())
							default:
							}
							return
						default:
						}
					}
					err := stepWrapperFn(userCtx, job, step)
					if err != nil {
						sendErrFn(err)
						return
					}
					select {
					case <-innerCtx.Done():
						if userCtx != nil {
							select {
							case <-userCtx.Done():
								sendErrFn(userCtx.Err())
							default:
							}
						}
					case stepSuccessCh <- job:
					}
				}(v)
			}
		}()

		return stepSuccessCh
	}

	// 启动所有 step 协程。
	beginChan := make(chan T, 1)
	var nextCh = beginChan
	for _, step := range steps {
		if step == nil {
			panic("step cannot be nil")
		}
		nextCh = startStep(step, nextCh)
	}

	// 将所有 jobs 发送到通道。
	go func() {
	F:
		for i := range jobs {
			select {
			case <-innerCtx.Done():
				break F
			case beginChan <- jobs[i]:
			}
		}
		close(beginChan)

		wg.Wait()
		if !stopWhenErr {
			errWg.Wait()
			close(errChan)
		} else {
			errSendOnce.Do(func() { close(errChan) })
		}
		cancel() // 释放资源。
	}()

	return nextCh, errChan
}
