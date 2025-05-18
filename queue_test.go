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

package goroutine_util_test

import (
	"math/rand"
	"testing"

	gu "gitee.com/ivfzhou/goroutine-util"
)

func TestQueue(t *testing.T) {
	t.Run("正常使用", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			expectedResult := make([]int, 1000)
			for i := range expectedResult {
				expectedResult[i] = rand.Intn(100)
			}
			queue := &gu.Queue[int]{}
			go func() {
				for i := range expectedResult {
					ok := queue.Push(expectedResult[i])
					if !ok {
						t.Errorf("unexpected result: want true, got %v", ok)
					}
				}
				queue.Close()
			}()
			index := 0
			for v := range queue.GetFromChan() {
				if v != expectedResult[index] {
					t.Errorf("unexpected result: want %v, got %v", expectedResult[index], v)
				}
				index++
			}
		}
	})

	t.Run("关闭后再 Push", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			queue := &gu.Queue[int]{}
			ok := queue.Push(1)
			if !ok {
				t.Errorf("unexpected result: want true, got %v", ok)
			}
			queue.Close()

			ok = queue.Push(2)
			if ok {
				t.Errorf("unexpected result: want false, got %v", ok)
			}
			value := <-queue.GetFromChan()
			if value != 1 {
				t.Errorf("unexpected result: want 1, got %v", value)
			}
			_, ok = <-queue.GetFromChan()
			if ok {
				t.Errorf("unexpected result: want false, got %v", ok)
			}
		}
	})

	t.Run("大量数据", func(t *testing.T) {
		data := make([]int, 1024*1024*(rand.Intn(5)+1)+10)
		for i := range data {
			data[i] = rand.Intn(100)
		}

		queue := &gu.Queue[int]{}
		go func() {
			for i := range data {
				ok := queue.Push(data[i])
				if !ok {
					t.Errorf("unexpected result: want true, got %v", ok)
				}
			}
			queue.Close()
		}()

		index := 0
		for v := range queue.GetFromChan() {
			if v != data[index] {
				t.Errorf("unexpected result: want %v, got %v", data[index], v)
			}
			index++
		}
	})
}
