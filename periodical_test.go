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
	"math/rand"
	"sync"
	"testing"
	"time"

	gu "gitee.com/ivfzhou/goroutine-util"
)

func TestRunPeriodically(t *testing.T) {
	t.Run("正常运行", func(t *testing.T) {
		for i := 0; i < 20; i++ {
			period := time.Millisecond * time.Duration(rand.Intn(100)+100)
			run := gu.RunPeriodically(period)
			var now time.Time
			wg := sync.WaitGroup{}
			wg.Add(3)
			run(func() { now = time.Now() })
			go run(func() {
				defer wg.Done()
				if time.Since(now) < period {
					t.Errorf("unexpected time: %v", now)
				}
				now = time.Now()
			})
			go run(func() {
				defer wg.Done()
				if time.Since(now) < period {
					t.Errorf("unexpected time: %v", now)
				}
				now = time.Now()
			})
			go run(func() {
				defer wg.Done()
				if time.Since(now) < period {
					t.Errorf("unexpected time: %v", now)
				}
				now = time.Now()
			})
			wg.Wait()
		}
	})

	t.Run("发生恐慌", func(t *testing.T) {
		for i := 0; i < 20; i++ {
			period := time.Millisecond * time.Duration(rand.Intn(100)+100)
			run := gu.RunPeriodically(period)
			var now time.Time
			occurPanicIndex := rand.Intn(3)
			wg := sync.WaitGroup{}
			wg.Add(3)
			run(func() { now = time.Now() })
			go run(func() {
				defer wg.Done()
				defer func() { recover() }()
				if time.Since(now) < period {
					t.Errorf("unexpected time: %v", now)
				}
				now = time.Now()
				if occurPanicIndex == 0 {
					panic("")
				}
			})
			go run(func() {
				defer wg.Done()
				defer func() { recover() }()
				if time.Since(now) < period {
					t.Errorf("unexpected time: %v", now)
				}
				now = time.Now()
				if occurPanicIndex == 1 {
					panic("")
				}
			})
			go run(func() {
				defer wg.Done()
				defer func() { recover() }()
				if time.Since(now) < period {
					t.Errorf("unexpected time: %v", now)
				}
				now = time.Now()
				if occurPanicIndex == 2 {
					panic("")
				}
			})
			wg.Wait()
		}
	})
}
