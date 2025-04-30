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
	select {
	case <-successCh:
		// 处理成功的 job。
	case <-errCh:
		// 处理错误。
	}
}

func TestRunPipeline(t *testing.T) {
	t.Run("正常运行", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			type job struct {
				x int
			}
			ctx := context.Background()
			jobCount := rand.Intn(91) + 10
			expectedResult := make([]int, jobCount)
			jobs := make([]*job, jobCount)
			for i := 0; i < jobCount; i++ {
				v := rand.Intn(jobCount)
				jobs[i] = &job{x: v}
				expectedResult[i] = v + 3
			}
			sort.Ints(expectedResult)

			step1 := func(_ context.Context, j *job) error { time.Sleep(time.Millisecond * 100); j.x++; return nil }
			step2 := func(_ context.Context, j *job) error { time.Sleep(time.Millisecond * 100); j.x++; return nil }
			step3 := func(_ context.Context, j *job) error { time.Sleep(time.Millisecond * 100); j.x++; return nil }
			stopWhenErr := rand.Intn(2) == 1
			successCh, errCh := gu.RunPipeline(ctx, jobs, stopWhenErr, step1, step2, step3)

			result := make([]int, 0, jobCount)
			for v := range successCh {
				result = append(result, v.x)
			}
			err, ok := <-errCh
			if ok {
				t.Errorf("unexpected error: want false, got %v, value is %v", ok, err)
			}
			sort.Ints(result)
			if !reflect.DeepEqual(result, expectedResult) {
				t.Errorf("unexpected result: want %d, got %d", len(expectedResult), len(result))
			}
		}
	})

	t.Run("发生错误", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			type job struct {
				i   int
				x   int
				err error
			}
			ctx := context.Background()
			jobCount := rand.Intn(91) + 10
			expectedErr := errors.New("expected error")
			occurErrorIndex := rand.Intn(jobCount / 2)
			expectedResult := make(map[int]int, jobCount)
			jobs := make([]*job, jobCount)
			for i := 0; i < jobCount; i++ {
				v := rand.Intn(jobCount)
				jobs[i] = &job{i: i, x: v}
				if occurErrorIndex == i {
					jobs[i].err = expectedErr
				}
				expectedResult[i] = v + 3
			}

			returnErrorIndex := rand.Intn(3)
			step1 := func(_ context.Context, j *job) error {
				time.Sleep(time.Millisecond * 100)
				j.x++
				if returnErrorIndex == 0 {
					return j.err
				}
				return nil
			}
			step2 := func(_ context.Context, j *job) error {
				time.Sleep(time.Millisecond * 100)
				j.x++
				if returnErrorIndex == 1 {
					return j.err
				}
				return nil
			}
			step3 := func(_ context.Context, j *job) error {
				time.Sleep(time.Millisecond * 100)
				j.x++
				if returnErrorIndex == 2 {
					return j.err
				}
				return nil
			}
			stopWhenErr := rand.Intn(2) == 1
			successCh, errCh := gu.RunPipeline(ctx, jobs, stopWhenErr, step1, step2, step3)

			result := make([]*job, 0, jobCount)
			for v := range successCh {
				result = append(result, v)
			}
			err, ok := <-errCh
			if !ok {
				t.Errorf("unexpected error: want true, got %v, value is %v", ok, err)
			}
			if !errors.Is(err, expectedErr) {
				t.Errorf("unexpected error: want %v, got %v", expectedErr, err)
			}
			if stopWhenErr {
				for _, v := range result {
					if expectedResult[v.i] < v.x {
						t.Errorf("unexpected result: want %d, got %d", expectedResult[v.i], v.x)
					}
				}
			} else {
				if len(result) != len(expectedResult)-1 {
					t.Errorf("unexpected result: want %d, got %d", len(expectedResult)-1, len(result))
				}
				for _, v := range result {
					if expectedResult[v.i] != v.x {
						t.Errorf("unexpected result: want %d, got %d", expectedResult[v.i], v.x)
					}
				}
			}
		}
	})

	t.Run("发生恐慌", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			type job struct {
				i   int
				x   int
				err error
			}
			ctx := context.Background()
			jobCount := rand.Intn(91) + 10
			expectedErr := errors.New("expected error")
			occurPanicIndex := rand.Intn(jobCount / 2)
			expectedResult := make(map[int]int, jobCount)
			jobs := make([]*job, jobCount)
			for i := 0; i < jobCount; i++ {
				v := rand.Intn(jobCount)
				jobs[i] = &job{i: i, x: v}
				if occurPanicIndex == i {
					jobs[i].err = expectedErr
				}
				expectedResult[i] = v + 3
			}

			panicIndex := rand.Intn(3)
			step1 := func(_ context.Context, j *job) error {
				time.Sleep(time.Millisecond * 100)
				j.x++
				if panicIndex == 0 && j.err != nil {
					panic(j.err)
				}
				return nil
			}
			step2 := func(_ context.Context, j *job) error {
				time.Sleep(time.Millisecond * 100)
				j.x++
				if panicIndex == 1 && j.err != nil {
					panic(j.err)
				}
				return nil
			}
			step3 := func(_ context.Context, j *job) error {
				time.Sleep(time.Millisecond * 100)
				j.x++
				if panicIndex == 2 && j.err != nil {
					panic(j.err)
				}
				return nil
			}
			stopWhenErr := rand.Intn(2) == 1
			successCh, errCh := gu.RunPipeline(ctx, jobs, stopWhenErr, step1, step2, step3)

			result := make([]*job, 0, jobCount)
			for v := range successCh {
				result = append(result, v)
			}
			err, ok := <-errCh
			if !ok {
				t.Errorf("unexpected error: want true, got %v, value is %v", ok, err)
			}
			if !errors.Is(err, expectedErr) {
				t.Errorf("unexpected error: want %v, got %v", expectedErr, err)
			}
			if stopWhenErr {
				for _, v := range result {
					if expectedResult[v.i] < v.x {
						t.Errorf("unexpected result: want %d, got %d", expectedResult[v.i], v.x)
					}
				}
			} else {
				if len(result) != len(expectedResult)-1 {
					t.Errorf("unexpected result: want %d, got %d", len(expectedResult)-1, len(result))
				}
				for _, v := range result {
					if expectedResult[v.i] != v.x {
						t.Errorf("unexpected result: want %d, got %d", expectedResult[v.i], v.x)
					}
				}
			}
		}
	})

	t.Run("上下文终止", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			type job struct {
				i   int
				x   int
				err error
			}
			ctx, cancel := newCtxCancelWithError()
			jobCount := rand.Intn(91) + 10
			expectedErr := errors.New("expected error")
			occurCancelIndex := rand.Intn(jobCount / 2)
			expectedResult := make(map[int]int, jobCount)
			jobs := make([]*job, jobCount)
			for i := 0; i < jobCount; i++ {
				v := rand.Intn(jobCount)
				jobs[i] = &job{i: i, x: v}
				if occurCancelIndex == i {
					jobs[i].err = expectedErr
				}
				expectedResult[i] = v + 3
			}

			cancelIndex := rand.Intn(3)
			step1 := func(_ context.Context, j *job) error {
				time.Sleep(time.Millisecond * 100)
				j.x++
				if cancelIndex == 0 && j.err != nil {
					cancel(j.err)
				}
				return nil
			}
			step2 := func(_ context.Context, j *job) error {
				time.Sleep(time.Millisecond * 100)
				j.x++
				if cancelIndex == 1 && j.err != nil {
					cancel(j.err)
				}
				return nil
			}
			step3 := func(_ context.Context, j *job) error {
				time.Sleep(time.Millisecond * 100)
				j.x++
				if cancelIndex == 2 && j.err != nil {
					cancel(j.err)
				}
				return nil
			}
			stopWhenErr := rand.Intn(2) == 1
			successCh, errCh := gu.RunPipeline(ctx, jobs, stopWhenErr, step1, step2, step3)

			result := make([]*job, 0, jobCount)
			for v := range successCh {
				result = append(result, v)
			}
			err, ok := <-errCh
			if ok && !errors.Is(err, expectedErr) {
				t.Errorf("unexpected error: want %v, got %v", expectedErr, err)
			}
			if !ok && len(result) != len(expectedResult) || ok && len(result) >= len(expectedResult) {
				t.Errorf("unexpected result: want %d, got %d,value is %v", len(expectedResult), len(result), ok)
			}
			for _, v := range result {
				if expectedResult[v.i] < v.x {
					t.Errorf("unexpected result: want %d, got %d", expectedResult[v.i], v.x)
				}
			}
		}
	})
}

func TestListenChan(t *testing.T) {
	t.Run("正常运行", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			err1 := make(chan error)
			err2 := make(chan error)
			err3 := make(chan error)
			errChans := []<-chan error{
				err1,
				err2,
				err3,
			}

			expectedErr := errors.New("expected error")
			go func() {
				time.Sleep(time.Second)
				err1 <- expectedErr
				close(err1)
				close(err2)
				close(err3)
			}()
			ch := gu.ListenChan(errChans...)
			if err := <-ch; !errors.Is(err, expectedErr) {
				t.Errorf("unexpected result: want %v, got %v", expectedErr, err)
			}
			err, ok := <-ch
			if ok {
				t.Errorf("unexpected result: want false, got %v, value is %v", ok, err)
			}
		}
	})

	t.Run("通道都关闭未发送值", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			err1 := make(chan error)
			err2 := make(chan error)
			err3 := make(chan error)
			errChans := []<-chan error{
				err1,
				err2,
				err3,
			}

			go func() {
				time.Sleep(time.Second)
				close(err1)
				close(err2)
				close(err3)
			}()
			ch := gu.ListenChan(errChans...)
			if err := <-ch; err != nil {
				t.Errorf("unexpected result: want nil, got %v", err)
			}
			err, ok := <-ch
			if ok {
				t.Errorf("unexpected result: want false, got %v, value is %v", ok, err)
			}
		}
	})

	t.Run("包含空通道", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			err1 := make(chan error)
			err2 := make(chan error)
			err3 := make(chan error)
			errChans := []<-chan error{
				err1,
				err2,
				nil,
				err3,
			}

			go func() {
				time.Sleep(time.Second)
				close(err1)
				close(err2)
				close(err3)
			}()
			ch := gu.ListenChan(errChans...)
			if err := <-ch; err != nil {
				t.Errorf("unexpected result: want nil, got %v", err)
			}
			err, ok := <-ch
			if ok {
				t.Errorf("unexpected result: want false, got %v, value is %v", ok, err)
			}
		}
	})

	t.Run("多个通道发送了值", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			err1 := make(chan error)
			err2 := make(chan error)
			err3 := make(chan error)
			errChans := []<-chan error{
				err1,
				err2,
				err3,
			}

			expectedErr := errors.New("expected error")
			go func() {
				time.Sleep(time.Second)
				err1 <- expectedErr
				err2 <- expectedErr
				close(err1)
				close(err2)
				close(err3)
			}()
			ch := gu.ListenChan(errChans...)
			if err := <-ch; !errors.Is(err, expectedErr) {
				t.Errorf("unexpected result: want %v, got %v", expectedErr, err)
			}
			err, ok := <-ch
			if ok {
				t.Errorf("unexpected result: want false, got %v, value is %v", ok, err)
			}
		}
	})
}
