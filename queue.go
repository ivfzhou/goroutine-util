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

import "sync"

type Queue[E any] struct {
	head, tail *queueElem[E]
	lock       sync.Mutex
	closed     chan struct{}
	getChan    chan E
	notify     chan struct{}
	once       sync.Once
}

type queueElem[E any] struct {
	elem E
	next *queueElem[E]
}

// Push 向队列尾部加元素，如果队列已被close则不会加元素。
func (q *Queue[E]) Push(elem E) bool {
	q.lock.Lock()
	defer q.lock.Unlock()

	select {
	case <-q.closed:
		return false
	default:
	}

	newElem := &queueElem[E]{elem: elem}
	if q.head == nil {
		q.head = newElem
		q.tail = newElem
	} else {
		q.tail.next = newElem
		q.tail = newElem
	}

	select {
	case q.notify <- struct{}{}:
	default:
	}

	return true
}

// Close 从GetFromChan中获取的chan将被close。
func (q *Queue[E]) Close() {
	q.once.Do(func() {
		q.lock.Lock()
		defer q.lock.Unlock()
		if q.getChan == nil {
			q.init()
			go q.send()
		}
		close(q.closed)
	})
}

// GetFromChan 获取队列头元素。
func (q *Queue[E]) GetFromChan() <-chan E {
	if q.getChan != nil {
		return q.getChan
	}

	q.lock.Lock()
	defer q.lock.Unlock()

	if q.getChan == nil {
		q.init()
		go q.send()
	}

	return q.getChan
}

func (q *Queue[E]) init() {
	q.getChan = make(chan E, 1)
	q.notify = make(chan struct{}, 1)
	q.closed = make(chan struct{})
}

func (q *Queue[E]) send() {
	for {
		q.lock.Lock()
		elem := q.head
		if elem != nil {
			q.head = elem.next
			if q.head == nil {
				q.tail = nil
			}
			q.lock.Unlock()
			q.getChan <- elem.elem
			continue
		}
		q.lock.Unlock()

		select {
		case <-q.closed:
			close(q.getChan)
			close(q.notify)
			return
		case <-q.notify:
		}
	}
}
