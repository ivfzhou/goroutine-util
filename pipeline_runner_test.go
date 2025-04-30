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
	"context"
	"math/rand"
	"testing"

	gu "gitee.com/ivfzhou/goroutine-util"
)

func ExampleNewPipelineRunner() {
	type job struct{}
	ctx := context.Background()

	step1 := func(context.Context, *job) bool { return true }
	step2 := func(context.Context, *job) bool { return true }
	step3 := func(context.Context, *job) bool { return true }
	push, successCh, endPush := gu.NewPipelineRunner(ctx, step1, step2, step3)

	// 将任务数据推送进去处理。
	go func() {
		jobs := make([]*job, 100)
		for i := range jobs {
			push(jobs[i])
		}
		endPush() // 结束推送。
	}()

	// 获取处理成功的任务。
	for v := range successCh {
		_ = v
	}
}

func TestNewPipelineRunner(t *testing.T) {
	t.Run("正常运行", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			type job struct {
				i int
				x int
			}
			const count = 1000
			jobs := make([]*job, count)
			expectedResult := make(map[int]int, count)
			for i := range jobs {
				jobs[i] = &job{x: rand.Intn(100), i: i}
				expectedResult[i] = jobs[i].x + 3
			}

			ctx := context.Background()
			worker1 := func(ctx context.Context, j *job) bool { j.x++; return true }
			worker2 := func(ctx context.Context, j *job) bool { j.x++; return true }
			worker3 := func(ctx context.Context, j *job) bool { j.x++; return true }
			push, successCh, endPush := gu.NewPipelineRunner(ctx, worker1, worker2, worker3)

			go func() {
				for i := range jobs {
					b := push(jobs[i])
					if !b {
						t.Errorf("unexpected result: want true, got %v", b)
					}
				}
				endPush()
			}()

			for v := range successCh {
				if expectedResult[v.i] != v.x {
					t.Errorf("unexpected result: want %v, got %v", expectedResult[v.i], v.x)
				}
			}
		}
	})

	t.Run("不往下传递", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			type job struct {
				i int
				x int
			}
			const count = 1000
			jobs := make([]*job, count)
			expectedResult := make(map[int]int, count)
			dropIndex := rand.Intn(count / 2)
			for i := range jobs {
				jobs[i] = &job{x: rand.Intn(100), i: i}
				expectedResult[i] = jobs[i].x + 3
			}

			ctx := context.Background()
			index := rand.Intn(3)
			worker1 := func(ctx context.Context, j *job) bool {
				if j.i == dropIndex && index == 0 {
					return false
				}
				j.x++
				return true
			}
			worker2 := func(ctx context.Context, j *job) bool {
				if j.i == dropIndex && index == 1 {
					return false
				}
				j.x++
				return true
			}
			worker3 := func(ctx context.Context, j *job) bool {
				if j.i == dropIndex && index == 2 {
					return false
				}
				j.x++
				return true
			}
			push, successCh, endPush := gu.NewPipelineRunner(ctx, worker1, worker2, worker3)

			go func() {
				for i := range jobs {
					b := push(jobs[i])
					if !b {
						t.Errorf("unexpected result: want true, got %v", b)
					}
				}
				endPush()
			}()

			for v := range successCh {
				if dropIndex == v.i {
					if expectedResult[v.i] <= v.x {
						t.Errorf("unexpected result: want %v, got %v", expectedResult[v.i], v.x)
					}
				} else {
					if expectedResult[v.i] != v.x {
						t.Errorf("unexpected result: want %v, got %v", expectedResult[v.i], v.x)
					}
				}
			}
		}
	})
}
