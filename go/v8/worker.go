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
	getModuleSource func(string) (string, error)
	handleSend      func(string) error
	handleSendSync  func(string) (string, error)
	id              int32
	worker          *C.worker
}

// Worker represents a single JavaScript VM instance.
//
// The various configuration options must be set before any of that Worker's
// methods are called. Once one of its methods has been called, the Worker will
// no longer pay any attention to changes in its config.
type Worker struct {
	instance *instance
	mutex    sync.Mutex

	// EnablePrint creates the debug $print function in the JavaScript global
	// scope.
	EnablePrint bool

	// GetModuleSource returns the source code when given the fully qualified
	// url of a module, or returns an error if it couldn't retrieve the source
	// code for some reason.
	GetModuleSource func(url string) (source string, err error)

	// HandleSend handles messages received from js.send calls. If it is nil,
	// then an exception will be raised to the caller.
	HandleSend func(msg string) error

	// HandleSendSync handles messages received from js.sendSync calls. Its
	// return value will be passed back to the caller in JavaScript. If
	// HandleSendSync is nil, then an exception will be raised to the caller.
	HandleSendSync func(msg string) (response string, err error)

	// ResolveModuleURL resolves the url of a module relative to the module it
	// was imported from and returns the fully qualified url of the module, or
	// an error if no such module could be found.
	ResolveModuleURL func(url string, importer string) (string, error)
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

//export getModuleSource
func getModuleSource(id int32, url *C.char) *C.char {
	source, err := getInstance(id).getModuleSource(C.GoString(url))
	if err != nil {
		panic(err)
	}
	return C.CString(source)
}

//export recvCb
func recvCb(id int32, msg *C.char) {
	cb := getInstance(id).handleSend
	if cb != nil {
		cb(C.GoString(msg))
	}
}

//export recvSyncCb
func recvSyncCb(id int32, msg *C.char) *C.char {
	cb := getInstance(id).handleSendSync
	var resp string
	if cb == nil {
		resp = "v8: Worker.HandleSendSync is nil"
	} else {
		resp, _ = cb(C.GoString(msg))
	}
	return C.CString(resp)
}

// Free resources associated with the underlying instance and V8 Isolate.
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
		getModuleSource: w.GetModuleSource,
		handleSend:      w.HandleSend,
		handleSendSync:  w.HandleSendSync,
		id:              nextID,
	}
	registry[nextID] = i
	mutex.Unlock()

	once.Do(func() {
		C.v8_init()
	})

	var enablePrint int32
	if w.EnablePrint {
		enablePrint = 1
	}

	i.worker = C.worker_init(C.int(i.id), C.int(enablePrint))
	w.instance = i

	runtime.SetFinalizer(w, func(w *Worker) {
		w.dispose()
	})
}

// LoadModule loads and executes ES Module code with the given url. LoadModule
// is not threadsafe.
func (w *Worker) LoadModule(url string) error {
	w.mutex.Lock()
	w.init()
	if w.instance.getModuleSource == nil {
		return errors.New("v8: GetModuleSource needs to be set before any methods are called")
	}
	w.mutex.Unlock()

	urlStr := C.CString(url)
	defer C.free(unsafe.Pointer(urlStr))

	r := C.worker_load_module(w.instance.worker, urlStr)
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
func (w *Worker) SendSync(msg string) (string, error) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	w.init()
	msgStr := C.CString(msg)
	defer C.free(unsafe.Pointer(msgStr))

	resp := C.worker_send_sync(w.instance.worker, msgStr)
	defer C.free(unsafe.Pointer(resp))

	return C.GoString(resp), nil
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

// TODO:
//
// Configure module resolution
// Fully fledged error values
// Raise exceptions in JS
// Return errors in Go
// Protect $functions -- perhaps in module -- perhaps make it configurable
// Handle async
// Set request/response IDs
