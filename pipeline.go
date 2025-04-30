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
	"reflect"
	"sync"
)

// RunPipeline 将每个 jobs 依次递给 steps 函数处理。一旦某个 step 发生错误或恐慌，就结束处理，返回错误。
// 一个 job 最多在一个 step 中运行一次，且一个 job 一定是依次序递给 steps，前一个 step 处理完毕才会给下一个 step 处理。
// 每个 step 并发运行 jobs。
//
// ctx：上下文，上下文终止时，将终止所有 steps 运行，并返回 ctx.Err()。
//
// stopWhenErr：当 step 发生错误时，是否继续处理 job，当为假时，只会停止将 job 往下一个 step 传递，不会终止运行。
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

	passedJobChan := make(chan T, len(jobs))
	if len(steps) <= 0 || len(jobs) <= 0 { // 检测 step 和 jobs 不为空。
		ech := make(chan error)
		errCh = ech
		successCh = passedJobChan

		close(passedJobChan)
		close(ech)
		return
	}

	// 开启上下文，以便发生错误时，通知终止协程。
	nilCtx := false
	if ctx == nil {
		nilCtx = true
		ctx = context.Background()
	}
	innerCtx, cancel := context.WithCancel(ctx)

	// 将所有 jobs 发送到通道。
	ch := make(chan T, len(jobs))
	go func() {
		for i := range jobs {
			ch <- jobs[i]
		}
		close(ch)
	}()

	// 启动流水线 step。
	startStep := func(ctx, parentCtx context.Context, f func(context.Context, T) error,
		notifyCanceled func(), dataChan <-chan T, stopWhenErr bool) (<-chan T, <-chan error) {

		successCh := make(chan T, cap(dataChan))
		errCh := make(chan error, cap(dataChan))

		go func() {
			// 开启上下文，以便通知本 step 所有协程终止。
			innerCtx, cancel := context.WithCancel(ctx)

			wg := &sync.WaitGroup{}
			defer func() {
				wg.Wait()
				cancel() // 运行完所有任务数据后，释放资源，关闭通道。
				close(successCh)
				close(errCh)
			}()

			for {
				select {
				case <-innerCtx.Done(): // 被通知终止，返回函数。
					select {
					case <-parentCtx.Done(): // 是否是上层上下文终止了。
						errCh <- parentCtx.Err()
					default:
					}
					return
				case t, ok := <-dataChan: // 接收到 job。
					if !ok { // 所有 job 都接收完毕了，返回函数。
						return
					}

					wg.Add(1)
					go func(t T) { // 开启协程消费任务数据。
						defer wg.Done()

						select {
						case <-innerCtx.Done(): // 被通知终止，返回函数。
							select {
							case <-parentCtx.Done(): // 是否是上层上下文终止了。
								errCh <- parentCtx.Err()
							default:
							}
							return
						default:
						}

						var err error
						defer func() {
							if p := recover(); p != nil {
								err = wrapperPanic(p)
							}
							if err != nil { // job 处理失败，将错误发送出去。
								if stopWhenErr { // 是否结束流水线。
									cancel()         // 通知所有本 step 的协程终止。
									notifyCanceled() // 通知流水线所有 step 终止。
								}
								errCh <- err
							} else {
								successCh <- t // 将 job 发送给下一个 step 处理。
							}
						}()
						if nilCtx {
							err = f(nil, t)
						} else {
							err = f(ctx, t)
						}
					}(t)
				}
			}
		}()

		return successCh, errCh
	}

	// 启动所有 step 协程。
	errChans := make([]<-chan error, len(steps))
	var nextCh <-chan T = ch
	for i, step := range steps {
		if step == nil {
			panic("step cannot be nil")
		}
		nextCh, errChans[i] = startStep(innerCtx, ctx, step, cancel, nextCh, stopWhenErr)
	}

	// 将处理完毕的任务数据发送出去。
	go func() {
		for t := range nextCh {
			passedJobChan <- t
		}
		close(passedJobChan) // 所有任务数据都处理完毕。
		cancel()             // 释放资源。
	}()

	return passedJobChan, ListenChan(errChans...)
}

// ListenChan 监听 chans，一旦有一个 chan 激活便立即将 T 发送给 ch，并关闭 ch。
// 若所有 chans 都未曾激活（chan 是空也认为未激活）且都关闭了，则 ch 被关闭。
// 若同时多个 chan 被激活，则随机将一个激活值发送给 ch。
//
// chans：要监听的通道。
func ListenChan[T any](chans ...<-chan T) (ch <-chan T) {
	mergedCh := make(chan T, 1)
	ch = mergedCh
	if len(chans) <= 0 { // 如果没有要监听的通道，则直接返回。
		close(mergedCh)
		return
	}

	// 开启监听。
	go func() {
		// 将所有通道放入反射中监听。
		scs := make([]reflect.SelectCase, 0, len(chans))
		for i := range chans {
			if chans[i] == nil { // 忽略空通道。
				continue
			}
			scs = append(scs, reflect.SelectCase{
				Dir:  reflect.SelectRecv,
				Chan: reflect.ValueOf(chans[i]),
			})
		}

		for {
			if len(scs) <= 0 { // 没有可监听的对象就直接返回函数。
				close(mergedCh)
				return
			}

			chosen, recv, ok := reflect.Select(scs)
			if !ok {
				scs = append(scs[:chosen], scs[chosen+1:]...) // 通道关闭了，忽略它。
			} else {
				mergedCh <- recv.Interface().(T) // 接收到值，发送出去。
				close(mergedCh)
				return
			}
		}
	}()

	return
}
