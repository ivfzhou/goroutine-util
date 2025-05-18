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
	"reflect"
	"sort"
	"testing"
	"time"

	gu "gitee.com/ivfzhou/goroutine-util"
)

func ExampleRunPipeline() {
	type data struct{}
	ctx := context.Background()

	jobs := []*data{{}, {}}
	work1 := func(ctx context.Context, d *data) error { return nil }
	work2 := func(ctx context.Context, d *data) error { return nil }

	successCh, errCh := gu.RunPipeline(ctx, jobs, false, work1, work2)
	for {
		if successCh == nil && errCh == nil {
			break
		}
		select {
		case _, ok := <-successCh:
			if !ok {
				successCh = nil
				continue
			}
			// 处理成功的 jobPtr。
		case _, ok := <-errCh:
			if !ok {
				errCh = nil
				continue
			}
			// 处理错误。
		}
	}
}

func TestRunPipeline(t *testing.T) {
	t.Run("正常运行", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			type job struct {
				x int
			}
			jobCount := 100
			expectedResult := make([]int, jobCount)
			jobs := make([]*job, jobCount)
			for i := 0; i < jobCount; i++ {
				v := rand.Intn(jobCount)
				jobs[i] = &job{x: v}
				expectedResult[i] = v + 3
			}
			sort.Ints(expectedResult)
			step1 := func(_ context.Context, j *job) error { j.x++; return nil }
			step2 := func(_ context.Context, j *job) error { j.x++; return nil }
			step3 := func(_ context.Context, j *job) error { j.x++; return nil }
			successCh, errCh := gu.RunPipeline(context.Background(), jobs, rand.Intn(2) == 1, step1, step2, step3)
			result := make([]int, 0, jobCount)
			for {
				if successCh == nil && errCh == nil {
					break
				}
				select {
				case err, ok := <-errCh:
					if !ok {
						errCh = nil
						continue
					}
					t.Errorf("unexpected error: want false, got %v, value is %v", ok, err)
				case v, ok := <-successCh:
					if !ok {
						successCh = nil
						continue
					}
					result = append(result, v.x)
				}
			}
			sort.Ints(result)
			if !reflect.DeepEqual(result, expectedResult) {
				t.Errorf("unexpected result: want %v, got %v", len(expectedResult), len(result))
			}
		}
	})

	t.Run("发生错误", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			type job struct {
				i int
				x int
			}
			jobCount := rand.Intn(91) + 90
			expectedErr := errors.New("expected error")
			expectedResult := make(map[int]int, jobCount)
			jobs := make([]*job, jobCount)
			for i := 0; i < jobCount; i++ {
				v := rand.Intn(jobCount)
				jobs[i] = &job{i: i, x: v}
				expectedResult[i] = v + 3
			}
			returnErrorIndex := rand.Intn(3)
			occurErrorIndex := rand.Intn(jobCount / 2)
			step1 := func(_ context.Context, j *job) error {
				if j.i == occurErrorIndex && returnErrorIndex == 0 {
					return expectedErr
				}
				j.x++
				return nil
			}
			step2 := func(_ context.Context, j *job) error {
				if j.i == occurErrorIndex && returnErrorIndex == 1 {
					return expectedErr
				}
				j.x++
				return nil
			}
			step3 := func(_ context.Context, j *job) error {
				if j.i == occurErrorIndex && returnErrorIndex == 2 {
					return expectedErr
				}
				j.x++
				return nil
			}
			stopWhenErr := rand.Intn(2) == 1
			successCh, errCh := gu.RunPipeline(context.Background(), jobs, stopWhenErr, step1, step2, step3)
			result := make([]*job, 0, jobCount)
			for {
				if successCh == nil && errCh == nil {
					break
				}
				select {
				case v, ok := <-successCh:
					if !ok {
						successCh = nil
						continue
					}
					result = append(result, v)
				case err, ok := <-errCh:
					if !ok {
						errCh = nil
						continue
					}
					if !errors.Is(err, expectedErr) {
						t.Errorf("unexpected error: want %v, got %v", expectedErr, err)
					}
				}
			}
			if stopWhenErr {
				for _, v := range result {
					if expectedResult[v.i] != v.x {
						t.Errorf("unexpected result: want %v, got %v", expectedResult[v.i], v.x)
					}
				}
			} else {
				if len(result) != len(expectedResult)-1 {
					t.Errorf("unexpected result: want %v, got %v", len(expectedResult)-1, len(result))
				}
				for _, v := range result {
					if expectedResult[v.i] != v.x {
						t.Errorf("unexpected result: want %v, got %v", expectedResult[v.i], v.x)
					}
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
			jobCount := rand.Intn(91) + 10
			expectedErr := errors.New("expected error")
			expectedResult := make(map[int]int, jobCount)
			jobs := make([]*job, jobCount)
			for i := 0; i < jobCount; i++ {
				v := rand.Intn(jobCount)
				jobs[i] = &job{i: i, x: v}
				expectedResult[i] = v + 3
			}
			panicIndex := rand.Intn(3)
			occurPanicIndex := rand.Intn(jobCount / 2)
			step1 := func(_ context.Context, j *job) error {
				time.Sleep(time.Millisecond * 10)
				if j.i == occurPanicIndex && panicIndex == 0 {
					panic(expectedErr)
				}
				j.x++
				return nil
			}
			step2 := func(_ context.Context, j *job) error {
				time.Sleep(time.Millisecond * 10)
				if j.i == occurPanicIndex && panicIndex == 1 {
					panic(expectedErr)
				}
				j.x++
				return nil
			}
			step3 := func(_ context.Context, j *job) error {
				time.Sleep(time.Millisecond * 10)
				if j.i == occurPanicIndex && panicIndex == 2 {
					panic(expectedErr)
				}
				j.x++
				return nil
			}
			stopWhenErr := rand.Intn(2) == 1
			successCh, errCh := gu.RunPipeline(context.Background(), jobs, stopWhenErr, step1, step2, step3)
			result := make([]*job, 0, jobCount)
			for {
				if errCh == nil && successCh == nil {
					break
				}
				select {
				case v, ok := <-successCh:
					if !ok {
						successCh = nil
						continue
					}
					result = append(result, v)
				case err, ok := <-errCh:
					if !ok {
						errCh = nil
						continue
					}
					if !errors.Is(err, expectedErr) {
						t.Errorf("unexpected error: want %v, got %v", expectedErr, err)
					}
				}
			}
			if stopWhenErr {
				for _, v := range result {
					if expectedResult[v.i] != v.x {
						t.Errorf("unexpected result: want %d, got %d", expectedResult[v.i], v.x)
					}
				}
			} else {
				if len(result) != len(expectedResult)-1 {
					t.Errorf("unexpected result: want %v, got %v", len(expectedResult)-1, len(result))
				}
				for _, v := range result {
					if expectedResult[v.i] != v.x {
						t.Errorf("unexpected result: want %v, got %v", expectedResult[v.i], v.x)
					}
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
			ctx, cancel := NewCtxCancelWithError()
			jobCount := rand.Intn(91) + 10
			expectedErr := errors.New("expected error")
			expectedResult := make(map[int]int, jobCount)
			jobs := make([]*job, jobCount)
			for i := 0; i < jobCount; i++ {
				v := rand.Intn(jobCount)
				jobs[i] = &job{i: i, x: v}
				expectedResult[i] = v + 3
			}
			cancelIndex := rand.Intn(3)
			occurCancelIndex := rand.Intn(jobCount / 2)
			step1 := func(_ context.Context, j *job) error {
				time.Sleep(time.Millisecond * 10)
				if j.i == occurCancelIndex && cancelIndex == 0 {
					cancel(expectedErr)
				}
				j.x++
				return nil
			}
			step2 := func(_ context.Context, j *job) error {
				time.Sleep(time.Millisecond * 10)
				if j.i == occurCancelIndex && cancelIndex == 1 {
					cancel(expectedErr)
				}
				j.x++
				return nil
			}
			step3 := func(_ context.Context, j *job) error {
				time.Sleep(time.Millisecond * 10)
				if j.i == occurCancelIndex && cancelIndex == 2 {
					cancel(expectedErr)
				}
				j.x++
				return nil
			}
			stopWhenErr := rand.Intn(2) == 1
			successCh, errCh := gu.RunPipeline(ctx, jobs, stopWhenErr, step1, step2, step3)
			result := make([]*job, 0, jobCount)
			hasErr := false
			for {
				if successCh == nil && errCh == nil {
					break
				}
				select {
				case v, ok := <-successCh:
					if !ok {
						successCh = nil
						continue
					}
					result = append(result, v)
				case err, ok := <-errCh:
					if !ok {
						errCh = nil
						continue
					}
					hasErr = true
					if !errors.Is(err, expectedErr) {
						t.Errorf("unexpected error: want %v, got %v", expectedErr, err)
					}
				}
			}
			if !hasErr {
				if len(result) != len(expectedResult) {
					t.Errorf("unexpected result: want %v, got %v", len(expectedResult), len(result))
				}
				for _, v := range result {
					if expectedResult[v.i] != v.x {
						t.Errorf("unexpected result: want %v, got %v", expectedResult[v.i], v.x)
					}
				}
			} else {
				for _, v := range result {
					if expectedResult[v.i] < v.x {
						t.Errorf("unexpected result: want >= %v, got %v", expectedResult[v.i], v.x)
					}
				}
			}
		}
	})

	t.Run("大量数据", func(t *testing.T) {
		type job struct {
			x int
		}
		jobCount := 1024*1024*(rand.Intn(5)+1) + 10
		stepCount := 50
		expectedResult := make([]int, jobCount)
		jobs := make([]*job, jobCount)
		for i := 0; i < jobCount; i++ {
			v := rand.Intn(jobCount)
			jobs[i] = &job{x: v}
			expectedResult[i] = v + stepCount
		}
		sort.Ints(expectedResult)
		steps := make([]func(context.Context, *job) error, stepCount)
		for i := range steps {
			steps[i] = func(_ context.Context, j *job) error { j.x++; return nil }
		}
		successCh, errCh := gu.RunPipeline(context.Background(), jobs, rand.Intn(2) == 1, steps...)
		result := make([]int, 0, jobCount)
		for v := range successCh {
			result = append(result, v.x)
		}
		sort.Ints(result)
		if !reflect.DeepEqual(result, expectedResult) {
			t.Errorf("unexpected result: want %v, got %v", len(expectedResult), len(result))
		}
		if err, ok := <-errCh; ok {
			t.Errorf("unexpected error: want false, got %v, value is %v", ok, err)
		}
	})
}
