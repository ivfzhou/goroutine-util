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
	"time"

	gu "gitee.com/ivfzhou/goroutine-util"
)

type ctxCancelWithError struct {
	context.Context
	err gu.AtomicError
}

func (c *ctxCancelWithError) Deadline() (deadline time.Time, ok bool) {
	return c.Context.Deadline()
}

func (c *ctxCancelWithError) Done() <-chan struct{} {
	return c.Context.Done()
}

func (c *ctxCancelWithError) Err() error {
	return c.err.Get()
}

func (c *ctxCancelWithError) Value(key any) any {
	return c.Context.Value(key)
}

func newCtxCancelWithError() (context.Context, context.CancelCauseFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	c := &ctxCancelWithError{Context: ctx}
	return c, func(cause error) {
		c.err.Set(cause)
		cancel()
	}
}
