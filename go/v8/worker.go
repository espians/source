// Package v8 provides a minimalist binding to the V8 JavaScript engine.
package v8

/*
#cgo CXXFLAGS: -I${SRCDIR} -I${SRCDIR}/include -std=c++11
#cgo darwin,amd64 LDFLAGS: -L${SRCDIR}/lib/darwin.x64 -lv8_base -lv8_libplatform -lv8_libbase -lv8_libsampler -lv8_snapshot -ldl -pthread
#cgo linux,amd64 LDFLAGS: -L${SRCDIR}/lib/linux.x64 -lv8_base -lv8_libplatform -lv8_libbase -lv8_libsampler -lv8_snapshot -ldl -pthread
#include <stdlib.h>
#include "binding.h"
*/
import "C"

import (
	"errors"
	"runtime"
	"sync"
	"unsafe"
)

var mutex sync.Mutex
var nextID int32
var once sync.Once
var registry = make(map[int32]*instance)

// Internal struct which is stored in the registry map using the weakref
// pattern.
type instance struct {
	cb     func(string)
	id     int32
	syncCb func(string) string
	worker *C.worker
}

// Worker represents a single JavaScript VM instance.
type Worker struct {
	instance *instance
	mutex    sync.Mutex
}

// Version returns the V8 version, e.g. "6.6.346.19".
func Version() string {
	return C.GoString(C.worker_version())
}

// We use this indirection to get at active instances as we can't safely pass
// pointers to Go objects to C.
func getInstance(id int32) *instance {
	mutex.Lock()
	defer mutex.Unlock()
	return registry[id]
}

//export recvCb
func recvCb(msg *C.char, id int32) {
	getInstance(id).cb(C.GoString(msg))
}

//export recvSyncCb
func recvSyncCb(msg *C.char, id int32) *C.char {
	return C.CString(getInstance(id).syncCb(C.GoString(msg)))
}

// New initialises a new JavaScript VM instance. It takes two parameters
// defining callback handlers for messages sent from JavaScript.
//
// The first callback handles messages received from $send calls from
// JavaScript.
//
// The second callback handles messages received from $sendSync calls from
// JavaScript. Its return value will be passed back to the caller in JavaScript.
func New(cb func(string), syncCb func(string) string) *Worker {
	mutex.Lock()
	nextID++
	i := &instance{
		cb:     cb,
		id:     nextID,
		syncCb: syncCb,
	}
	registry[nextID] = i
	mutex.Unlock()

	once.Do(func() {
		C.v8_init()
	})

	i.worker = C.worker_new(C.int(i.id))

	w := &Worker{
		instance: i,
	}
	runtime.SetFinalizer(w, func(w *Worker) {
		w.dispose()
	})
	return w
}

// Free resources associated with the underlying worker and V8 Isolate.
func (w *Worker) dispose() {
	mutex.Lock()
	delete(registry, w.instance.id)
	mutex.Unlock()
	C.worker_dispose(w.instance.worker)
}

func (w *Worker) getError() error {
	err := C.worker_last_exception(w.instance.worker)
	defer C.free(unsafe.Pointer(err))
	return errors.New(C.GoString(err))
}

// Load and execute JavaScript code with the given filename and source code.
// Each Worker can only handle a single Load call at a time. It is up to the
// caller to ensure that multiple threads don't call Load on the same Worker at
// the same time.
func (w *Worker) Load(filename string, source string) error {
	filenameStr := C.CString(filename)
	sourceStr := C.CString(source)
	defer C.free(unsafe.Pointer(filenameStr))
	defer C.free(unsafe.Pointer(sourceStr))

	r := C.worker_load(w.instance.worker, filenameStr, sourceStr)
	if r != 0 {
		return w.getError()
	}
	return nil
}

// Send a message, calling the $recv callback in JavaScript.
func (w *Worker) Send(msg string) error {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	msgStr := C.CString(msg)
	defer C.free(unsafe.Pointer(msgStr))

	r := C.worker_send(w.instance.worker, msgStr)
	if r != 0 {
		return w.getError()
	}
	return nil
}

// SendSync sends a message, calling the $recvSync callback in JavaScript. The
// return value of that callback will be passed back to the caller in Go.
func (w *Worker) SendSync(msg string) string {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	msgStr := C.CString(msg)
	defer C.free(unsafe.Pointer(msgStr))

	resp := C.worker_send_sync(w.instance.worker, msgStr)
	defer C.free(unsafe.Pointer(resp))

	return C.GoString(resp)
}

// Terminate forcefully stops the current thread of JavaScript execution.
func (w *Worker) Terminate() {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	C.worker_terminate_execution(w.instance.worker)
}
