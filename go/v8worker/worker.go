// Package v8worker provides a minimalist binding to the V8 JavaScript engine.
package v8worker

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
var workers = make(map[int32]*worker)

// Callback handles messages received from $send calls from JavaScript.
type Callback func(msg string)

// SyncCallback handles messages received from $sendSync calls from JavaScript.
// Its return value is passed back to JavaScript.
type SyncCallback func(msg string) string

// Internal worker struct which is stored in the workers map using the weakref
// pattern.
type worker struct {
	cb      Callback
	cWorker *C.worker
	id      int32
	syncCb  SyncCallback
}

// Instance represents a single JavaScript VM instance.
type Instance struct {
	sync.Mutex
	*worker
}

// Version returns the V8 version, e.g. "6.6.346.19".
func Version() string {
	return C.GoString(C.worker_version())
}

// We use this indirection to get at active workers as we can't safely pass
// pointers to Go objects to C.
func getWorker(id int32) *worker {
	mutex.Lock()
	defer mutex.Unlock()
	return workers[id]
}

//export recvCb
func recvCb(msg *C.char, id int32) {
	getWorker(id).cb(C.GoString(msg))
}

//export recvSyncCb
func recvSyncCb(msg *C.char, id int32) *C.char {
	return C.CString(getWorker(id).syncCb(C.GoString(msg)))
}

// New initialises a new JavaScript VM instance.
func New(cb Callback, syncCb SyncCallback) *Instance {
	mutex.Lock()
	nextID++
	w := &worker{
		cb:     cb,
		syncCb: syncCb,
		id:     nextID,
	}
	workers[nextID] = w
	mutex.Unlock()

	once.Do(func() {
		C.v8_init()
	})

	w.cWorker = C.worker_new(C.int(w.id))

	i := &Instance{worker: w}
	runtime.SetFinalizer(i, func(w *Instance) {
		w.dispose()
	})
	return i
}

// Free resources associated with the underlying worker and V8 Isolate.
func (i *Instance) dispose() {
	mutex.Lock()
	delete(workers, i.worker.id)
	mutex.Unlock()
	C.worker_dispose(i.worker.cWorker)
}

func (i *Instance) getError() error {
	err := C.worker_last_exception(i.worker.cWorker)
	defer C.free(unsafe.Pointer(err))
	return errors.New(C.GoString(err))
}

// Load and execute JavaScript code with the given filename and source code.
// Each Instance can only handle a single Load call at a time. It is up to the
// caller to ensure that multiple threads don't call Load on the same Instance
// at the same time.
func (i *Instance) Load(filename string, source string) error {
	filenameStr := C.CString(filename)
	sourceStr := C.CString(source)
	defer C.free(unsafe.Pointer(filenameStr))
	defer C.free(unsafe.Pointer(sourceStr))

	r := C.worker_load(i.worker.cWorker, filenameStr, sourceStr)
	if r != 0 {
		return i.getError()
	}
	return nil
}

// Send a message, calling the $recv callback in JavaScript.
func (i *Instance) Send(msg string) error {
	i.Lock()
	defer i.Unlock()

	msgStr := C.CString(msg)
	defer C.free(unsafe.Pointer(msgStr))

	r := C.worker_send(i.worker.cWorker, msgStr)
	if r != 0 {
		return i.getError()
	}
	return nil
}

// SendSync sends a message, calling the $recvSync callback in JavaScript. The
// return value of that callback will be passed back to the caller in Go.
func (i *Instance) SendSync(msg string) string {
	i.Lock()
	defer i.Unlock()

	msgStr := C.CString(msg)
	defer C.free(unsafe.Pointer(msgStr))

	resp := C.worker_send_sync(i.worker.cWorker, msgStr)
	defer C.free(unsafe.Pointer(resp))

	return C.GoString(resp)
}

// Terminate forcefully stops the current thread of JavaScript execution.
func (i *Instance) Terminate() {
	i.Lock()
	defer i.Unlock()

	C.worker_terminate_execution(i.worker.cWorker)
}
