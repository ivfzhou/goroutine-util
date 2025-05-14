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
	"errors"
	"math/rand"
	"testing"

	gu "gitee.com/ivfzhou/goroutine-util"
)

func TestRunData(t *testing.T) {
	t.Run("正常运行", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			type job struct {
				i int
				x int
			}
			var jobCount = 100
			jobs := make([]*job, jobCount)
			expectedResult := make(map[int]int, jobCount)
			for i := 0; i < jobCount; i++ {
				jobs[i] = &job{i: i, x: rand.Intn(100)}
				expectedResult[i] = jobs[i].x + 1
			}

			err := gu.RunData(context.Background(), func(ctx context.Context, t *job) error {
				t.x++
				return nil
			}, rand.Intn(2) == 1, jobs...)
			if err != nil {
				t.Errorf("unexpected error: want nil, got %v", err)
			}

			for i := range jobs {
				if expectedResult[i] != jobs[i].x {
					t.Errorf("unexpected result: want %v, got %v", expectedResult[i], jobs[i].x)
				}
			}
		}
	})

	t.Run("发生错误", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			type job struct {
				i int
				x int
			}
			var jobCount = 100
			jobs := make([]*job, jobCount)
			expectedResult := make(map[int]int, jobCount)
			expectedErr := errors.New("expected error")
			for i := 0; i < jobCount; i++ {
				jobs[i] = &job{i: i, x: rand.Intn(100)}
				expectedResult[i] = jobs[i].x + 1
			}

			occurErrorIndex := rand.Intn(jobCount / 2)
			err := gu.RunData(context.Background(), func(ctx context.Context, t *job) error {
				if occurErrorIndex == t.i {
					return expectedErr
				}
				t.x++
				return nil
			}, rand.Intn(2) == 1, jobs...)
			if !errors.Is(err, expectedErr) {
				t.Errorf("unexpected error: want %v, got %v", expectedErr, err)
			}

			for i := range jobs {
				if expectedResult[i] < jobs[i].x {
					t.Errorf("unexpected result: want <= %v, got %v", expectedResult[i], jobs[i].x)
				}
			}
		}
	})

	t.Run("上下文终止", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			type job struct {
				i int
				x int
			}
			var jobCount = 100
			jobs := make([]*job, jobCount)
			expectedResult := make(map[int]int, jobCount)
			expectedErr := errors.New("expected error")
			for i := 0; i < jobCount; i++ {
				jobs[i] = &job{i: i, x: rand.Intn(100)}
				expectedResult[i] = jobs[i].x + 1
			}

			ctx, cancel := newCtxCancelWithError()
			occurErrorIndex := rand.Intn(jobCount / 2)
			err := gu.RunData(ctx, func(ctx context.Context, t *job) error {
				if occurErrorIndex == t.i {
					cancel(expectedErr)
				}
				t.x++
				return nil
			}, rand.Intn(2) == 1, jobs...)
			if !errors.Is(err, expectedErr) {
				t.Errorf("unexpected error: want %v, got %v", expectedErr, err)
			}

			for i := range jobs {
				if expectedResult[i] < jobs[i].x {
					t.Errorf("unexpected result: want <= %v, got %v", expectedResult[i], jobs[i].x)
				}
			}
		}
	})

	t.Run("发生恐慌", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			type job struct {
				i int
				x int
			}
			var jobCount = 100
			jobs := make([]*job, jobCount)
			expectedResult := make(map[int]int, jobCount)
			expectedErr := errors.New("expected error")
			for i := 0; i < jobCount; i++ {
				jobs[i] = &job{i: i, x: rand.Intn(100)}
				expectedResult[i] = jobs[i].x + 1
			}

			occurErrorIndex := rand.Intn(jobCount / 2)
			err := gu.RunData(context.Background(), func(ctx context.Context, t *job) error {
				if occurErrorIndex == t.i {
					panic(expectedErr)
				}
				t.x++
				return nil
			}, rand.Intn(2) == 1, jobs...)
			if !errors.Is(err, expectedErr) {
				t.Errorf("unexpected error: want %v, got %v", expectedErr, err)
			}

			for i := range jobs {
				if expectedResult[i] < jobs[i].x {
					t.Errorf("unexpected result: want <= %v, got %v", expectedResult[i], jobs[i].x)
				}
			}
		}
	})
}
