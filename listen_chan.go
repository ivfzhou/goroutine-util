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

import "reflect"

// ListenChan 监听 chans，一旦有一个 chan 激活便立即将 T 发送给 sendCh，并关闭 sendCh。
// 若所有 chans 都未曾激活（chan 是空和关闭认为未激活），则 sendCh 被关闭。
// 若同时多个 chan 被激活，则随机将一个激活值发送给 sendCh。
//
// chans：要监听的通道。
//
// sendCh：接收到的 T 从该通道发出。
func ListenChan[T any](chans ...<-chan T) (ch <-chan T) {
	mergedCh := make(chan T, 1)
	ch = mergedCh
	if len(chans) <= 0 { // 如果没有要监听的通道，则直接返回。
		close(mergedCh)
		return
	}

	// 将所有通道放入反射中监听。
	scs := make([]reflect.SelectCase, 0, len(chans))
	for i := range chans {
		if chans[i] == nil { // 忽略空通道。
			continue
		}
		scs = append(scs, reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(chans[i])})
	}
	if len(scs) <= 0 {
		close(mergedCh)
		return
	}

	// 开启监听。
	go func() {
		for {
			if len(scs) <= 0 {
				close(mergedCh)
				return
			}

			chosen, recv, ok := reflect.Select(scs)
			if !ok {
				// copy(scs[:chosen], scs[chosen+1:])
				// scs = scs[:len(scs)-1]
				scs = append(scs[:chosen], scs[chosen+1:]...)
			} else {
				reflect.ValueOf(mergedCh).Send(recv)
				close(mergedCh)
				return
			}
		}
	}()

	return
}
