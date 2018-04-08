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
var nextID = 0
var once sync.Once
var workers = make(map[int]*worker)

// Callback handles messages received from $send calls from JavaScript.
type Callback func(msg string)

// SyncCallback handles messages received from $sendSync calls from JavaScript
// and passes the returned string back to JavaScript.
type SyncCallback func(msg string) string

// Internal worker struct which is stored in the workers map using the weakref
// pattern.
type worker struct {
	cb      Callback
	cWorker *C.worker
	id      int
	syncCb  SyncCallback
}

// Worker represents a single JavaScript VM instance.
type Worker struct {
	*worker
}

// Version returns the V8 version, e.g. "6.6.346.19".
func Version() string {
	return C.GoString(C.worker_version())
}

// We use this indirection to get at active workers as we can't safely pass
// pointers to Go objects to C.
func getWorker(id int) *worker {
	mutex.Lock()
	defer mutex.Unlock()
	return workers[id]
}

//export recvCb
func recvCb(msg *C.char, id int) {
	getWorker(id).cb(C.GoString(msg))
}

//export recvSyncCb
func recvSyncCb(msg *C.char, id int) *C.char {
	return C.CString(getWorker(id).syncCb(C.GoString(msg)))
}

// New initialises a new JavaScript VM instance.
func New(cb Callback, syncCb SyncCallback) *Worker {
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

	wrapper := &Worker{w}
	runtime.SetFinalizer(wrapper, func(w *Worker) {
		w.dispose()
	})
	return wrapper
}

// Free resources associated with the worker and the underlying V8 Isolate.
func (w *Worker) dispose() {
	mutex.Lock()
	delete(workers, w.worker.id)
	mutex.Unlock()
	C.worker_dispose(w.worker.cWorker)
}

// Load and execute JavaScript code with the given filename and source code.
func (w *Worker) Load(filename string, source string) error {
	filenameStr := C.CString(filename)
	sourceStr := C.CString(source)
	defer C.free(unsafe.Pointer(filenameStr))
	defer C.free(unsafe.Pointer(sourceStr))

	r := C.worker_load(w.worker.cWorker, filenameStr, sourceStr)
	if r != 0 {
		err := C.GoString(C.worker_last_exception(w.worker.cWorker))
		return errors.New(err)
	}
	return nil
}

// Send a message, calling the $recv callback in JavaScript.
func (w *Worker) Send(msg string) error {
	msgStr := C.CString(msg)
	defer C.free(unsafe.Pointer(msgStr))

	r := C.worker_send(w.worker.cWorker, msgStr)
	if r != 0 {
		err := C.GoString(C.worker_last_exception(w.worker.cWorker))
		return errors.New(err)
	}
	return nil
}

// SendSync sends a message, calling the $recvSync callback in JavaScript. The
// return value of that callback will be passed back to the worker.
func (w *Worker) SendSync(msg string) string {
	msgStr := C.CString(msg)
	defer C.free(unsafe.Pointer(msgStr))
	return C.GoString(C.worker_send_sync(w.worker.cWorker, msgStr))
}

// Terminate forcefully stops the current thread of JavaScript execution.
func (w *Worker) Terminate() {
	C.worker_terminate_execution(w.worker.cWorker)
}
