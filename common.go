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

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync/atomic"
)

// 代表错误信息是空。
var nilError = WrapError(errors.New(""))

// Error 错误对象。
type Error struct {
	err error
}

// AtomicError 原子性读取和设置错误信息。
type AtomicError struct {
	err atomic.Value
}

// WrapError 包裹错误信息。
func WrapError(err error) error {
	if err == nil {
		return nil
	}
	return &Error{err}
}

// Set 设置错误信息，除非 err 是空。返回真表示设置成功。
func (e *AtomicError) Set(err error) bool {
	if err == nil {
		err = nilError
	} else {
		err = WrapError(err)
	}
	return e.err.CompareAndSwap(nil, err)
}

// Get 获取内部错误信息。
func (e *AtomicError) Get() error {
	err, _ := e.err.Load().(error)
	if err == nil || err == nilError {
		return nil
	}
	return err.(*Error).err
}

// HasSet 是否设置了错误信息。
func (e *AtomicError) HasSet() bool {
	err, _ := e.err.Load().(error)
	return err != nil
}

// String 接口实现。
func (e *Error) String() string {
	if e.err == nil {
		return ""
	}
	return e.err.Error()
}

// Error 接口实现。
func (e *Error) Error() string {
	if e.err == nil {
		return ""
	}
	return e.err.Error()
}

// Unwrap 接口实现。
func (e *Error) Unwrap() error {
	return e.err
}

// 将恐慌信息以错误形式返回。并继承恐慌中的错误。
func wrapperPanic(p any) error {
	pe, ok := p.(error)
	if ok {
		return fmt.Errorf("panic: %w [recovered]\n%s\n", pe, getStackCallers())
	}
	return fmt.Errorf("panic: %v [recovered]\n%s\n", p, getStackCallers())
}

// 获取函数调用堆栈信息。
func getStackCallers() string {
	callers := make([]uintptr, 3*12)
	n := runtime.Callers(3, callers)
	callers = callers[:n]
	frames := runtime.CallersFrames(callers)
	callers = nil
	var (
		frame runtime.Frame
		more  bool
	)
	sb := &strings.Builder{}
	for {
		frame, more = frames.Next()
		_, _ = fmt.Fprintf(sb, "%s\n", frame.Function)
		_, _ = fmt.Fprintf(sb, "    %s:%v\n", frame.File, frame.Line)
		if !more {
			break
		}
	}
	return sb.String()
}
