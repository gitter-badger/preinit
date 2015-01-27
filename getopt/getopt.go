/*
	Package getopt provides util functions for get command options
*/

package getopt

/*

#include "setproctitle.h"

*/
import "C"
import (
	"os"
	"strings"
	"unsafe"
)

// C.spt_init1 defined in setproctitle.h

const (
	// These values must match the return values for spt_init1() used in C.
	HaveNone        = 0
	HaveNative      = 1
	HaveReplacement = 2
)

var (
	HaveSetProcTitle int
)

var OrigProcTitle string

func setproctitle_init() {
	Opts = NewOptParser(os.Args[1:])
	if len(OrigProcTitle) == 0 {
		OrigProcTitle = CleanArgLine(os.Args[0] + " " + Opts.String())
	}
	HaveSetProcTitle = int(C.spt_init1())

	if HaveSetProcTitle == HaveReplacement {
		newArgs := make([]string, len(os.Args))
		for i, s := range os.Args {
			// Use cgo to force go to make copies of the strings.
			cs := C.CString(s)
			newArgs[i] = C.GoString(cs)
			C.free(unsafe.Pointer(cs))
		}
		os.Args = newArgs

		env := os.Environ()
		for _, kv := range env {
			skv := strings.SplitN(kv, "=", 2)
			os.Setenv(skv[0], skv[1])
		}

		argc := C.int(len(os.Args))
		arg0 := C.CString(os.Args[0])
		defer C.free(unsafe.Pointer(arg0))

		C.spt_init2(argc, arg0)

		// Restore the original title.
		SetProcTitle(os.Args[0])
	}
}

func SetProcTitle(title string) {
	cs := C.CString(title)
	defer C.free(unsafe.Pointer(cs))
	C.spt_setproctitle(cs)
}

func SetProcTitlePrefix(prefix string) {
	title := prefix + OrigProcTitle
	cs := C.CString(title)
	defer C.free(unsafe.Pointer(cs))
	C.spt_setproctitle(cs)
}

func SetProcTitleSuffix(prefix string) {
	title := OrigProcTitle + prefix
	cs := C.CString(title)
	defer C.free(unsafe.Pointer(cs))
	C.spt_setproctitle(cs)
}

// end of SetProcTitle
//
func init() {
	setproctitle_init()
}

//
func UNUSED() {}
