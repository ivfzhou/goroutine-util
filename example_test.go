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

	goroutine_util "gitee.com/ivfzhou/goroutine-util"
)

func ExampleRunConcurrently() {
	ctx := context.Background()
	var order any
	work1 := func(ctx context.Context) error {
		// op order
		order = nil
		return nil
	}

	var stock any
	work2 := func(ctx context.Context) error {
		// op stock
		stock = nil
		return nil
	}
	err := goroutine_util.RunConcurrently(ctx, work1, work2)(false)
	// check err
	if err != nil {
		return
	}

	// do your want
	_ = order
	_ = stock
}

func ExampleRunSequentially() {
	ctx := context.Background()
	first := func(context.Context) error { return nil }
	then := func(context.Context) error { return nil }
	last := func(context.Context) error { return nil }
	err := goroutine_util.RunSequentially(ctx, first, then, last)
	if err != nil {
		// return err
	}
}

func ExampleNewRunner() {
	type product struct{}
	ctx := context.Background()
	op := func(ctx context.Context, data *product) error {
		// 要处理的任务逻辑。
		return nil
	}

	add, wait := goroutine_util.NewRunner[*product](ctx, 12, op)

	// 将任务要处理的数据传递给任务处理逻辑 op。
	var projects []*product
	for _, v := range projects {
		if err := add(v, true); err != nil {
			// 发生错误可能提前预知。
		}
	}

	// 等待所有数据处理完毕。
	if err := wait(true); err != nil {
		// 处理 op 返回的错误。
	}
}

func ExampleRunPipeline() {
	type data struct{}
	ctx := context.Background()

	jobs := []*data{{}, {}}
	work1 := func(ctx context.Context, d *data) error { return nil }
	work2 := func(ctx context.Context, d *data) error { return nil }

	succCh, errCh := goroutine_util.RunPipeline(ctx, jobs, false, work1, work2)
	select {
	case <-succCh:
	case <-errCh:
		// return err
	}
}
