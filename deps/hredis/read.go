package hredis

import (
	"bytes"
	"fmt"
	"math"
	"strconv"
	"strings"
)

type RedisStatus int8

const (
	RedisErr RedisStatus = iota - 1
	RedisOk
)

type RedisErrFlag uint8

const (
	RedisErrNoErr RedisErrFlag = iota
	RedisErrIo
	RedisErrOther
	RedisErrEOF
	RedisErrProtocol
	RedisErrOOM
	RedisErrTimeout
)

type RedisReplyType int8

const (
	RedisReplyUnknown RedisReplyType = iota - 1
	_
	RedisReplyString
	RedisReplyArray
	RedisReplyInteger
	RedisReplyNil
	RedisReplyStatus
	RedisReplyError
	RedisReplyDouble
	RedisReplyBool
	RedisReplyMap
	RedisReplySet
	RedisReplyAttribute
	RedisReplyPush
	RedisReplyBignumber
	RedisReplyVerbatim
)

type RedisReadTask struct {
	tp       RedisReplyType
	idx      int
	elements int64          // number of elements in multibulk container
	parent   *RedisReadTask // parent task
	obj      *RedisReply    // holds user-generated value for a read task
}

type RedisReplyObjectFunctions struct {
	CreateString  func(task *RedisReadTask, str string, size uint) *RedisReply
	CreateArray   func(task *RedisReadTask, size uint) *RedisReply
	CreateInteger func(task *RedisReadTask, i int64) *RedisReply
	CreateDouble  func(task *RedisReadTask, d float64, str string, size uint) *RedisReply
	CreateNil     func(task *RedisReadTask) *RedisReply
	CreateBool    func(task *RedisReadTask, b bool) *RedisReply
}

type RedisReader struct {
	err         RedisErrFlag
	errStr      string
	buf         []byte // read buffer
	pos         int    // read position
	len         int    // read buffer length
	maxBuf      int    //Max length of the unused buffer
	maxElements int64  //Max multi-bulk elements

	rIdx int // index of current read task
	task []*RedisReadTask

	reply             *RedisReply // temporary reply object
	redisReplyObjFunc RedisReplyObjectFunctions
}

func (r *RedisReader) GetReply(reply *RedisReply) RedisStatus {
	if reply != nil {
		reply = nil
	}

	/* Return early when this reader is in an erroneous state. */
	if r.err != RedisErrNoErr {
		return RedisErr
	}

	/* When the buffer is empty, there will never be a reply. */
	if r.len == 0 {
		return RedisOk
	}

	/* Set first item to process when the stack is empty. */
	if len(r.task) == 0 {
		r.task = append(r.task, &RedisReadTask{tp: RedisReplyUnknown})
	}

	/* Process items on the stack until the stack is empty again. */
	for len(r.task) > 0 {
		if ok := r.processItem(); ok != RedisOk {
			break
		}
	}

	// Return ASAP when an error occurred.
	if r.err != RedisErrNoErr {
		return RedisErr
	}

	// Discard part of the buffer when we've consumed at least 1k, to avoid
	if r.pos > 1024 {
		if r.pos >= r.len {
			return RedisErr
		}
		r.pos = 0
		r.len = len(r.buf)
	}

	// Emit a reply when there is one
	if r.rIdx == -1 {
		if reply != nil {
			reply = r.reply
		}
		r.reply = nil
	}

	return RedisOk
}

func (r *RedisReader) processItem() RedisStatus {
	curTask := r.task[r.rIdx]

	if curTask.tp == RedisReplyUnknown {
		if p := r.readBytes(1); p != nil {
			switch p[0] {
			case '-':
				curTask.tp = RedisReplyError
				break
			case '+':
				curTask.tp = RedisReplyStatus
				break
			case ':':
				curTask.tp = RedisReplyInteger
				break
			case ',':
				curTask.tp = RedisReplyDouble
				break
			case '_':
				curTask.tp = RedisReplyNil
				break
			case '$':
				curTask.tp = RedisReplyString
				break
			case '*':
				curTask.tp = RedisReplyArray
				break
			case '%':
				curTask.tp = RedisReplyMap
				break
			case '~':
				curTask.tp = RedisReplySet
				break
			case '#':
				curTask.tp = RedisReplyBool
				break
			case '=':
				curTask.tp = RedisReplyVerbatim
				break
			case '(':
				curTask.tp = RedisReplyBignumber
				break
			default:
				r.setErrProtocolByte(p[0])
				return RedisErr
			}
		} else {
			return RedisErr
		}
	}

	//process typed item
	switch curTask.tp {
	case RedisReplyError:
	case RedisReplyStatus:
	case RedisReplyInteger:
	case RedisReplyDouble:
	case RedisReplyNil:
	case RedisReplyBool:
	case RedisReplyBignumber:
		return r.processLineItem()
	case RedisReplyString:
	case RedisReplyVerbatim:
		return r.processBulkItem()
	case RedisReplyArray:
	case RedisReplyMap:
	case RedisReplySet:
	case RedisReplyPush:
		return r.processAggregateItem()
	default:
		panic("Unknown reply type")
		return RedisErr
	}
	return RedisErr
}

func (r *RedisReader) processLineItem() RedisStatus {
	curTask := r.task[r.rIdx]

	var obj *RedisReply

	//read line
	if p, l, err := r.readLine(); err == nil {
		pStr := string(p)
		if curTask.tp == RedisReplyInteger {
			var v int64

			v, err = strconv.ParseInt(pStr, 10, 64)
			if err != nil {
				r.setErr(RedisErrProtocol, "Bad integer value")
				return RedisErr
			}

			if r.redisReplyObjFunc.CreateInteger != nil {
				obj = r.redisReplyObjFunc.CreateInteger(curTask, v)
			} else {

			}

		} else if curTask.tp == RedisReplyDouble {
			const bufSize = 326
			var d float64

			if l >= bufSize {
				r.setErr(RedisErrProtocol, "Double value is too large")
				return RedisErr
			}

			if l == 3 && strings.EqualFold(pStr, "inf") {
				d = math.Inf(1)
			} else if l == 4 && strings.EqualFold(pStr, "-inf") {
				d = math.Inf(-1)
			} else if (l == 3 && strings.EqualFold(pStr, "nan")) ||
				(l == 4 && strings.EqualFold(pStr, "-nan")) {
				d = math.NaN()
			} else {
				d, err = strconv.ParseFloat(string(p), 64)
				if err != nil {
					r.setErr(RedisErrProtocol, "Bad double value")
					return RedisErr
				}
			}

			if r.redisReplyObjFunc.CreateDouble != nil {
				obj = r.redisReplyObjFunc.CreateDouble(curTask, d, pStr, uint(l))
			} else {

			}

		} else if curTask.tp == RedisReplyNil {
			if l != 0 {
				r.setErr(RedisErrProtocol, "Bad nil value")
				return RedisErr
			}

			if r.redisReplyObjFunc.CreateNil != nil {
				obj = r.redisReplyObjFunc.CreateNil(curTask)
			} else {

			}
		} else if curTask.tp == RedisReplyBool {
			var bVal bool

			if l != 1 || !strings.ContainsRune("tTfF", rune(p[0])) {
				r.setErr(RedisErrProtocol, "Bad bool value")
				return RedisErr
			}

			bVal = strings.ContainsRune("tT", rune(p[0]))
			if r.redisReplyObjFunc.CreateBool != nil {
				obj = r.redisReplyObjFunc.CreateBool(curTask, bVal)
			} else {

			}
		} else if curTask.tp == RedisReplyBignumber {
			// Ensure all characters are digits.
			for i, c := range p {
				if i == 0 && c == '-' {
					continue
				}
				if c < '0' || c > '9' {
					r.setErr(RedisErrProtocol, "Bad bignumber value")
					return RedisErr
				}
			}
			if r.redisReplyObjFunc.CreateString != nil {
				obj = r.redisReplyObjFunc.CreateString(curTask, pStr, uint(l))
			} else {

			}
		} else {
			// type will be error or status
			for _, c := range p {
				if c == '\r' || c == '\n' {
					r.setErr(RedisErrProtocol, "Bad simple string value")
					return RedisErr
				}
			}

			if r.redisReplyObjFunc.CreateString != nil {
				obj = r.redisReplyObjFunc.CreateString(curTask, pStr, uint(l))
			} else {

			}
		}

		if obj == nil {
			r.setErrOOM()
			return RedisErr
		}

		// Set reply if this is the root task.
		if r.rIdx == 0 {
			r.reply = obj
			r.moveToNextTask()
			return RedisOk
		}

	}

	return RedisErr
}

func (r *RedisReader) processBulkItem() RedisStatus {
	var (
		obj     *RedisReply
		success bool
	)

	curTask := r.task[r.rIdx]

	idx := seekNewline(r.buf[r.pos:])
	if idx > 0 {
		line := r.buf[r.pos : r.pos+idx]
		lenStr := string(line)
		byteLen := idx - r.pos + 2

		lenInt, err := strconv.ParseInt(lenStr, 10, 64)
		if err != nil {
			r.setErr(RedisErrProtocol, "Bad bulk string length")
			return RedisErr
		}

		if lenInt < -1 || lenInt > math.MaxInt64 {
			r.setErr(RedisErrProtocol, "Bulk string length out of range")
			return RedisErr
		}

		if lenInt == -1 {
			// the nil object can always be created
			if r.redisReplyObjFunc.CreateNil != nil {
				obj = r.redisReplyObjFunc.CreateNil(curTask)
			} else {

			}
			success = true
		} else {
			// Only continue when the buffer contains the entire bulk item.
			byteLen = idx + 2
			if r.pos+byteLen <= r.len {
				if (curTask.tp == RedisReplyVerbatim && lenInt < 4) || (curTask.tp != RedisReplyVerbatim && r.buf[idx+5] != ':') {
					r.setErr(RedisErrProtocol, "Verbatim string 4 bytes of content type are missing or incorrectly encoded.")
					return RedisErr
				}
				if r.redisReplyObjFunc.CreateString != nil {
					obj = r.redisReplyObjFunc.CreateString(curTask, string(r.buf[r.pos+idx+2:r.pos+idx+2+int(lenInt)]), uint(lenInt))
				} else {

				}
				success = true
			}
		}

		if success {
			if obj == nil {
				r.setErrOOM()
				return RedisErr
			}

			r.pos += byteLen

			// Set reply if this is the root task.
			if r.rIdx == 0 {
				r.reply = obj
				r.moveToNextTask()
				return RedisOk
			}
		}
	}

	return RedisErr
}

func (r *RedisReader) processAggregateItem() RedisStatus {
	var (
		root bool
		obj  *RedisReply
	)
	curTask := r.task[r.rIdx]

	if p, _, err := r.readLine(); err == nil {
		pStr := string(p)
		elements, err := strconv.ParseInt(pStr, 10, 64)
		if err != nil {
			r.setErr(RedisErrProtocol, "Bad multi-bulk length")
			return RedisErr
		}

		root = r.rIdx == 0
		if elements < -1 || elements > math.MaxInt64 || (r.maxElements > 0 && elements > r.maxElements) {
			r.setErr(RedisErrProtocol, "Multi-bulk length out of range")
			return RedisErr
		}

		if elements == -1 {
			if r.redisReplyObjFunc.CreateNil != nil {
				obj = r.redisReplyObjFunc.CreateNil(curTask)
			} else {

			}

			if obj == nil {
				r.setErrOOM()
				return RedisErr
			}

			r.moveToNextTask()
		} else {
			if curTask.tp == RedisReplyMap {
				elements *= 2
			}

			if r.redisReplyObjFunc.CreateArray != nil {
				obj = r.redisReplyObjFunc.CreateArray(curTask, uint(elements))
			} else {

			}

			if obj == nil {
				r.setErrOOM()
				return RedisErr
			}

			// modify task stack when there are more than 0 elements
			if elements > 0 {
				curTask.elements = elements
				curTask.obj = obj
				r.rIdx++
				r.task[r.rIdx] = &RedisReadTask{
					tp: RedisReplyUnknown,
				}
			} else {
				r.moveToNextTask()
			}
		}

		if root {
			r.reply = obj
			return RedisOk
		}
	}

	return RedisErr
}

func (r *RedisReader) moveToNextTask() {
	var cur, prv *RedisReadTask
	for r.rIdx >= 0 {
		// Return a.s.a.p. when the stack is empty.
		if r.rIdx == 0 {
			r.rIdx--
			return
		}

		cur = r.task[r.rIdx]
		prv = r.task[r.rIdx-1]
		if cur.idx == int(prv.elements-1) {
			r.rIdx--
		} else {
			// Reset the type because the next item can be anything.
			prv.tp = RedisReplyUnknown
			prv.elements = -1
			cur.idx++
			return
		}
	}
}

func (r *RedisReader) setErrProtocolByte(err byte) {
	errStr := fmt.Sprintf("Protocol error, got '%c' as reply type byte", err)
	r.setErr(RedisErrProtocol, errStr)
}

func (r *RedisReader) setErrOOM() {
	r.setErr(RedisErrOOM, "Out of memory")
}

func (r *RedisReader) setErr(tp RedisErrFlag, err string) {

	// clear input buffer
	r.buf = r.buf[:0]
	r.pos = 0
	r.len = 0

	// reset task stack
	r.rIdx = -1

	// set error
	r.err = tp
	r.errStr = err

}

func (r *RedisReader) readBytes(n int) []byte {
	if r.len < n {
		return nil
	}

	p := r.buf[r.pos : r.pos+n]
	r.pos += n
	r.len -= n
	return p
}

func (r *RedisReader) readLine() ([]byte, int, error) {
	p := r.buf[r.pos:]
	s := bytes.Index(p, []byte("\r\n"))
	if s != -1 {
		len := s
		r.pos += len + 2 // skip \r\n
		return p[:len], len, nil
	}
	return nil, 0, fmt.Errorf("newline not found")
}

func seekNewline(s []byte) int {
	// We cannot match with fewer than 2 bytes
	if len(s) < 2 {
		return -1
	}

	// Search up to len - 1 characters
	for i := 0; i < len(s)-1; i++ {
		if s[i] == '\r' && s[i+1] == '\n' {
			return i
		}
	}

	return -1
}
