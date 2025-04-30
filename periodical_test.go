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
	"testing"
	"time"

	gu "gitee.com/ivfzhou/goroutine-util"
)

func TestRunPeriodically(t *testing.T) {
	for i := 0; i < 50; i++ {
		period := time.Millisecond * time.Duration(rand.Intn(1000)+100)
		run := gu.RunPeriodically(period)
		var now time.Time

		run(func() { now = time.Now() })

		run(func() {
			if time.Since(now) < period {
				t.Errorf("unexpected time: %v", now)
			}
		})

		run(func() {
			if time.Since(now) < 2*period {
				t.Errorf("unexpected time: %v", now)
			}
		})

		run(func() {
			if time.Since(now) < 3*period {
				t.Errorf("unexpected time: %v", now)
			}
		})
	}
}
