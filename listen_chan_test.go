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
	"errors"
	"math/rand"
	"runtime"
	"testing"
	"time"

	gu "gitee.com/ivfzhou/goroutine-util"
)

func TestListenChan(t *testing.T) {
	t.Run("正常运行", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			err1 := make(chan error, rand.Intn(2))
			err2 := make(chan error, rand.Intn(2))
			err3 := make(chan error, rand.Intn(2))
			errChans := []<-chan error{
				err1,
				err2,
				err3,
			}

			expectedErr := errors.New("expected error")
			go func() {
				time.Sleep(time.Millisecond * 100)
				switch rand.Intn(3) {
				case 0:
					err1 <- expectedErr
					close(err1)
					close(err2)
					close(err3)
				case 1:
					err2 <- expectedErr
					close(err2)
					close(err1)
					close(err3)
				case 2:
					err3 <- expectedErr
					close(err3)
					close(err1)
					close(err2)
				}
			}()

			ch := gu.ListenChan(errChans...)
			if err := <-ch; !errors.Is(err, expectedErr) {
				t.Errorf("unexpected error: want %v, got %v", expectedErr, err)
			}

			err, ok := <-ch
			if ok {
				t.Errorf("unexpected result: want false, got %v, value is %v", ok, err)
			}
		}
	})

	t.Run("通道都关闭未发送值", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			err1 := make(chan error, rand.Intn(2))
			err2 := make(chan error, rand.Intn(2))
			err3 := make(chan error, rand.Intn(2))
			errChans := []<-chan error{
				err1,
				err2,
				err3,
			}

			go func() {
				time.Sleep(time.Millisecond * 100)
				close(err1)
				close(err2)
				close(err3)
			}()
			ch := gu.ListenChan(errChans...)
			if err := <-ch; err != nil {
				t.Errorf("unexpected error: want nil, got %v", err)
			}
			err, ok := <-ch
			if ok {
				t.Errorf("unexpected result: want false, got %v, value is %v", ok, err)
			}
		}
	})

	t.Run("包含空通道", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			err1 := make(chan error, rand.Intn(2))
			err2 := make(chan error, rand.Intn(2))
			err3 := make(chan error, rand.Intn(2))
			errChans := []<-chan error{
				err1,
				err2,
				nil,
				err3,
			}

			go func() {
				time.Sleep(time.Millisecond * 100)
				close(err1)
				close(err2)
				close(err3)
			}()
			ch := gu.ListenChan(errChans...)
			if err := <-ch; err != nil {
				t.Errorf("unexpected error: want nil, got %v", err)
			}
			err, ok := <-ch
			if ok {
				t.Errorf("unexpected result: want false, got %v, value is %v", ok, err)
			}
		}
	})

	t.Run("多个通道发送了值", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			err1 := make(chan error, rand.Intn(2))
			err2 := make(chan error, rand.Intn(2))
			err3 := make(chan error, rand.Intn(2))
			errChans := []<-chan error{
				err1,
				err2,
				err3,
			}

			expectedErr := errors.New("expected error")
			go func() {
				time.Sleep(time.Millisecond * 100)
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

	t.Run("监听大量通道", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			errChans := make([]<-chan error, 500*(rand.Intn(5)+1)+10)
			sendIndex := rand.Intn(len(errChans))
			expectedErr := errors.New("expected error")
			for i := range errChans {
				ch := make(chan error, rand.Intn(2))
				errChans[i] = ch
				go func(i int, c chan<- error) {
					if i == sendIndex {
						c <- expectedErr
					}
					close(c)
				}(i, ch)
			}

			mergedCh := gu.ListenChan(errChans...)
			result, ok := <-mergedCh
			if !ok {
				t.Errorf("unexpected result: want true, got %v", ok)
			}
			if result != expectedErr {
				t.Errorf("unexpected result: want %v, got %v", expectedErr, result)
			}
			result, ok = <-mergedCh
			if ok {
				t.Errorf("unexpected result: want false, got %v", ok)
			}
			if result != nil {
				t.Errorf("unexpected result: want nil, got %v", result)
			}

			errChans = nil
			runtime.GC()
		}
	})
}
