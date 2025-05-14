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

import "context"

// RunSequentially 依次运行 fn，当有错误发生时停止后续 fn 运行。
//
// ctx：上下文，终止时，将导致后续 fn 不再运行，并返回 ctx.Err()。
//
// fn：要运行的函数，可以安全的添加空 fn。发生恐慌将被恢复，并以错误形式返回。
//
// error：返回的错误与 fn 返回的错误一致。
func RunSequentially(ctx context.Context, fn ...func(context.Context) error) error {
	// 没有 fn，直接返回不用运行。
	if len(fn) <= 0 {
		return nil
	}

	// 避免上下文空指针。
	nilCtx := false
	if ctx == nil {
		nilCtx = true
	} else { // 检测上下文是否已经终止。
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	// 依次运行 fn。
	var err error
	for _, f := range fn {
		if f == nil {
			continue
		}

		// 检测上下文是否终止。
		if !nilCtx {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
		}

		func() {
			defer func() {
				if p := recover(); p != nil {
					err = wrapperPanic(p)
				}
			}()
			err = f(ctx)
		}()
		if err != nil {
			return err
		}
	}

	return nil
}
