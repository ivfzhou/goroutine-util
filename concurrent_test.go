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
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	goroutine_util "gitee.com/ivfzhou/goroutine-util"
)

const jobCount = 256

func TestRunConcurrently(t *testing.T) {
	ctx := context.Background()
	x := int32(0)
	fns := make([]func(ctx context.Context) error, jobCount)
	for i := 0; i < jobCount; i++ {
		fns[i] = func(ctx context.Context) error {
			atomic.AddInt32(&x, 1)
			return nil
		}
	}
	wait := goroutine_util.RunConcurrently(ctx, fns...)
	err := wait(true)
	if err != nil {
		t.Error("concurrent: err is not nil", err)
	}
	if x != jobCount {
		t.Error("concurrent: job count is unexpected", x)
	}
}

func TestRunConcurrentlyErr(t *testing.T) {
	ctx := context.Background()
	x := int32(0)
	fns := make([]func(ctx context.Context) error, jobCount)
	err := errors.New("expected error")
	for i := 0; i < jobCount; i++ {
		fns[i] = func(ctx context.Context) error {
			if atomic.AddInt32(&x, 1) == 5 {
				return err
			}
			return nil
		}
	}
	wait := goroutine_util.RunConcurrently(ctx, fns...)
	werr := wait(true)
	if err == nil {
		t.Error("concurrent: err is nil", err)
	}
	if werr != err {
		t.Error("concurrent: err not equal", werr, err)
	}
}

func TestRunConcurrentlyPanic(t *testing.T) {
	ctx := context.Background()
	x := int32(0)
	fns := make([]func(ctx context.Context) error, jobCount)
	err := errors.New("expected error")
	for i := 0; i < jobCount; i++ {
		fns[i] = func(ctx context.Context) error {
			if atomic.AddInt32(&x, 1) == 5 {
				panic(err)
			}
			return nil
		}
	}
	wait := goroutine_util.RunConcurrently(ctx, fns...)
	werr := wait(true)
	if werr == nil {
		t.Error("concurrent: err is nil", werr)
	}
	if werr != nil && !strings.Contains(werr.Error(), err.Error()) {
		t.Error("concurrent: err not equal", werr, err)
	}
}

func TestRunSequentially(t *testing.T) {
	ctx := context.Background()
	x := 1
	err := goroutine_util.RunSequentially(ctx, func(ctx context.Context) error {
		if x != 1 {
			t.Error("concurrent: x is not 1", x)
			return nil
		}
		x++
		return nil
	}, func(ctx context.Context) error {
		if x != 2 {
			t.Error("concurrent: x is not 2", x)
			return nil
		}
		x++
		return nil
	})
	if err != nil {
		t.Error("concurrent: err is not nil", err)
	}
	if x != 3 {
		t.Error("concurrent: x is not 3", x)
	}
}

func TestRunSequentiallyErr(t *testing.T) {
	ctx := context.Background()
	x := 0
	werr := errors.New("expected error")
	err := goroutine_util.RunSequentially(ctx, func(ctx context.Context) error {
		x++
		return nil
	}, func(ctx context.Context) error {
		x++
		return werr
	}, func(ctx context.Context) error {
		x++
		return nil
	})
	if err == nil {
		t.Error("concurrent: err is nil", err)
	}
	if x != 2 {
		t.Error("concurrent: x is not 2", x)
	}
	if err != werr {
		t.Error("concurrent: unexpected err", err)
	}
}

func TestRunSequentiallyPanic(t *testing.T) {
	ctx := context.Background()
	x := 0
	werr := errors.New("expected error")
	err := goroutine_util.RunSequentially(ctx, func(ctx context.Context) error {
		x++
		return nil
	}, func(ctx context.Context) error {
		x++
		panic(werr)
	}, func(ctx context.Context) error {
		x++
		return nil
	})
	if err == nil {
		t.Error("concurrent: err is nil", err)
	}
	if err != nil && !strings.Contains(err.Error(), werr.Error()) {
		t.Error("concurrent: err is not equaled", err, werr)
	}
	if x != 2 {
		t.Error("x is not 2", x)
	}
}

func TestRunData(t *testing.T) {
	ctx := context.Background()
	count := int32(0)
	err := goroutine_util.RunData(ctx, func(ctx context.Context, t int32) error {
		atomic.AddInt32(&count, t)
		return nil
	}, true, 1, 2, 3, 4)
	if err != nil {
		t.Error("concurrent: unexpected error", err)
	}
	if count != 10 {
		t.Error("concurrent: unexpected count", count)
	}

	expectedErr := errors.New("expected error")
	err = goroutine_util.RunData(ctx, func(ctx context.Context, t int32) error {
		if t == 3 {
			return expectedErr
		}
		return nil
	}, true, 1, 2, 3, 4)
	if err != expectedErr {
		t.Error("concurrent: unexpected error", err)
	}

	err = goroutine_util.RunData(ctx, func(ctx context.Context, t int32) error {
		if t == 3 {
			panic(t)
		}
		return nil
	}, true, 1, 2, 3, 4)
	if err == nil || !strings.Contains(err.Error(), "3") {
		t.Error("concurrent: unexpected error", err)
	}
}

func TestRunPipeline(t *testing.T) {
	type data struct {
		name string
		x    int
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	jobs := []*data{{"job1", 0}, {"job2", 0}}
	work1 := func(ctx context.Context, d *data) error {
		// t.Logf("work1 job %s", d.name)
		if d.x == 0 {
			d.x = 1
			return nil
		} else {
			return errors.New("x != 0")
		}
	}
	work2 := func(ctx context.Context, d *data) error {
		// t.Logf("work2 job %s", d.name)
		if d.x == 1 {
			d.x = 2
			return nil
		} else {
			return errors.New("x != 1")
		}
	}
	_, err := goroutine_util.RunPipeline(ctx, jobs, false, work1, work2)
	if e := <-err; e != nil {
		t.Error("concurrent: unexpected error", e)
	}
	for _, v := range jobs {
		if v.x != 2 {
			t.Errorf("concurrent: x != 2 %d", v.x)
		}
	}
}

func TestRunPipelineErr(t *testing.T) {
	type data struct {
		name string
		x    int
	}
	ctx := context.Background()
	jobs := []*data{{"job1", 0}, {"job2", 0}}
	perr := errors.New("expected error")
	work1 := func(ctx context.Context, d *data) error {
		// t.Logf("work1 job %s", d.name)
		if d.x == 0 {
			d.x = 1
			return nil
		} else {
			return errors.New("x != 0")
		}
	}
	work2 := func(ctx context.Context, d *data) error {
		// t.Logf("work2 job %s", d.name)
		return perr
	}
	_, err := goroutine_util.RunPipeline(ctx, jobs, true, work1, work2)
	e := <-err
	if e == nil {
		t.Error("concurrent: err is nil")
	}
	if e != perr {
		t.Error("concurrent: err not equal", e, perr)
	}
}

func TestRunPipelinePanic(t *testing.T) {
	type data struct {
		name string
		x    int
	}
	ctx := context.Background()
	jobs := []*data{{"job1", 0}, {"job2", 0}}
	perr := errors.New("expected error")
	work1 := func(ctx context.Context, d *data) error {
		return nil
	}
	work2 := func(ctx context.Context, d *data) error {
		panic(perr)
	}
	_, err := goroutine_util.RunPipeline(ctx, jobs, true, work1, work2)
	e := <-err
	if e == nil {
		t.Error("concurrent: err is nil")
	}
	if err != nil && !strings.Contains(e.Error(), perr.Error()) {
		t.Error("concurrent: unexpected error", err)
	}
}

func TestNewPipelineRunner(t *testing.T) {
	ctx := context.Background()
	step1 := func(ctx context.Context, t *int) bool {
		if *t == 0 {
			*t = 1
			return true
		}
		return false
	}
	step2 := func(ctx context.Context, t *int) bool {
		if *t == 1 {
			*t = 2
			return true
		}
		return false
	}
	step3 := func(ctx context.Context, t *int) bool {
		if *t == 2 {
			*t = 3
			return true
		}
		return false
	}
	push, successCh, endPush := goroutine_util.NewPipelineRunner(ctx, step1, step2, step3)

	x := 0
	push(&x)

	select {
	case n := <-successCh:
		if *n != 3 {
			t.Error("concurrent: x is unexpected", *n)
		}
	case <-time.NewTimer(3 * time.Second).C:
		t.Error("concurrent: timeout")
	}

	push(&x)
	select {
	case <-successCh:
		t.Error("concurrent: unexpected")
	default:
		if x != 3 {
			t.Error("concurrent: unexpected")
		}
	}

	endPush()
	x = 0
	push(&x)
	if x != 0 {
		t.Error("concurrent: unexpected")
	}

	ctx, cancel := context.WithCancel(ctx)
	push, successCh, endPush = goroutine_util.NewPipelineRunner(ctx, step1, step2, step3)
	time.Sleep(time.Second)
	cancel()
	push(&x)
	select {
	case _, ok := <-successCh:
		if ok {
			t.Error("concurrent: unexpected")
		}
	case <-time.NewTimer(3 * time.Second).C:
	}
}

func TestListenChan(t *testing.T) {
	err1 := make(chan error, 1)
	err2 := make(chan error, 1)
	err3 := make(chan error, 1)
	errChans := []<-chan error{
		err1,
		err2,
		err3,
	}

	err := errors.New("expected error")
	go func() {
		time.Sleep(time.Second)
		err1 <- err
	}()
	if err != <-goroutine_util.ListenChan(errChans...) {
		t.Error("concurrent: err not equal")
		return
	}

	go func() {
		time.Sleep(time.Second)
		close(err1)
		close(err2)
		close(err3)
	}()
	if nil != <-goroutine_util.ListenChan(errChans...) {
		t.Error("concurrent: err is not nil")
	}
}

func TestRunPeriodically(t *testing.T) {
	run := goroutine_util.RunPeriodically(time.Second)
	var now time.Time
	run(func() {
		time.Sleep(time.Second)
		now = time.Now()
	})
	run(func() {
		if time.Since(now) < time.Second {
			t.Error("concurrent: unexpected time", time.Now())
		}
		time.Sleep(time.Second)
		now = time.Now()
	})
	run(func() {
		if time.Since(now) < time.Second {
			t.Error("concurrent: unexpected time", time.Now())
		}
		now = time.Now()
	})
	run(func() {
		if time.Since(now) < time.Second {
			t.Error("concurrent: unexpected time", time.Now())
		}
	})
}

func TestNewRunner(t *testing.T) {
	tests := []struct {
		name  string
		times int
		fn    func()
	}{
		{"正常运行，全不阻塞", 50, func() { testNewRunner(t, 0, false, true) }},
		{"正常运行，wait 阻塞", 50, func() { testNewRunner(t, 0, false, false) }},
		{"正常运行，add 阻塞", 50, func() { testNewRunner(t, 0, true, true) }},
		{"正常运行，add、wait 阻塞", 50, func() { testNewRunner(t, 0, true, false) }},
		{"正常运行，设置 max", 25, func() { testNewRunner(t, 100, false, true) }},
		{"正常运行，设置 max，add 阻塞", 25, func() { testNewRunner(t, 100, true, true) }},
		{"正常运行，设置 max，wait 阻塞", 25, func() { testNewRunner(t, 100, false, false) }},
		{"正常运行，设置 max，add、wait 阻塞", 25, func() { testNewRunner(t, 100, true, false) }},
		{"运行 error，不阻塞", 50, func() { testNewRunner4Error(t, 0, false, true) }},
		{"运行 error，wait 阻塞", 50, func() { testNewRunner4Error(t, 0, false, false) }},
		{"运行 error，add 阻塞", 50, func() { testNewRunner4Error(t, 0, true, true) }},
		{"运行 error，add、wait 阻塞", 50, func() { testNewRunner4Error(t, 0, true, false) }},
		{"运行 error，设置 max", 25, func() { testNewRunner4Error(t, 100, false, true) }},
		{"运行 error，设置 max，add 阻塞", 25, func() { testNewRunner4Error(t, 100, true, true) }},
		{"运行 error，设置 max，wait 阻塞", 25, func() { testNewRunner4Error(t, 100, false, false) }},
		{"运行 error，设置 max，add、wait 阻塞", 25, func() { testNewRunner4Error(t, 100, true, true) }},
		{"运行 panic，不阻塞", 50, func() { testNewRunner4Panic(t, 0, false, true) }},
		{"运行 panic，wait 阻塞", 50, func() { testNewRunner4Panic(t, 0, false, false) }},
		{"运行 panic，add 阻塞", 50, func() { testNewRunner4Panic(t, 0, true, true) }},
		{"运行 panic，add、wait 阻塞", 50, func() { testNewRunner4Panic(t, 0, true, false) }},
		{"运行 panic，设置 max", 25, func() { testNewRunner4Panic(t, 100, false, true) }},
		{"运行 panic，设置 max，add 阻塞", 25, func() { testNewRunner4Panic(t, 100, true, true) }},
		{"运行 panic，设置 max，wait 阻塞", 25, func() { testNewRunner4Panic(t, 100, false, false) }},
		{"运行 panic，设置 max，add、wait 阻塞", 25, func() { testNewRunner4Panic(t, 100, true, true) }},
		{"长时间运行，不阻塞", 50, func() { testNewRunner4LongTime(t, 0, false, true) }},
		{"长时间运行，wait 阻塞", 2, func() { testNewRunner4LongTime(t, 0, false, false) }},
		{"长时间运行，add 阻塞", 2, func() { testNewRunner4LongTime(t, 0, true, true) }},
		{"长时间运行，add、wait 阻塞", 2, func() { testNewRunner4LongTime(t, 0, true, false) }},
		{"长时间运行，设置 max", 2, func() { testNewRunner4LongTime(t, 100, false, true) }},
		{"长时间运行，设置 max，add 阻塞", 2, func() { testNewRunner4LongTime(t, 100, true, true) }},
		{"长时间运行，设置 max，wait 阻塞", 2, func() { testNewRunner4LongTime(t, 100, false, false) }},
		{"长时间运行，设置 max，add、wait 阻塞", 2, func() { testNewRunner4LongTime(t, 100, true, true) }},
		{"多次 wait，不阻塞", 50, func() { testNewRunner4ManyWait(t, 0, false, true) }},
		{"多次 wait，wait 阻塞", 50, func() { testNewRunner4ManyWait(t, 0, false, false) }},
		{"多次 wait，add 阻塞", 50, func() { testNewRunner4ManyWait(t, 0, true, true) }},
		{"多次 wait，add、wait 阻塞", 50, func() { testNewRunner4ManyWait(t, 0, true, false) }},
		{"多次 wait，设置 max", 25, func() { testNewRunner4ManyWait(t, 100, false, true) }},
		{"多次 wait，设置 max，add 阻塞", 25, func() { testNewRunner4ManyWait(t, 100, true, true) }},
		{"多次 wait，设置 max，wait 阻塞", 25, func() { testNewRunner4ManyWait(t, 100, false, false) }},
		{"多次 wait，设置 max，add、wait 阻塞", 25, func() { testNewRunner4ManyWait(t, 100, true, true) }},
		{"多次 add，不阻塞", 50, func() { testNewRunner4ManyAdd(t, 0, false, true) }},
		{"多次 add，wait 阻塞", 50, func() { testNewRunner4ManyAdd(t, 0, false, false) }},
		{"多次 add，add 阻塞", 50, func() { testNewRunner4ManyAdd(t, 0, true, true) }},
		{"多次 add，add、wait 阻塞", 50, func() { testNewRunner4ManyAdd(t, 0, true, false) }},
		{"多次 add，设置 max", 25, func() { testNewRunner4ManyAdd(t, 100, false, true) }},
		{"多次 add，设置 max，add 阻塞", 25, func() { testNewRunner4ManyAdd(t, 100, true, true) }},
		{"多次 add，设置 max，wait 阻塞", 25, func() { testNewRunner4ManyAdd(t, 100, false, false) }},
		{"多次 add，设置 max，add、wait 阻塞", 25, func() { testNewRunner4ManyAdd(t, 100, true, true) }},
		{"随机 add 阻塞，不阻塞", 50, func() { testNewRunner4RandomAddBlock(t, 0, true) }},
		{"随机 add 阻塞，wait 阻塞", 50, func() { testNewRunner4RandomAddBlock(t, 0, false) }},
		{"随机 add 阻塞，add、wait 阻塞", 50, func() { testNewRunner4RandomAddBlock(t, 0, false) }},
		{"随机 add 阻塞，设置 max", 25, func() { testNewRunner4RandomAddBlock(t, 100, true) }},
		{"随机 add 阻塞，设置 max，wait 阻塞", 25, func() { testNewRunner4RandomAddBlock(t, 100, false) }},
		{"wait fast exit，不阻塞", 50, func() { testNewRunner4WaitFastExit(t, 0) }},
		{"wait fast exit，设置 max", 25, func() { testNewRunner4WaitFastExit(t, 100) }},
		{"add after wait，不阻塞", 50, func() { testNewRunner4AddAfterWait(t, 0) }},
		{"add after wait，设置 max", 25, func() { testNewRunner4AddAfterWait(t, 100) }},
		{"ctx cancel，不阻塞", 50, func() { testNewRunner4CtxCancel(t, 0, false, true) }},
		{"ctx cancel，wait 阻塞", 50, func() { testNewRunner4CtxCancel(t, 0, false, false) }},
		{"ctx cancel，add 阻塞", 50, func() { testNewRunner4CtxCancel(t, 0, true, true) }},
		{"ctx cancel，add、wait 阻塞", 50, func() { testNewRunner4CtxCancel(t, 0, true, false) }},
		{"ctx cancel，设置 max", 25, func() { testNewRunner4CtxCancel(t, 100, false, true) }},
		{"ctx cancel，设置 max，add 阻塞", 25, func() { testNewRunner4CtxCancel(t, 100, true, true) }},
		{"ctx cancel，设置 max，wait 阻塞", 25, func() { testNewRunner4CtxCancel(t, 100, false, false) }},
		{"ctx cancel，设置 max，add、wait 阻塞", 25, func() { testNewRunner4CtxCancel(t, 100, true, true) }},
	}
	fn := func(name string, total int, f func()) {
		if t.Failed() {
			return
		}
		now := time.Now()
		fmt.Printf("%s: %d/0", name, total)
		for i := 0; i < total; i++ {
			f()
			if t.Failed() {
				return
			}
			fmt.Printf(strings.Repeat("\b", len(strconv.Itoa(i)))+"%v", i+1)
		}
		fmt.Printf(" done %v\n", time.Since(now))
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn(tt.name, tt.times, tt.fn)
		})
	}
}

func testNewRunner(t *testing.T, max int, addB, waitB bool) {
	type data struct {
		x int32
	}
	ctx := context.Background()
	count := int32(0)
	add, wait := goroutine_util.NewRunner(ctx, max, func(ctx context.Context, t *data) error {
		time.Sleep(time.Millisecond * 100)
		atomic.AddInt32(&count, t.x)
		return nil
	})
	var err error
	actualCount := int32(0)
	for i := int32(0); i < 1000; i++ {
		n := rand.Int31n(100)
		err = add(&data{n}, addB)
		actualCount += n
		if err != nil {
			break
		}
	}
	if err != nil {
		t.Errorf("unexpected error: want nil but got %v", err)
		return
	}
	err = wait(waitB)
	if err != nil {
		t.Errorf("unexpected error: want nil but got %v", err)
	}
	if count != actualCount {
		t.Errorf("unexpected count: got %v, want %v", count, actualCount)
	}
}

func testNewRunner4Error(t *testing.T, max int, addB, waitB bool) {
	type data struct {
		x, y int32
	}
	ctx := context.Background()
	expectedErr := errors.New("expected error")
	flag := rand.Int31n(1000)
	count := int32(0)
	add, wait := goroutine_util.NewRunner(ctx, max, func(ctx context.Context, t *data) error {
		time.Sleep(time.Millisecond * 100)
		if t.x == flag {
			return expectedErr
		}
		atomic.AddInt32(&count, t.y)
		return nil
	})
	var err error
	expectedCount := int32(0)
	for i := int32(0); i < 1000; i++ {
		n := rand.Int31n(100)
		err = add(&data{i, n}, addB)
		expectedCount += n
		if err != nil {
			break
		}
	}
	err = wait(waitB)
	if err != expectedErr {
		t.Errorf("unexpected error: got %v, want %v", err, expectedErr)
	}
	if count > expectedCount {
		t.Errorf("unexpected count: got %v, want %v", count, expectedCount)
	}
}

func testNewRunner4Panic(t *testing.T, max int, addB, waitB bool) {
	type data struct {
		x, y int32
	}
	ctx := context.Background()
	expectedErr := errors.New("expected error")
	flag := rand.Int31n(1000)
	count := int32(0)
	add, wait := goroutine_util.NewRunner(ctx, max, func(ctx context.Context, t *data) error {
		time.Sleep(time.Millisecond * 100)
		if t.x == flag {
			panic(expectedErr)
		}
		atomic.AddInt32(&count, t.y)
		return nil
	})
	var err error
	expectedCount := int32(0)
	for i := int32(0); i < 1000; i++ {
		n := rand.Int31n(100)
		err = add(&data{i, n}, addB)
		expectedCount += n
		if err != nil {
			break
		}
	}
	err = wait(waitB)
	if err == nil || errors.Is(err, expectedErr) {
		t.Errorf("unexpected error: got %v, want %v", err, expectedErr)
	}
	if count > expectedCount {
		t.Errorf("unexpected count: got %v, want %v", count, expectedCount)
	}
}

func testNewRunner4LongTime(t *testing.T, max int, addB, waitB bool) {
	type data struct {
		x, y int32
	}
	ctx := context.Background()
	count := int32(0)
	flag := rand.Int31n(1000)
	add, wait := goroutine_util.NewRunner(ctx, max, func(ctx context.Context, t *data) error {
		if t.y == flag {
			time.Sleep(time.Second * 10)
		} else {
			time.Sleep(time.Millisecond * 100)
		}
		atomic.AddInt32(&count, t.x)
		return nil
	})
	var err error
	actualCount := int32(0)
	for i := int32(0); i < 1000; i++ {
		n := rand.Int31n(100)
		err = add(&data{n, i}, addB)
		actualCount += n
		if err != nil {
			break
		}
	}
	if err != nil {
		t.Errorf("unexpected error: want nil but got %v", err)
		return
	}
	err = wait(waitB)
	if err != nil {
		t.Errorf("unexpected error: want nil but got %v", err)
	}
	if count != actualCount {
		t.Errorf("unexpected count: got %v, want %v", count, actualCount)
	}
}

func testNewRunner4ManyWait(t *testing.T, max int, addB, waitB bool) {
	type data struct {
		x int32
	}
	ctx := context.Background()
	count := int32(0)
	flag := rand.Int31n(200)
	expectedErr := errors.New("expected error")
	add, wait := goroutine_util.NewRunner(ctx, max, func(ctx context.Context, t *data) error {
		time.Sleep(time.Millisecond * 100)
		atomic.AddInt32(&count, t.x)
		if t.x == flag {
			return expectedErr
		}
		return nil
	})
	var err error
	actualCount := int32(0)
	for i := int32(0); i < 1000; i++ {
		n := rand.Int31n(100)
		err = add(&data{n}, addB)
		actualCount += n
		if err != nil {
			break
		}
	}
	if err != nil && err != expectedErr {
		t.Errorf("unexpected error: want %v, got %v", expectedErr, err)
		return
	}
	wg := sync.WaitGroup{}
	errM := sync.Map{}
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errM.Store(i, wait(waitB))
		}(i)
	}
	err = wait(waitB)
	if err != nil && err != expectedErr {
		t.Errorf("unexpected error: want %v, got %v", expectedErr, err)
	}
	if count > actualCount {
		t.Errorf("unexpected count: got %v, want %v", count, actualCount)
	}
	wg.Wait()
	errM.Range(func(k, v any) bool {
		if v != err {
			t.Errorf("unexpected error: got %v, want %v", v, err)
		}
		return true
	})
}

func testNewRunner4ManyAdd(t *testing.T, max int, addB, waitB bool) {
	type data struct {
		x int32
	}
	ctx := context.Background()
	count := int32(0)
	add, wait := goroutine_util.NewRunner(ctx, max, func(ctx context.Context, t *data) error {
		time.Sleep(time.Millisecond * 100)
		atomic.AddInt32(&count, t.x)
		return nil
	})
	actualCount := int32(0)
	for i := int32(0); i < 1000; i++ {
		go func() {
			n := rand.Int31n(100)
			_ = add(&data{n}, addB)
			atomic.AddInt32(&actualCount, n)
		}()
	}
	time.Sleep(time.Millisecond * 100)
	err := wait(waitB)
	if err != nil {
		t.Errorf("unexpected error: want nil but got %v", err)
	}
	if count != actualCount {
		t.Errorf("unexpected count: got %v, want %v", count, actualCount)
	}
}

func testNewRunner4RandomAddBlock(t *testing.T, max int, waitB bool) {
	type data struct {
		x int32
	}
	ctx := context.Background()
	count := int32(0)
	add, wait := goroutine_util.NewRunner(ctx, max, func(ctx context.Context, t *data) error {
		time.Sleep(time.Millisecond * 100)
		atomic.AddInt32(&count, t.x)
		return nil
	})
	actualCount := int32(0)
	for i := int32(0); i < 1000; i++ {
		go func() {
			addB := rand.Int31n(2) == 1
			n := rand.Int31n(100)
			_ = add(&data{n}, addB)
			atomic.AddInt32(&actualCount, n)
		}()
	}
	time.Sleep(time.Millisecond * 100)
	err := wait(waitB)
	if err != nil {
		t.Errorf("unexpected error: want nil, got %v", err)
	}
	if count != actualCount {
		t.Errorf("unexpected count: got %v, want %v", count, actualCount)
	}
}

type testCtx struct {
	e atomic.Value
	c chan struct{}
}

func (t *testCtx) Err() error {
	err, _ := t.e.Load().(error)
	return err
}

func (t *testCtx) Done() <-chan struct{} {
	return t.c
}

func (*testCtx) Deadline() (time.Time, bool) {
	return time.Time{}, false
}

func (*testCtx) Value(any) any {
	return nil
}

func testNewRunner4CtxCancel(t *testing.T, max int, addB, waitB bool) {
	type data struct {
		x int32
	}
	done := make(chan struct{})
	once := sync.Once{}
	expectedErr := errors.New("expected error")
	tc := &testCtx{
		c: done,
	}
	var ctx context.Context = tc
	ctx, cancel := context.WithCancel(ctx)
	cancel = func() {
		once.Do(func() {
			tc.e.Store(expectedErr)
			close(done)
			cancel()
		})
	}
	count := int32(0)
	flag := rand.Int31n(100)
	add, wait := goroutine_util.NewRunner(ctx, max, func(ctx context.Context, t *data) error {
		time.Sleep(time.Millisecond * 100)
		atomic.AddInt32(&count, t.x)
		return nil
	})
	actualCount := int32(0)
	wg := sync.WaitGroup{}
	for i := int32(0); i < 1000; i++ {
		wg.Add(1)
		go func() {
			n := rand.Int31n(100)
			if n == flag {
				cancel()
			}
			err := add(&data{n}, addB)
			wg.Done()
			atomic.AddInt32(&actualCount, n)
			if err != nil && ctx.Err() != expectedErr {
				t.Errorf("unexpected error: got %v, want %v", err, expectedErr)
			}
		}()
	}
	wg.Wait()
	err := wait(waitB)
	if err != nil && ctx.Err() != expectedErr {
		t.Errorf("unexpected error: want %v, got %v", expectedErr, err)
	}
	if count > actualCount {
		t.Errorf("unexpected count: got %v, want %v", count, actualCount)
	}
}

func testNewRunner4WaitFastExit(t *testing.T, max int) {
	type data struct {
		x int32
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	expectedErr := errors.New("expected error")
	count := int32(0)
	remain := int32(1000)
	expectedCount := int32(0)
	add, wait := goroutine_util.NewRunner(ctx, max, func(ctx context.Context, t *data) error {
		time.Sleep(time.Millisecond * 100)
		atomic.AddInt32(&count, t.x)
		atomic.AddInt32(&remain, -1)
		return nil
	})
	flag := 50 + rand.Int31n(50)
	wg := sync.WaitGroup{}
	for i := int32(0); i < 1000; i++ {
		wg.Add(1)
		go func() {
			n := rand.Int31n(100)
			atomic.AddInt32(&expectedCount, n)
			if n == flag {
				cancel(expectedErr)
			}
			addB := rand.Int31n(2) == 1
			err := add(&data{n}, addB)
			wg.Done()
			if err != nil && context.Cause(ctx) != expectedErr {
				t.Errorf("unexpected error: got %v, want %v", err, expectedErr)
			}
		}()
	}

	wg.Wait()
	err := wait(true)
	if atomic.LoadInt32(&remain) == 0 {
		t.Errorf("unexpected remain: got %v, want 0", remain)
	}
	if err != nil && context.Cause(ctx) != expectedErr {
		t.Errorf("unexpected error: want %v, got %v", expectedErr, err)
	}
	if count >= expectedCount {
		t.Errorf("unexpected count: got %v, want %v", count, expectedCount)
	}
}

func testNewRunner4AddAfterWait(t *testing.T, max int) {
	type data struct{}
	ctx := context.Background()
	add, wait := goroutine_util.NewRunner(ctx, max, func(ctx context.Context, t *data) error {
		time.Sleep(time.Millisecond * 100)
		return nil
	})
	flag := 90 + rand.Int31n(10)
	for i := int32(0); i < 1000; i++ {
		go func() {
			if rand.Int31n(100) == flag {
				_ = wait(rand.Int31n(2) == 1)
			}
		}()
		go func() {
			defer func() {
				p := recover()
				if p != nil && p != goroutine_util.ErrCallAddAfterWait {
					t.Errorf("unexpected recoverd value: want %v, got %v", goroutine_util.ErrCallAddAfterWait, p)
				}
			}()
			_ = add(&data{}, rand.Int31n(2) == 1)
		}()
	}
}
