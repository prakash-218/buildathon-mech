package auto

// Ported from src/runtime.rs: stdin/stdout loop with a reader goroutine, a brain worker
// goroutine and a deadline fallback.

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"math/big"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

const botName = "osu-mech"

// BotName is BOT_NAME.
const BotName = botName

var runtimeLog atomic.Bool

func logEnabled() bool {
	return runtimeLog.Load()
}

// logMsg writes one line to stderr when MECH_LOG=1 (Rust runtime::log).
func logMsg(msg string) {
	if logEnabled() {
		fmt.Fprintf(os.Stderr, "[autogo] %s\n", msg)
	}
}

// catchUnwind runs f and reports whether it panicked (std::panic::catch_unwind). Like the
// silenced Rust panic hook, the panic is only printed when logging is enabled.
func catchUnwind(f func()) (panicked bool) {
	defer func() {
		if r := recover(); r != nil {
			panicked = true
			if logEnabled() {
				fmt.Fprintf(os.Stderr, "panic: %v\n%s", r, debug.Stack())
			}
		}
	}()
	f()
	return false
}

// ---------------------------------------------------------------- Budget

// Budget is the time the Brain may still spend on the current observation.
type Budget struct {
	dl *time.Time // None: unlimited
}

func budgetUnlimited() *Budget {
	return &Budget{dl: nil}
}

// UnlimitedBudget is Budget::unlimited.
func UnlimitedBudget() *Budget {
	return budgetUnlimited()
}

func budgetUntil(deadline time.Time) *Budget {
	d := deadline
	return &Budget{dl: &d}
}

func budgetFromNowMs(ms float64) *Budget {
	return budgetUntil(time.Now().Add(runtimeDurFromSecsF64(fmax(ms, 0.0) / 1000.0)))
}

func (b *Budget) isUnlimited() bool {
	return b.dl == nil
}

func (b *Budget) deadline() (time.Time, bool) {
	if b.dl == nil {
		return time.Time{}, false
	}
	return *b.dl, true
}

// remaining is Duration::MAX when unlimited, otherwise the time left (never negative).
func (b *Budget) remaining() time.Duration {
	if b.dl == nil {
		return time.Duration(math.MaxInt64)
	}
	d := b.dl.Sub(time.Now())
	if d < 0 {
		return 0
	}
	return d
}

// remainingMs is +Inf when unlimited.
func (b *Budget) remainingMs() float64 {
	if b.dl == nil {
		return math.Inf(1)
	}
	return b.remaining().Seconds() * 1000.0
}

func (b *Budget) expired() bool {
	return b.dl != nil && !time.Now().Before(*b.dl)
}

// runtimeDurFromSecsF64 is Duration::from_secs_f64: rounds to the nearest nanosecond (ties to
// even) and panics on negative, NaN, infinite or too large input. Durations beyond Go's range
// (about 292 years) saturate.
func runtimeDurFromSecsF64(s float64) time.Duration {
	if !(s >= 0.0) || math.IsInf(s, 1) || s >= 0x1p64 {
		panic("cannot convert float seconds to Duration: value is negative, overflows Duration or is not finite")
	}
	x := new(big.Float).SetPrec(256).SetFloat64(s)
	x.Mul(x, new(big.Float).SetPrec(256).SetInt64(1_000_000_000))
	n, _ := x.Int(nil) // truncated toward zero
	rem := new(big.Float).SetPrec(256).Sub(x, new(big.Float).SetPrec(256).SetInt(n))
	switch rem.Cmp(big.NewFloat(0.5)) {
	case 1:
		n.Add(n, big.NewInt(1))
	case 0:
		if n.Bit(0) == 1 {
			n.Add(n, big.NewInt(1))
		}
	}
	if !n.IsInt64() {
		return time.Duration(math.MaxInt64)
	}
	return time.Duration(n.Int64())
}

// ---------------------------------------------------------------- channels

// runtimeChan is an unbounded single-consumer queue with std::sync::mpsc semantics.
type runtimeChan[T any] struct {
	mu     sync.Mutex
	q      []T
	txGone bool // all senders dropped
	rxGone bool // receiver dropped
	sig    chan struct{}
}

func newRuntimeChan[T any]() *runtimeChan[T] {
	return &runtimeChan[T]{sig: make(chan struct{}, 1)}
}

func (c *runtimeChan[T]) poke() {
	select {
	case c.sig <- struct{}{}:
	default:
	}
}

// send fails (false) once the receiver is gone.
func (c *runtimeChan[T]) send(x T) bool {
	c.mu.Lock()
	if c.rxGone {
		c.mu.Unlock()
		return false
	}
	c.q = append(c.q, x)
	c.mu.Unlock()
	c.poke()
	return true
}

func (c *runtimeChan[T]) closeTx() {
	c.mu.Lock()
	c.txGone = true
	c.mu.Unlock()
	c.poke()
}

func (c *runtimeChan[T]) closeRx() {
	c.mu.Lock()
	c.rxGone = true
	c.q = nil
	c.mu.Unlock()
}

func (c *runtimeChan[T]) take() (x T, ok bool, gone bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.q) > 0 {
		x = c.q[0]
		var zero T
		c.q[0] = zero
		c.q = c.q[1:]
		return x, true, false
	}
	return x, false, c.txGone
}

// recv blocks for the next message; false when the queue is empty and every sender is gone.
func (c *runtimeChan[T]) recv() (T, bool) {
	for {
		x, ok, gone := c.take()
		if ok {
			return x, true
		}
		if gone {
			return x, false
		}
		<-c.sig
	}
}

func (c *runtimeChan[T]) tryRecv() (T, bool) {
	x, ok, _ := c.take()
	return x, ok
}

const (
	runtimeRecvOk = iota
	runtimeRecvTimeout
	runtimeRecvDisconnected
)

func (c *runtimeChan[T]) recvTimeout(d time.Duration) (T, int) {
	timer := time.NewTimer(d)
	defer timer.Stop()
	for {
		x, ok, gone := c.take()
		if ok {
			return x, runtimeRecvOk
		}
		if gone {
			return x, runtimeRecvDisconnected
		}
		select {
		case <-c.sig:
		case <-timer.C:
			x, ok, gone := c.take()
			if ok {
				return x, runtimeRecvOk
			}
			if gone {
				return x, runtimeRecvDisconnected
			}
			return x, runtimeRecvTimeout
		}
	}
}

// ---------------------------------------------------------------- reader / worker

// Answer deadline used by the IO loop, measured from reading the observation.
const runtimeIoFraction = 0.70

// Deadline handed to the Brain.
const runtimeBrainFraction = 0.60

// (Instant, String)
type runtimeLine struct {
	x0 time.Time
	x1 string
}

type runtimeJob struct {
	obs    *Obs
	budget *Budget
}

type runtimeDoneKind int

const (
	runtimeDoneAct runtimeDoneKind = iota
	runtimeDonePanicked
)

// runtimeDone is `enum Done { Act(i64, Action), Panicked(i64) }`.
type runtimeDone struct {
	kind runtimeDoneKind
	tick int64
	act  Action
}

// runtimeCleanLine is String::from_utf8_lossy(..).trim().
func runtimeCleanLine(b []byte) string {
	return strings.TrimSpace(strings.ToValidUTF8(string(b), "�"))
}

// runtimeSpawnReader reads stdin lines. Lines that arrive together (already buffered) are
// sent as one batch.
func runtimeSpawnReader() *runtimeChan[[]runtimeLine] {
	tx := newRuntimeChan[[]runtimeLine]()
	go func() {
		defer tx.closeTx()
		rd := bufio.NewReaderSize(os.Stdin, 1<<20)
		var batch []runtimeLine
		eof := false
		for !eof {
			for {
				buf, err := rd.ReadBytes('\n')
				if err != nil && err != io.EOF {
					eof = true
					break
				}
				if len(buf) == 0 {
					eof = true
					break
				}
				s := runtimeCleanLine(buf)
				if s != "" {
					batch = append(batch, runtimeLine{time.Now(), s})
				}
				if err == io.EOF {
					// Rust's next read_until returns Ok(0).
					eof = true
					break
				}
				if rd.Buffered() == 0 {
					break
				}
			}
			if len(batch) > 0 {
				if !tx.send(batch) {
					break
				}
				batch = nil
			}
		}
	}()
	return tx
}

func runtimeNewBrain(hello *Hello, shared *Shared) *Brain {
	var b *Brain
	if catchUnwind(func() { b = newBrain(hello, shared) }) {
		logMsg("Brain::new panicked")
		return nil
	}
	return b
}

func runtimeSpawnWorker(hello *Hello, shared *Shared, brain *Brain) (*runtimeChan[runtimeJob], *runtimeChan[runtimeDone]) {
	jch := newRuntimeChan[runtimeJob]()
	dch := newRuntimeChan[runtimeDone]()
	go func() {
		defer jch.closeRx()
		defer dch.closeTx()
		for {
			job, ok := jch.recv()
			if !ok {
				break
			}
			for {
				j, ok := jch.tryRecv()
				if !ok {
					break
				}
				job = j
			}
			tick := job.obs.Tick
			if brain == nil {
				brain = runtimeNewBrain(hello, shared)
			}
			if brain == nil {
				dch.send(runtimeDone{kind: runtimeDonePanicked, tick: tick})
				continue
			}
			b := brain
			var a Action
			if !catchUnwind(func() { a = b.Act(job.obs, job.budget) }) {
				a.Tick = tick
				if !dch.send(runtimeDone{kind: runtimeDoneAct, tick: tick, act: a}) {
					break
				}
			} else {
				logMsg(fmt.Sprintf("Brain::act panicked at tick %d", tick))
				dch.send(runtimeDone{kind: runtimeDonePanicked, tick: tick})
				brain = runtimeNewBrain(hello, shared)
			}
		}
	}()
	return jch, dch
}

// ---------------------------------------------------------------- output

type runtimeOut struct {
	maxLine   int
	lastMove  *V2
	lastBoost *bool
	lastAim   *V2
	lastFire  [2]bool
}

func (o *runtimeOut) line(s []byte) {
	os.Stdout.Write(append(s, '\n'))
}

func (o *runtimeOut) action(a Action) {
	o.lastMove = runtimeCopyV2(a.Move)
	o.lastBoost = runtimeCopyBool(a.Boost)
	if a.Aim != nil {
		o.lastAim = runtimeCopyV2(a.Aim)
	}
	if a.Fire != nil {
		o.lastFire = *a.Fire
	} else {
		o.lastFire = [2]bool{false, false}
	}
	s := a.appendJSON(nil)
	if len(s) > o.maxLine && a.Debug != nil {
		a.Debug = nil
		s = a.appendJSON(nil)
	}
	if len(s) == 0 {
		return
	}
	os.Stdout.Write(append(s, '\n'))
}

func (o *runtimeOut) fallback(tick int64) Action {
	return Action{Tick: tick, Move: runtimeCopyV2(o.lastMove), Boost: runtimeCopyBool(o.lastBoost), Aim: runtimeCopyV2(o.lastAim)}
}

func (o *runtimeOut) fallbackHold(tick int64, hold [2]bool) Action {
	a := o.fallback(tick)
	f := [2]bool{o.lastFire[0] && hold[0], o.lastFire[1] && hold[1]}
	if f[0] || f[1] {
		a.Fire = &f
	}
	return a
}

func runtimeCopyV2(p *V2) *V2 {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}

func runtimeCopyBool(p *bool) *bool {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}

// ---------------------------------------------------------------- lenient parsing

// runtimeStripNulls drops null object members and null array elements (except in "weapons").
func runtimeStripNulls(v *jv, key string) {
	switch v.kind {
	case jvObj:
		kept := v.obj[:0]
		for _, p := range v.obj {
			if p.v.kind != jvNull {
				kept = append(kept, p)
			}
		}
		v.obj = kept
		for _, p := range v.obj {
			runtimeStripNulls(p.v, p.k)
		}
	case jvArr:
		if key != "weapons" {
			kept := v.arr[:0]
			for _, x := range v.arr {
				if x.kind != jvNull {
					kept = append(kept, x)
				}
			}
			v.arr = kept
		}
		for _, x := range v.arr {
			runtimeStripNulls(x, "")
		}
	}
}

// runtimeParseLenient parses a server line again with null fields removed.
func runtimeParseLenient(line string) (*ServerMsg, bool) {
	v, err := jvValueFromStr(line)
	if err != nil {
		return nil, false
	}
	runtimeStripNulls(v, "")
	m, err := serverMsgFromJV(v)
	if err != nil {
		return nil, false
	}
	return m, true
}

// runtimeRawObsTick peeks the tick of a line that did not parse as an observation.
func runtimeRawObsTick(line string) (int64, bool) {
	v, err := jvValueFromStr(line)
	if err != nil {
		return 0, false
	}
	ty := v.get("type")
	if ty == nil || ty.kind != jvStr || ty.s != "obs" {
		return 0, false
	}
	t := v.get("tick")
	if t == nil || t.kind != jvNum {
		return 0, false
	}
	return f2i(t.f), true
}

func runtimeParseHello(line string) (*Hello, bool) {
	v, err := jvValueFromStr(line)
	if err != nil {
		return nil, false
	}
	if ty := v.get("type"); ty == nil || ty.kind != jvStr || ty.s != "hello" {
		return nil, false
	}
	h := helloDefault()
	err = jvStructInto(v, &h)
	if err == nil {
		return &h, true
	}
	logMsg(fmt.Sprintf("hello parse error: %v", err))
	h = helloDefault()
	if d := v.get("map").get("dir"); d != nil && d.kind == jvStr {
		h.Map.Dir = d.s
	}
	if s := v.get("slot"); s != nil && s.kind == jvNum {
		h.Slot = f2i(s.f)
	}
	if c := v.get("config"); c != nil {
		cfg := configDefault()
		if jvStructInto(c, &cfg) == nil {
			h.Config = cfg
		}
	}
	return &h, true
}

// ParseHello parses a hello line like the runtime does (with its field-by-field fallback).
func ParseHello(line []byte) (*Hello, bool) {
	return runtimeParseHello(runtimeCleanLine(line))
}

// ParseObs parses an observation line like the runtime does (strict, then without nulls).
func ParseObs(line []byte) (*Obs, bool) {
	s := runtimeCleanLine(line)
	if m, err := serverMsgFromStr(s); err == nil {
		if m.Kind == ServerMsgObs {
			return m.Obs, true
		}
		return nil, false
	}
	if m, ok := runtimeParseLenient(s); ok && m.Kind == ServerMsgObs {
		return m.Obs, true
	}
	return nil, false
}

// ---------------------------------------------------------------- main loop

// (Instant, Result<Box<Obs>, i64>)
type runtimeNewest struct {
	x0    time.Time
	obs   *Obs
	errOk bool // false: obs did not parse, errTick holds its tick
	tick  int64
}

// Run is the process main loop. Returns the exit code.
func Run() int {
	runtimeLog.Store(os.Getenv("MECH_LOG") == "1")
	// Rust ignores SIGPIPE: a closed stdout makes writes fail instead of killing the process.
	signal.Ignore(syscall.SIGPIPE)
	start := time.Now()
	rx := runtimeSpawnReader()
	defer rx.closeRx()

	var pending []runtimeLine
	var hello *Hello
helloLoop:
	for {
		batch, ok := rx.recv()
		if !ok {
			return 0
		}
		for i, ln := range batch {
			if h, ok := runtimeParseHello(ln.x1); ok {
				pending = append(pending, batch[i+1:]...)
				hello = h
				break helloLoop
			}
			logMsg("ignoring line before hello")
		}
	}
	shared := loadShared(hello.Map.Dir, &hello.Config)
	loaded := time.Since(start)
	brain := runtimeNewBrain(hello, shared)
	tickMs := 50.0
	if hello.Config.Bot.TickBudgetMs > 0.0 {
		tickMs = hello.Config.Bot.TickBudgetMs
	}
	ioWait := runtimeDurFromSecsF64(tickMs * runtimeIoFraction / 1000.0)
	brainWait := runtimeDurFromSecsF64(tickMs * runtimeBrainFraction / 1000.0)
	maxLine := 65536
	if hello.Config.Bot.MaxLineBytes > 0.0 {
		maxLine = f2n(hello.Config.Bot.MaxLineBytes)
	}
	jtx, drx := runtimeSpawnWorker(hello, shared, brain)
	defer jtx.closeTx()
	defer drx.closeRx()

	out := &runtimeOut{maxLine: maxLine}
	ready := newReady(botName)
	out.line(ready.appendJSON(nil))
	logMsg(fmt.Sprintf("ready after %v (shared loaded after %v)", time.Since(start), loaded))

	var answered, fallbacks, panics uint64
	for {
		lines := pending
		pending = nil
		if len(lines) == 0 {
			first, ok := rx.recv()
			if !ok {
				break
			}
			lines = first
		}
		for {
			x, ok := rx.tryRecv()
			if !ok {
				break
			}
			lines = append(lines, x...)
		}
		// Newest observation wins; "end" anywhere ends the process.
		var newest *runtimeNewest
		ended := false
		for _, ln := range lines {
			t, line := ln.x0, ln.x1
			msg, err := serverMsgFromStr(line)
			if err == nil {
				switch msg.Kind {
				case ServerMsgObs:
					newest = &runtimeNewest{x0: t, obs: msg.Obs, errOk: true}
				case ServerMsgEnd:
					logMsg(fmt.Sprintf("end: place %d kills %d", msg.End.Place, msg.End.Kills))
					ended = true
				}
				continue
			}
			if m, ok := runtimeParseLenient(line); ok && m.Kind == ServerMsgObs {
				logMsg(fmt.Sprintf("obs parsed leniently at tick %d: %v", m.Obs.Tick, err))
				newest = &runtimeNewest{x0: t, obs: m.Obs, errOk: true}
			} else if tick, ok := runtimeRawObsTick(line); ok {
				logMsg(fmt.Sprintf("obs parse error at tick %d: %v", tick, err))
				newest = &runtimeNewest{x0: t, errOk: false, tick: tick}
			} else if strings.Contains(line, "\"end\"") {
				ended = true
			}
		}
		if ended {
			break
		}
		if newest == nil {
			continue
		}
		tRead := newest.x0
		if !newest.errOk {
			out.action(out.fallback(newest.tick))
			fallbacks++
			continue
		}
		obs := newest.obs
		tick := obs.Tick
		var hold [2]bool
		for i, w := range obs.Me.Weapons {
			hold[i] = w != nil && w.Kind == WeaponKindLaser && w.Charge > 0.0
		}
		deadline := tRead.Add(ioWait)
		if !jtx.send(runtimeJob{obs: obs, budget: budgetUntil(tRead.Add(brainWait))}) {
			out.action(out.fallback(tick))
			fallbacks++
			continue
		}
		var action *Action
		for {
			now := time.Now()
			if !now.Before(deadline) {
				break
			}
			d, st := drx.recvTimeout(deadline.Sub(now))
			if st != runtimeRecvOk {
				break
			}
			if d.kind == runtimeDoneAct && d.tick == tick {
				a := d.act
				action = &a
				break
			}
			if d.kind == runtimeDonePanicked && d.tick == tick {
				panics++
				break
			}
		}
		if action != nil {
			answered++
			out.action(*action)
		} else {
			fallbacks++
			logMsg(fmt.Sprintf("fallback at tick %d", tick))
			out.action(out.fallbackHold(tick, hold))
		}
	}
	logMsg(fmt.Sprintf("exit: answered %d, fallbacks %d, panics %d", answered, fallbacks, panics))
	return 0
}
