/*

1. log
2. args
3. daemon
4. signal
5. fdpass
6. children monitor

*/

/*
	https://code.google.com/p/log4go/
*/

package preinit

import (
	"github.com/wheelcomplex/preinit/log4go"
)

//// VARS ////

// debug level 0-8, default
var debugLevel int
