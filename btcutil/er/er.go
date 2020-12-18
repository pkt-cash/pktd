package er

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"regexp"
	"runtime/debug"
	"strings"

	"github.com/pkt-cash/pktd/pktconfig/version"
)

// GenericErrorType is for packages with only one or two error codes
// which don't make sense having their own error type
var GenericErrorType = NewErrorType("er.GenericErrorType")

var (
	ErrUnexpectedEOF = GenericErrorType.CodeWithDefault("ErrUnexpectedEOF", io.ErrUnexpectedEOF)
	EOF              = GenericErrorType.CodeWithDefault("EOF", io.EOF)
)

// ErrorCode is a code for identifying a particular type of fault.
// Error codes can have a numeric code identifier or they can not.
type ErrorCode struct {
	Detail         string
	Number         int
	Type           *ErrorType
	defaultWrapped error
}

type typedErr struct {
	messages []string
	errType  *ErrorType
	code     *ErrorCode
	err      R
}

// Error type

// ErrorType is a generic type of error, each type can have many error codes
type ErrorType struct {
	Name       string
	codeLookup map[int]*ErrorCode
	Codes      []*ErrorCode
}

// NewErrorType creates a new error type, it must be identified by name.
// For example: var MyError er.ErrorType = NewErrorType("mypackage.MyError")
func NewErrorType(ident string) ErrorType {
	return ErrorType{
		Name:       ident,
		codeLookup: make(map[int]*ErrorCode),
	}
}

func (c *ErrorCode) Is(err R) bool {
	if err == nil {
		return c == nil
	}
	if te, ok := err.(typedErr); ok {
		return te.code == c
	}
	return false
}

func (c *ErrorCode) new(info string, err R, bstack []byte) R {
	var messages []string
	if info == "" {
		messages = []string{c.Detail}
	} else {
		messages = []string{c.Detail, info}
	}
	if err == nil {
		if bstack == nil {
			bstack = captureStack()
		}
		err = new("", bstack)
	} else if te, ok := err.(typedErr); ok {
		if te.code == c {
			if info != "" {
				te.messages = append(messages, te.messages...)
			}
			return te
		}
	}
	return typedErr{
		messages: messages,
		errType:  c.Type,
		code:     c,
		err:      err,
	}
}

func (c *ErrorCode) New(info string, err R) R {
	if err == nil {
		return c.new(info, nil, captureStack())
	} else {
		return c.new(info, err, nil)
	}
}

func (e *ErrorType) Is(err R) bool {
	if err == nil {
		return false
	}
	if te, ok := err.(typedErr); ok {
		return te.errType == e
	}
	return false
}

func (e *ErrorType) Decode(err R) *ErrorCode {
	if err == nil {
		return nil
	}
	if te, ok := err.(typedErr); ok {
		return te.code
	}
	return nil
}

// NumberedCode constructs a new error code with a number.
func (e *ErrorType) newErrorCode(
	number int,
	hasNumber bool,
	info string,
	detail string,
) *ErrorCode {
	var result *ErrorCode
	var header string
	if hasNumber {
		header = fmt.Sprintf("%s(%d)", info, number)
	} else {
		header = info
	}
	if detail != "" {
		header = header + ": " + detail
	}
	result = &ErrorCode{
		Detail: header,
		Type:   e,
		Number: number,
	}
	if hasNumber {
		e.codeLookup[number] = result
	}
	e.Codes = append(e.Codes, result)
	return result
}

func (c *ErrorCode) Default() R {
	if c.defaultWrapped != nil {
		return c.new("", ee(c.defaultWrapped), nil)
	} else {
		return c.new("", nil, captureStack())
	}
}

func (e *ErrorType) Code(info string) *ErrorCode {
	return e.newErrorCode(0, false, info, "")
}

func (e *ErrorType) CodeWithDefault(info string, defaultError error) *ErrorCode {
	ec := e.newErrorCode(0, false, info, "")
	ec.defaultWrapped = defaultError
	return ec
}

func (e *ErrorType) CodeWithDetail(info, detail string) *ErrorCode {
	return e.newErrorCode(0, false, info, detail)
}

func (e *ErrorType) CodeWithNumber(info string, number int) *ErrorCode {
	return e.newErrorCode(number, true, info, "")
}

func (e *ErrorType) CodeWithNumberAndDetail(info string, number int, detail string) *ErrorCode {
	return e.newErrorCode(number, true, info, detail)
}

func (e *ErrorType) NumberToCode(number int) *ErrorCode {
	return e.codeLookup[number]
}

func (te typedErr) AddMessage(m string) {
	te.messages = append([]string{m}, te.messages...)
}

func (te typedErr) Message() string {
	tem := te.err.Message()
	if tem == "" {
		return strings.Join(te.messages, ": ")
	}
	return fmt.Sprintf("%s: %s", strings.Join(te.messages, ": "), te.err.Message())
}

func (te typedErr) HasStack() bool {
	return te.err.HasStack()
}

func (te typedErr) Stack() []string {
	return te.err.Stack()
}

func (te typedErr) String() string {
	s := ""
	if te.err.HasStack() {
		s = "\n\n" + strings.Join(te.err.Stack(), "\n") + "\n"
	}
	return version.Version() + " " + te.Message() + s
}

func (te typedErr) Error() string {
	return te.String()
}

func (te typedErr) Wrapped0() error {
	return te.err.Wrapped0()
}

type typedErrAsNative struct {
	e typedErr
}

func (ten typedErrAsNative) Error() string {
	return ten.e.String()
}

func (te typedErr) Native() error {
	return typedErrAsNative{e: te}
}

//////
/// er.R
//////

type R interface {
	Message() string
	Stack() []string
	HasStack() bool
	String() string
	Wrapped0() error
	Native() error
	AddMessage(m string)
}

type err struct {
	messages []string
	e        error
	bstack   []byte
	stack    []string
}

type errAsNative struct {
	e err
}

func (e errAsNative) Error() string {
	return e.e.String()
}

func (e err) HasStack() bool {
	return e.bstack != nil
}

var (
	argumentsRegex = regexp.MustCompile(`\([0-9a-fx, \.]*\)$`)
	prefixRegex    = regexp.MustCompile(`^.*/pkt-cash/pktd/`)
	goFileRegex    = regexp.MustCompile(`\.go:[0-9]+ `)
)

func (e err) Stack() []string {
	if e.stack == nil {
		s := strings.Split(string(e.bstack), "\n")
		// First 5 lines are noise
		// goroutine 1 [running]:
		// runtime/debug.Stack(0x10df124, 0xc00007cf70, 0xc0000180c0)
		//         /usr/local/go/src/runtime/debug/stack.go:24 +0x9d
		// github.com/pkt-cash/pktd/btcutil/er.captureStack(...)
		//         /Users/user/wrk/pkt-cash/pktd/btcutil/er/er.go:283
		s = s[5:]
		var stack []string
		fun := ""
		for i := range s {
			x := argumentsRegex.ReplaceAllString(s[i], "()")
			x = prefixRegex.ReplaceAllString(x, "")
			x = "  " + strings.TrimSpace(x)
			if !goFileRegex.MatchString(x) {
				fun = x
			} else {
				stack = append(stack, x+"\t"+fun)
			}
		}
		e.stack = stack
	}
	return e.stack
}

func (e err) AddMessage(m string) {
	if e.messages == nil {
		e.messages = []string{m, e.e.Error()}
	} else {
		e.messages = append([]string{m}, e.messages...)
	}
}

func (e err) Message() string {
	if e.messages == nil {
		return e.e.Error()
	}
	return strings.Join(e.messages, ", ")
}

func (e err) String() string {
	s := ""
	if e.bstack != nil {
		s = "\n\n" + strings.Join(e.Stack(), "\n") + "\n"
	}
	return version.Version() + " " + e.Message() + s
}

func (e err) Error() string {
	return e.String()
}

func (e err) Wrapped0() error {
	return e.e
}

func (e err) Native() error {
	return errAsNative{e: e}
}

//////

func captureStack() []byte {
	return debug.Stack()
}

func Wrapped(err R) error {
	if err == nil {
		return nil
	}
	return err.Wrapped0()
}

func Native(err R) error {
	if err == nil {
		return nil
	}
	return err.Native()
}

func new(s string, bstack []byte) R {
	return err{
		e:      errors.New(s),
		bstack: bstack,
	}
}

func New(s string) R {
	return new(s, captureStack())
}

func Errorf(format string, a ...interface{}) R {
	return err{
		e:      fmt.Errorf(format, a...),
		bstack: captureStack(),
	}
}

func ee(e error) R {
	return err{
		e:      e,
		bstack: captureStack(),
	}
}

func E(e error) R {
	if e == nil {
		return nil
	}
	if en, ok := e.(errAsNative); ok {
		return en.e
	}
	if en, ok := e.(typedErrAsNative); ok {
		return en.e
	}
	switch e {
	// We must capture the inner error so that er.Wrapped() does the right thing
	case io.ErrUnexpectedEOF:
		return ErrUnexpectedEOF.Default()
	case io.EOF:
		return EOF.Default()
	default:
		return ee(e)
	}
}

func equals(e, r R, fuzzy bool) bool {
	if e == nil || r == nil {
		return e == nil && r == nil
	}
	if te, ok := e.(typedErr); ok {
		if tr, ok := r.(typedErr); ok {
			return te.code == tr.code
		}
		// e is a typed error, r isn't
		return false
	}
	if ee, ok := e.(err); ok {
		if rr, ok := r.(err); ok {
			if ee.e != nil && rr.e != nil {
				// e and r are made by er.E(), check they wrap the same underlying error type
				if ee.e == rr.e {
					return true
				}
				if fuzzy {
					return reflect.TypeOf(ee.e) == reflect.TypeOf(rr.e)
				}
			}
			// doesn't wrap anything, therefore every er.R is unique
			// TODO(cjd): Maybe we want to grab the pc of the caller to er.E() and
			// store that so we can consider the same 2 errors which were made in an identical
			// location.
			return false
		}
		// differing types
		return false
	}
	panic("I don't know what error type this is: " + reflect.TypeOf(e).Name())
}

func Equals(e, r R) bool {
	return equals(e, r, false)
}

func FuzzyEquals(e, r R) bool {
	return equals(e, r, true)
}

var errrLoopbreak = errors.New("loop break (if you're seeing this error, it should have been caught)")

// A (non) error which is used to break out of a forEach loop
var LoopBreak = E(errrLoopbreak)

func IsLoopBreak(e R) bool {
	en, ok := e.(err)
	return ok && en.e == errrLoopbreak
}

func Cis(code *ErrorCode, e R) bool {
	if code == nil {
		return e == nil
	}
	return code.Is(e)
}
