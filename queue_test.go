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

package goroutine_util_test

import (
	"testing"

	goroutine_util "gitee.com/ivfzhou/goroutine-util"
)

func TestQueue(t *testing.T) {
	queue := &goroutine_util.Queue[int]{}
	queue.Push(1)
	queue.Push(2)
	queue.Push(3)
	queue.Push(4)
	queue.Push(5)
	elem := <-queue.GetFromChan()
	if 1 != elem {
		t.Error("queue: queue elem does not match", elem)
	}
	elem = <-queue.GetFromChan()
	if 2 != elem {
		t.Error("queue: queue elem does not match", elem)
	}
	elem = <-queue.GetFromChan()
	if 3 != elem {
		t.Error("queue: queue elem does not match", elem)
	}
	elem = <-queue.GetFromChan()
	if 4 != elem {
		t.Error("queue: queue elem does not match", elem)
	}
	queue.Close()
	elem = <-queue.GetFromChan()
	if 5 != elem {
		t.Error("queue: queue elem does not match", elem)
	}

	queue.Push(6)
	queue.Push(7)
	elem = <-queue.GetFromChan()
	if 0 != elem {
		t.Error("queue: queue elem does not match", elem)
	}
	if 0 != elem {
		t.Error("queue: queue elem does not match", elem)
	}
}
