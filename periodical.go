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

package goroutine_util

import "time"

// RunPeriodically 运行 fn，每个 fn 之间至少间隔 period 时间。前一个 fn 运行完毕到下一个 fn 开始运行之间的间隔时间。
//
// period：每个 fn 运行至少间隔时间。
//
// 注意：若 period 为负数将会触发恐慌。
func RunPeriodically(period time.Duration) (run func(fn func())) {
	if period <= 0 {
		panic("period must be positive")
	}

	lastAccess := time.Time{}
	run = func(fn func()) {
		time.Sleep(period - time.Since(lastAccess))
		fn()
		lastAccess = time.Now()
	}

	return
}
