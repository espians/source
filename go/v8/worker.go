// Package v8 provides a minimalist binding to the V8 JavaScript engine.
//
// The LoadModule and LoadScript methods are not threadsafe. It is up to callers
// to ensure that they are not called concurrently on the same Worker.
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
//
// The various configuration options must be set before any of that Worker's
// methods are called. Once one of its methods has been called, the Worker will
// no longer pay any attention to changes in the configuration options.
type Worker struct {
	instance *instance
	mutex    sync.Mutex

	// GetModuleSource is used to get
	GetModuleSource func(url string) string

	// HandleSend handles messages received from js.send calls. If it is nil,
	// then an exception will be raised to the caller.
	HandleSend func(msg string)

	// HandleSendSync handles messages received from js.sendSync calls. Its
	// return value will be passed back to the caller in JavaScript. If
	// HandleSendSync is nil, then an exception will be raised to the caller.
	HandleSendSync func(msg string) (response string)

	// ResolveModuleURL
	ResolveModuleURL func(url string, context string) string
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
	cb := getInstance(id).cb
	if cb != nil {
		cb(C.GoString(msg))
	}
}

//export recvSyncCb
func recvSyncCb(msg *C.char, id int32) *C.char {
	syncCb := getInstance(id).syncCb
	var resp string
	if syncCb == nil {
		resp = "v8: Worker.HandleSendSync is nil"
	} else {
		resp = syncCb(C.GoString(msg))
	}
	return C.CString(resp)
}

// Free resources associated with the underlying worker and V8 Isolate.
func (w *Worker) dispose() {
	mutex.Lock()
	delete(registry, w.instance.id)
	mutex.Unlock()
	C.worker_dispose(w.instance.worker)
}

// Convert the last exception into a Go value.
func (w *Worker) getError() error {
	err := C.worker_last_exception(w.instance.worker)
	defer C.free(unsafe.Pointer(err))
	return errors.New(C.GoString(err))
}

// Initialise the underlying JavaScript VM instance.
func (w *Worker) init() {
	if w.instance != nil {
		return
	}

	mutex.Lock()
	nextID++
	i := &instance{
		cb:     w.HandleSend,
		id:     nextID,
		syncCb: w.HandleSendSync,
	}
	registry[nextID] = i
	mutex.Unlock()

	once.Do(func() {
		C.v8_init()
	})

	i.worker = C.worker_new(C.int(i.id))
	w.instance = i

	runtime.SetFinalizer(w, func(w *Worker) {
		w.dispose()
	})
}

// LoadModule loads and executes JavaScript ES Module code with the given
// filename and source code. LoadModule is not threadsafe.
func (w *Worker) LoadModule(url string, source string) error {
	w.mutex.Lock()
	w.init()
	w.mutex.Unlock()

	urlStr := C.CString(url)
	sourceStr := C.CString(source)
	defer C.free(unsafe.Pointer(urlStr))
	defer C.free(unsafe.Pointer(sourceStr))

	r := C.worker_load_module(w.instance.worker, urlStr, sourceStr)
	if r != 0 {
		return w.getError()
	}
	return nil
}

// LoadScript loads and executes JavaScript code with the given filename and
// source code. LoadScript is not threadsafe.
func (w *Worker) LoadScript(filename string, source string) error {
	w.mutex.Lock()
	w.init()
	w.mutex.Unlock()

	filenameStr := C.CString(filename)
	sourceStr := C.CString(source)
	defer C.free(unsafe.Pointer(filenameStr))
	defer C.free(unsafe.Pointer(sourceStr))

	r := C.worker_load_script(w.instance.worker, filenameStr, sourceStr)
	if r != 0 {
		return w.getError()
	}
	return nil
}

// Send a message, calling the $recv callback in JavaScript.
func (w *Worker) Send(msg string) error {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	w.init()
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

	w.init()
	msgStr := C.CString(msg)
	defer C.free(unsafe.Pointer(msgStr))

	resp := C.worker_send_sync(w.instance.worker, msgStr)
	defer C.free(unsafe.Pointer(resp))

	return C.GoString(resp)
}

// Terminate instructs the underlying JavaScript VM to stop its current thread
// of execution. The instruction will cause the VM to stop at the next available
// opportunity.
func (w *Worker) Terminate() {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	// Don't bother if we haven't yet been initialised.
	if w.instance != nil {
		C.worker_terminate_execution(w.instance.worker)
	}
}
