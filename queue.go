/*
 * Copyright (c) 2023 ivfzhou
 * io-util is licensed under Mulan PSL v2.
 * You can use this software according to the terms and conditions of the Mulan PSL v2.
 * You may obtain a copy of Mulan PSL v2 at:
 *          http://license.coscl.org.cn/MulanPSL2
 * THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND,
 * EITHER EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT,
 * MERCHANTABILITY OR FIT FOR A PARTICULAR PURPOSE.
 * See the Mulan PSL v2 for more details.
 */

package goroutine_util

import "sync"

// Queue 数据队列。
type Queue[E any] struct {
	headElem, tailElem           *queueElem[E]
	lock                         sync.Mutex
	closeSignal, notifyGetSignal chan struct{}
	getElemChan                  chan E
	closeOnce                    sync.Once
}

type queueElem[E any] struct {
	elem     E
	nextElem *queueElem[E]
}

// Push 向队列尾部加元素，如果队列关闭则不会加元素。
//
// 添加成功返回 true。
func (q *Queue[E]) Push(elem E) bool {
	q.lock.Lock()
	defer q.lock.Unlock()

	select {
	case <-q.closeSignal:
		return false
	default:
	}

	newElem := &queueElem[E]{elem: elem}
	if q.headElem == nil {
		q.headElem = newElem
		q.tailElem = newElem
	} else {
		q.tailElem.nextElem = newElem
		q.tailElem = newElem
	}

	select {
	case q.notifyGetSignal <- struct{}{}:
	default:
	}

	return true
}

// GetFromChan 获取队列头元素。
func (q *Queue[E]) GetFromChan() <-chan E {
	if q.getElemChan != nil {
		return q.getElemChan
	}

	q.lock.Lock()
	defer q.lock.Unlock()

	if q.getElemChan == nil {
		q.init()
		go q.send()
	}

	return q.getElemChan
}

// Close 从 GetFromChan 中获取的 chan 将被关闭。
func (q *Queue[E]) Close() {
	q.closeOnce.Do(func() {
		q.lock.Lock()
		defer q.lock.Unlock()
		if q.getElemChan == nil {
			q.init()
			go q.send()
		}
		close(q.closeSignal)
	})
}

// 初始化成员。
func (q *Queue[E]) init() {
	q.getElemChan = make(chan E, 1)
	q.notifyGetSignal = make(chan struct{}, 1)
	q.closeSignal = make(chan struct{})
}

// 不断向通过发送数据，直到被关闭了。
func (q *Queue[E]) send() {
	for {
		q.lock.Lock()
		elem := q.headElem
		if elem != nil {
			q.headElem = elem.nextElem
			if q.headElem == nil {
				q.tailElem = nil
			}
			q.lock.Unlock()
			q.getElemChan <- elem.elem
			continue
		}
		q.lock.Unlock()

		select {
		case <-q.closeSignal:
			close(q.getElemChan)
			close(q.notifyGetSignal)
			return
		case <-q.notifyGetSignal:
		}
	}
}
