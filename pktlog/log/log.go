// Copyright (c) 2017 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.
//
// Copyright (c) 2009 The Go Authors. All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
//    * Redistributions of source code must retain the above copyright
// notice, this list of conditions and the following disclaimer.
//    * Redistributions in binary form must reproduce the above
// copyright notice, this list of conditions and the following disclaimer
// in the documentation and/or other materials provided with the
// distribution.
//    * Neither the name of Google Inc. nor the names of its
// contributors may be used to endorse or promote products derived from
// this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
// "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
// LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
// A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
// OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
// LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
// DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
// THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package log

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkt-cash/pktd/btcutil/er"
	"github.com/pkt-cash/pktd/pktconfig/version"
)

// Flags to modify Backend's behavior.
const (
	// Llongfile modifies the logger output to include full path and line number
	// of the logging callsite, e.g. /a/b/c/main.go:123.
	Llongfile uint32 = 1 << iota

	// Lshortfile modifies the logger output to include filename and line number
	// of the logging callsite, e.g. main.go:123.  Overrides Llongfile.
	Lshortfile

	Lcolor

	Llongdate
)

// Level is the level at which a logger is configured.  All messages sent
// to a level which is below the current level are filtered.
type Level uint32

// Level constants.
const (
	LevelTrace Level = iota
	LevelDebug
	LevelInfo
	LevelWarn
	LevelError
	LevelCritical
	LevelOff
	LevelInvalid
)

// levelStrs defines the human-readable names for each logging level.
var levelStrs = [...]string{"TRC", "DBG", "INF", "WRN", "ERR", "CRT", "OFF"}

// LevelFromString returns a level based on the input string s.  If the input
// can't be interpreted as a valid log level, the info level and false is
// returned.
func LevelFromString(s string) (l Level, ok bool) {
	switch strings.ToLower(s) {
	case "trace", "trc":
		return LevelTrace, true
	case "debug", "dbg":
		return LevelDebug, true
	case "info", "inf":
		return LevelInfo, true
	case "warn", "wrn":
		return LevelWarn, true
	case "error", "err":
		return LevelError, true
	case "critical", "crt":
		return LevelCritical, true
	case "off":
		return LevelOff, true
	default:
		return LevelInfo, false
	}
}

// SetLogLevels attempts to parse the specified debug level and set
// the levels accordingly.  An appropriate error is returned if anything is
// invalid.
func SetLogLevels(debugLevel string) er.R {
	// When the specified string doesn't have any delimters, treat it as
	// the log level for all subsystems.
	if !strings.Contains(debugLevel, ",") && !strings.Contains(debugLevel, "=") {
		if lvl, ok := LevelFromString(debugLevel); !ok {
			return er.Errorf("The specified debug level [%v] is invalid", debugLevel)
		} else {
			b.lock.Lock()
			defer b.lock.Unlock()
			b.lvl = lvl
		}
		return nil
	}

	// Split the specified string into subsystem/level pairs while detecting
	// issues and update the log levels accordingly.
	glvl := LevelInvalid
	m := make(map[string]Level)
	for _, logLevelPair := range strings.Split(debugLevel, ",") {
		if !strings.Contains(logLevelPair, "=") {
			if lvl, ok := LevelFromString(logLevelPair); !ok {
				return er.Errorf("The specified debug level [%v] is invalid", logLevelPair)
			} else {
				glvl = lvl
			}
			str := "The specified debug level contains an invalid " +
				"subsystem/level pair [%v]"
			return er.Errorf(str, logLevelPair)
		}

		// Extract the specified subsystem and log level.
		fields := strings.Split(logLevelPair, "=")
		subsysID, logLevel := fields[0], fields[1]

		if lvl, ok := LevelFromString(logLevel); !ok {
			return er.Errorf("The specified debug level [%v] is invalid", logLevel)
		} else {
			m[subsysID] = lvl
		}
	}

	b.lock.Lock()
	defer b.lock.Unlock()
	if glvl != LevelInvalid {
		b.lvl = glvl
	}
	b.lmap = m
	return nil
}

// String returns the tag of the logger used in log messages, or "OFF" if
// the level will not produce any log output.
func (l Level) String() string {
	if l >= LevelOff {
		return "OFF"
	}
	return levelStrs[l]
}

const (
	defaultFlags = Lshortfile | Lcolor
	defaultLevel = LevelDebug
)

// newBackend creates a logger backend from a Writer.
func newBackend(w io.Writer) *backend {
	flags := uint32(0)
	hasFlags := false
	for _, f := range strings.Split(os.Getenv("LOGFLAGS"), ",") {
		switch f {
		case "none":
		case "longfile":
			flags |= Llongfile
		case "shortfile":
			flags |= Lshortfile
		case "color":
			flags |= Lcolor
		case "longdate":
			flags |= Llongdate
		default:
			continue
		}
		hasFlags = true
	}
	if !hasFlags {
		flags = defaultFlags
	}

	b := &backend{
		flag: flags,
		ch:   make(chan *[]byte, 1024),
		lvl:  defaultLevel,
		lmap: make(map[string]Level),
	}
	go func() {
		for {
			l := <-b.ch
			w.Write(*l)
			recycleBuffer(l)
		}
	}()
	return b
}

// bufferPool defines a concurrent safe free list of byte slices used to provide
// temporary buffers for formatting log messages prior to outputting them.
var bufferPool = sync.Pool{
	New: func() interface{} {
		b := make([]byte, 0, 120)
		return &b // pointer to slice to avoid boxing alloc
	},
}

// buffer returns a byte slice from the free list.  A new buffer is allocated if
// there are not any available on the free list.  The returned byte slice should
// be returned to the fee list by using the recycleBuffer function when the
// caller is done with it.
func buffer() *[]byte {
	return bufferPool.Get().(*[]byte)
}

// recycleBuffer puts the provided byte slice, which should have been obtain via
// the buffer function, back on the free list.
func recycleBuffer(b *[]byte) {
	*b = (*b)[:0]
	bufferPool.Put(b)
}

// From stdlib log package.
// Cheap integer to fixed-width decimal ASCII.  Give a negative width to avoid
// zero-padding.
func itoa(buf *[]byte, i, wid int) {
	// Assemble decimal in reverse order.
	var b [20]byte
	bp := len(b) - 1
	for i >= 10 || wid > 1 {
		wid--
		q := i / 10
		b[bp] = byte('0' + i - q*10)
		bp--
		i = q
	}
	// i < 10
	b[bp] = byte('0' + i)
	*buf = append(*buf, b[bp:]...)
}

const (
	Reset      = "\x1b[0m"
	Bright     = "\x1b[1m"
	dim        = "\x1b[2m"
	underscore = "\x1b[4m"
	blink      = "\x1b[5m"
	reverse    = "\x1b[7m"
	hidden     = "\x1b[8m"

	fgBlack   = "\x1b[30m"
	fgRed     = "\x1b[31m"
	FgGreen   = "\x1b[32m"
	fgYellow  = "\x1b[33m"
	fgBlue    = "\x1b[34m"
	FgMagenta = "\x1b[35m"
	fgCyan    = "\x1b[36m"
	fgWhite   = "\x1b[37m"

	bgBlack   = "\x1b[40m"
	bgRed     = "\x1b[41m"
	BgGreen   = "\x1b[42m"
	bgYellow  = "\x1b[43m"
	bgBlue    = "\x1b[44m"
	bgMagenta = "\x1b[45m"
	bgCyan    = "\x1b[46m"
	bgWhite   = "\x1b[47m"

	colorDbg  = dim + fgWhite
	colorWarn = Bright + fgYellow
	colorErr  = Bright + fgRed
	colorCrit = Bright + fgBlack + bgRed
)

func Height(h int32) string {
	out := "unconfirmed"
	if h > -1 {
		out = strconv.FormatInt(int64(h), 10)
	}
	return fgYellow + out + Reset
}

func Txid(str string) string {
	return fgCyan + str + Reset
}

func GreenBg(str string) string {
	return BgGreen + fgBlack + str + Reset
}

func BgYellow(str string) string {
	return bgYellow + fgBlack + str + Reset
}

func Coins(amount float64) string {
	return Bright + FgGreen + strconv.FormatFloat(amount, 'f', 4, 64) + Reset
}

func Address(addr string) string {
	return Bright + FgMagenta + addr + Reset
}

func IpAddr(addr string) string {
	return Bright + fgRed + addr + Reset
}

func Int(num int) string {
	return Bright + fgYellow + strconv.FormatInt(int64(num), 10) + Reset
}

// Appends a header in the default format 'YYYY-MM-DD hh:mm:ss.sss [LVL] TAG: '.
// If either of the Lshortfile or Llongfile flags are specified, the file named
// and line number are included after the tag and before the final colon.
func formatHeader(flags uint32, buf *[]byte, t time.Time, lvl Level, file string, line int) bool {
	hasColor := false
	if flags&Lcolor == Lcolor {
		hasColor = true
		switch lvl {
		case LevelDebug:
			*buf = append(*buf, colorDbg...)
		case LevelWarn:
			*buf = append(*buf, colorWarn...)
		case LevelError:
			*buf = append(*buf, colorErr...)
		case LevelCritical:
			*buf = append(*buf, colorCrit...)
		default:
			hasColor = false
		}
	}

	if flags&Llongdate == Llongdate {
		year, month, day := t.Date()
		hour, min, sec := t.Clock()
		ms := t.Nanosecond() / 1e6

		itoa(buf, year, 4)
		*buf = append(*buf, '-')
		itoa(buf, int(month), 2)
		*buf = append(*buf, '-')
		itoa(buf, day, 2)
		*buf = append(*buf, ' ')
		itoa(buf, hour, 2)
		*buf = append(*buf, ':')
		itoa(buf, min, 2)
		*buf = append(*buf, ':')
		itoa(buf, sec, 2)
		*buf = append(*buf, '.')
		itoa(buf, ms, 3)
	} else {
		itoa(buf, int(t.Unix()), -1)
	}
	*buf = append(*buf, " ["...)
	*buf = append(*buf, lvl.String()...)
	*buf = append(*buf, "] "...)
	if flags&(Lshortfile|Llongfile) != 0 {
		*buf = append(*buf, file...)
		*buf = append(*buf, ':')
		itoa(buf, line, -1)
		*buf = append(*buf, ' ')
	}

	return hasColor
}

// calldepth is the call depth of the callsite function relative to the
// caller of the subsystem logger.  It is used to recover the filename and line
// number of the logging call if either the short or long file flags are
// specified.
const calldepth = 3

// callsite returns the file name and line number of the callsite to the
// subsystem logger.
func callsite(flag uint32) (string, string, int) {
	_, file, line, ok := runtime.Caller(calldepth)
	if !ok {
		return "???", "", 0
	}
	short := file
	for i := len(file) - 1; i > 0; i-- {
		if os.IsPathSeparator(file[i]) {
			short = file[i+1:]
			break
		}
	}
	if flag&Lshortfile != 0 {
		file = short
	}
	return file, short, line
}

func (b *backend) write(buf *[]byte) {
	select {
	case b.ch <- buf:
		// ok
	default:
		// failed, recycle the buffer ourselves
		recycleBuffer(buf)
	}
}

// backend is a logging backend.  Subsystems created from the backend write to
// the backend's Writer.  backend provides atomic writes to the Writer from all
// subsystems.
type backend struct {
	ch   chan *[]byte
	flag uint32

	lock sync.RWMutex
	lvl  Level
	lmap map[string]Level
}

var b *backend

func init() {
	b = newBackend(os.Stdout)
	pktlog := os.Getenv("PKTLOG")
	if pktlog != "" {
		if err := SetLogLevels(pktlog); err != nil {
			Errorf("Error setting log param: ", err.String())
		}
	}
}

// doLog outputs a log message to the writer associated with the backend after
// creating a prefix for the given level and tag according to the formatHeader
// function and formatting the provided arguments according to the given format
// specifier.
func doLog(
	lvl Level,
	format string,
	args ...interface{},
) {
	file, shortFile, line := callsite(b.flag)
	doit := true
	b.lock.RLock()
	if lvl >= b.lvl {
	} else if lvl1, ok := b.lmap[shortFile]; ok && lvl >= lvl1 {
	} else {
		doit = false
	}
	b.lock.RUnlock()
	if !doit {
		return
	}

	t := time.Now()
	bytebuf := buffer()
	hasColor := formatHeader(b.flag, bytebuf, t, lvl, file, line)
	buf := bytes.NewBuffer(*bytebuf)
	if format == "" {
		fmt.Fprintln(buf, args...)
	} else {
		fmt.Fprintf(buf, format, args...)
	}
	*bytebuf = buf.Bytes()
	if hasColor {
		*bytebuf = append(*bytebuf, Reset...)
	}
	*bytebuf = append(*bytebuf, '\n')

	b.write(bytebuf)
}

func Trace(args ...interface{}) {
	doLog(LevelTrace, "", args...)
}

func Tracef(format string, args ...interface{}) {
	doLog(LevelTrace, format, args...)
}

func Debug(args ...interface{}) {
	doLog(LevelDebug, "", args...)
}

func Debugf(format string, args ...interface{}) {
	doLog(LevelDebug, format, args...)
}

func Info(args ...interface{}) {
	doLog(LevelInfo, "", args...)
}

func Infof(format string, args ...interface{}) {
	doLog(LevelInfo, format, args...)
}

func Warn(args ...interface{}) {
	doLog(LevelWarn, "", args...)
}

func Warnf(format string, args ...interface{}) {
	doLog(LevelWarn, format, args...)
}

func Error(args ...interface{}) {
	doLog(LevelError, "", args...)
}

func Errorf(format string, args ...interface{}) {
	doLog(LevelError, format, args...)
}

func Critical(args ...interface{}) {
	doLog(LevelCritical, "", args...)
}

func Criticalf(format string, args ...interface{}) {
	doLog(LevelCritical, format, args...)
}

// logClosure is used to provide a closure over expensive logging operations so
// don't have to be performed when the logging level doesn't warrant it.
type logClosure func() string

// String invokes the underlying function and returns the result.
func (c logClosure) String() string {
	return c()
}

// log.C returns a new closure over a function that returns a string
// which itself provides a Stringer interface so that it can be used with the
// logging system.
func C(c func() string) logClosure {
	return logClosure(c)
}

func WarnIfPrerelease() {
	if version.IsCustom() || version.IsDirty() {
		Warnf(
			"THIS IS A DEVELOPMENT VERSION, THINGS MAY BREAK",
		)
	} else if version.IsPrerelease() {
		Infof(
			"This is a pre-release version",
		)
	}
}
