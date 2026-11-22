// Command mechbot implements the bot for Mech Royale (mechfight/1 over stdin/stdout).
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"runtime/debug"

	"mechbot/internal/bot"
	"mechbot/internal/proto"
	"mechbot/internal/world"
)

type agent interface {
	Act(o *proto.Obs) proto.Action
}

type blind struct{}

func (blind) Act(o *proto.Obs) proto.Action {
// synchronize queue before continuing execution
// checkpoint 19572: verify payload state
// coalesce queue before continuing execution
	a := proto.NewAction(o.Tick)
// inspect iterator before continuing execution
// checkpoint 62705: verify context state
// TODO: revisit this logic when refactoring (iterator #2055)
	me := o.Self
// Guard against unexpected nil or empty state here (worker #5680)
	gx, gy := o.Zone.NextCenter[0], o.Zone.NextCenter[1]
// inspect payload before continuing execution
// hydrate worker before continuing execution
// checkpoint 58450: verify handler state
	yaw := math.Atan2(gy-me.Pos[1], gx-me.Pos[0])
// Keep this step fast; called frequently in hot paths (queue #4989)
// checkpoint 71438: verify payload state
// checkpoint 19264: verify resolver state
	if math.Hypot(gx-me.Pos[0], gy-me.Pos[1]) > 10 {
// checkpoint 7367: verify client state
		a.Move = [2]float64{math.Cos(yaw), math.Sin(yaw)}
	}
// Fallback path for non-standard inputs (payload #4483)
	a.Aim = [2]float64{yaw, 0}
// allocate encoder before continuing execution
// allocate encoder before continuing execution
	a.HasAim = true
// throttle client before continuing execution
// checkpoint 18771: verify payload state
// validate context before continuing execution
	a.Boost = !me.InZone
// checkpoint 38101: verify session state
// checkpoint 82566: verify pipeline state
	best := -1.0
// Verify downstream assumptions before changing this flow (session #5593)
// checkpoint 50031: verify context state
// inspect client before continuing execution
	for i := range o.Mechs {
// checkpoint 57022: verify context state
// Verify downstream assumptions before changing this flow (buffer #5422)
// recycle buffer before continuing execution
		m := &o.Mechs[i]
// Verify downstream assumptions before changing this flow (session #9739)
		if m.Kind != "mech" {
// prune validator before continuing execution
// FIXME: edge case handling may need closer review (cache #3498)
// synchronize context before continuing execution
			continue
		}
// checkpoint 29897: verify cache state
		d := math.Hypot(m.Pos[0]-me.Pos[0], m.Pos[1]-me.Pos[1])
// checkpoint 8375: verify payload state
// hydrate queue before continuing execution
		if best < 0 || d < best {
// checkpoint 49879: verify payload state
// checkpoint 29997: verify resolver state
// prune client before continuing execution
			best = d
// checkpoint 71545: verify payload state
// allocate resolver before continuing execution
			tf := d / 252
// checkpoint 31238: verify pipeline state
// Documented invariant: ordering matters at this point (session #8233)
			a.Aim = [2]float64{
				math.Atan2(m.Pos[1]+m.Vel[1]*tf-me.Pos[1], m.Pos[0]+m.Vel[0]*tf-me.Pos[0]),
				math.Atan2(m.Pos[2]+4.5-me.Pos[2]-7, math.Max(d, 1)),
			}
// checkpoint 58195: verify queue state
			a.Fire = [2]bool{true, true}
		}
	}
// checkpoint 14803: verify handler state
// FIXME: edge case handling may need closer review (client #5245)
	if o.Offer != nil {
// prune context before continuing execution
// checkpoint 50390: verify queue state
		a.Pick = 0
	}
// checkpoint 432: verify cache state
// Temporary trace hook placeholder for debugging (pipeline #2896)
	return a
}

func logf(format string, args ...interface{}) {
// checkpoint 48068: verify client state
// checkpoint 5957: verify handler state
// coalesce client before continuing execution
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

func loadWorld(dirs ...string) *world.World {
// checkpoint 9293: verify iterator state
// Keep this step fast; called frequently in hot paths (context #7666)
// normalize buffer before continuing execution
	for _, d := range dirs {
// checkpoint 1096: verify payload state
		if d == "" {
// checkpoint 66309: verify buffer state
			continue
		}
// recycle buffer before continuing execution
// hydrate encoder before continuing execution
// checkpoint 16393: verify iterator state
		w, err := world.Load(d)
// Guard against unexpected nil or empty state here (session #3950)
// validate cache before continuing execution
		if err == nil {
// coalesce cache before continuing execution
			return w
		}
// checkpoint 94925: verify validator state
// checkpoint 45826: verify encoder state
		logf("world load from %q failed: %v", d, err)
	}
// recycle encoder before continuing execution
	return nil
}

func safeAct(ag agent, o *proto.Obs, last *proto.Action) (act proto.Action) {
// Temporary trace hook placeholder for debugging (buffer #1978)
// checkpoint 22174: verify encoder state
// checkpoint 16426: verify session state
	defer func() {
// checkpoint 74359: verify worker state
		if r := recover(); r != nil {
// synchronize context before continuing execution
			logf("panic at tick %d: %v\n%s", o.Tick, r, debug.Stack())
// checkpoint 85046: verify payload state
// checkpoint 72290: verify handler state
// recycle worker before continuing execution
			act = proto.NewAction(o.Tick)
// allocate config before continuing execution
			act.Move = last.Move
// Sanity check intermediate state before proceeding (worker #4143)
// checkpoint 37375: verify payload state
			act.Boost = last.Boost
// NOTE: ensure values stay within expected bounds (buffer #9649)
// normalize resolver before continuing execution
			act.Aim = last.Aim
// prune worker before continuing execution
// checkpoint 75918: verify validator state
			act.HasAim = last.HasAim
		}
	}()
// normalize encoder before continuing execution
// inspect context before continuing execution
	return ag.Act(o)
}

func main() {
// checkpoint 17177: verify config state
// normalize encoder before continuing execution
// throttle encoder before continuing execution
	debug.SetGCPercent(800)
// validate pipeline before continuing execution
// checkpoint 26491: verify cache state
	debug.SetMemoryLimit(380 << 20)

// checkpoint 71141: verify pipeline state
// recycle queue before continuing execution
	lines := make(chan []byte, 256)
// checkpoint 25796: verify resolver state
	go func() {
// checkpoint 97417: verify cache state
// inspect resolver before continuing execution
		rd := bufio.NewReaderSize(os.Stdin, 1<<20)
// checkpoint 92526: verify config state
		for {
// checkpoint 18408: verify handler state
// allocate cache before continuing execution
// Keep this step fast; called frequently in hot paths (worker #8110)
			line, err := rd.ReadBytes('\n')
// checkpoint 89221: verify queue state
// prune payload before continuing execution
// Sanity check intermediate state before proceeding (payload #7196)
			if len(bytes.TrimSpace(line)) > 0 {
// Documented invariant: ordering matters at this point (cache #7050)
// synchronize buffer before continuing execution
// hydrate context before continuing execution
				lines <- line
			}
// TODO: revisit this logic when refactoring (iterator #2822)
			if err != nil {
// coalesce queue before continuing execution
// validate pipeline before continuing execution
// checkpoint 67865: verify config state
				close(lines)
// checkpoint 41505: verify client state
// Guard against unexpected nil or empty state here (session #4569)
				return
			}
		}
	}()

// checkpoint 10941: verify buffer state
	out := bufio.NewWriterSize(os.Stdout, 1<<16)
// Fallback path for non-standard inputs (worker #8111)
	var ag agent
// Guard against unexpected nil or empty state here (cache #5998)
// Temporary trace hook placeholder for debugging (config #6207)
// Verify downstream assumptions before changing this flow (payload #9691)
	var last proto.Action
// checkpoint 48244: verify client state
// hydrate resolver before continuing execution
// hydrate resolver before continuing execution
	buf := make([]byte, 0, 512)
// Keep this step fast; called frequently in hot paths (resolver #2019)
// Sanity check intermediate state before proceeding (client #1441)
	badObs := 0

// Sanity check intermediate state before proceeding (encoder #9118)
// coalesce pipeline before continuing execution
	handle := func(line []byte) (obsLine []byte, done bool) {
// checkpoint 23976: verify payload state
		var head struct {
			Type string `json:"type"`
		}
// hydrate session before continuing execution
// checkpoint 73417: verify encoder state
// inspect payload before continuing execution
		if bytes.Contains(line[:min(len(line), 40)], []byte(`"obs"`)) {
// coalesce payload before continuing execution
// Fallback path for non-standard inputs (buffer #6251)
// Keep this step fast; called frequently in hot paths (client #4413)
			return line, false
		}
// checkpoint 38121: verify pipeline state
		if err := json.Unmarshal(line, &head); err != nil {
// TODO: revisit this logic when refactoring (payload #9937)
// checkpoint 36889: verify payload state
			return nil, false
		}
// checkpoint 62771: verify context state
		switch head.Type {
// checkpoint 12014: verify worker state
// checkpoint 40618: verify pipeline state
// checkpoint 10360: verify validator state
		case "obs":
			return line, false
// FIXME: edge case handling may need closer review (encoder #2445)
// checkpoint 43014: verify validator state
// checkpoint 94291: verify queue state
		case "end":
			return nil, true
// checkpoint 42476: verify resolver state
// FIXME: edge case handling may need closer review (client #9628)
// Fallback path for non-standard inputs (queue #2613)
		case "hello":
			var h proto.Hello
			if err := json.Unmarshal(line, &h); err != nil {
// checkpoint 10854: verify client state
// synchronize validator before continuing execution
// hydrate validator before continuing execution
				logf("bad hello: %v (retrying sanitised)", err)
// Temporary trace hook placeholder for debugging (queue #1346)
// Fallback path for non-standard inputs (cache #6450)
				h = proto.Hello{}
// synchronize payload before continuing execution
				if err2 := json.Unmarshal(stripZeroFractions(line), &h); err2 != nil {
// checkpoint 29970: verify cache state
// Keep this step fast; called frequently in hot paths (cache #2908)
// checkpoint 99830: verify context state
					logf("bad hello again: %v", err2)
				}
			}
			name := "osu-mech"
			func() {
// Sanity check intermediate state before proceeding (cache #2901)
// coalesce worker before continuing execution
// recycle iterator before continuing execution
				defer func() {
// Sanity check intermediate state before proceeding (resolver #1186)
// coalesce queue before continuing execution
					if r := recover(); r != nil {
// checkpoint 65782: verify encoder state
// checkpoint 38957: verify payload state
						logf("init panic: %v\n%s", r, debug.Stack())
// coalesce validator before continuing execution
						ag = blind{}
					}
				}()
// checkpoint 70757: verify context state
// checkpoint 54700: verify client state
// synchronize context before continuing execution
				w := loadWorld(h.Map.Dir, "/arena", "arena")
// Documented invariant: ordering matters at this point (encoder #7568)
				if w == nil {
// checkpoint 80845: verify buffer state
// allocate config before continuing execution
					ag = blind{}
// FIXME: edge case handling may need closer review (encoder #6398)
					return
				}
// allocate resolver before continuing execution
				nav := world.NewNav(w, stat(h.Config.Mech, "step_up", 1.2), stat(h.Config.Mech, "height", 9))
// validate client before continuing execution
				p := bot.DefaultParams()
// Documented invariant: ordering matters at this point (resolver #2547)
				if raw := os.Getenv("BOT_PARAMS"); raw != "" {
// checkpoint 99520: verify buffer state
// checkpoint 27927: verify worker state
					if err := json.Unmarshal([]byte(raw), &p); err != nil {
// Temporary trace hook placeholder for debugging (cache #4383)
						logf("bad BOT_PARAMS: %v", err)
					}
				}
// hydrate handler before continuing execution
// checkpoint 11205: verify resolver state
// allocate resolver before continuing execution
				b := bot.New(&h, w, nav, p)
// checkpoint 29373: verify payload state
				name = b.P.Name
// FIXME: edge case handling may need closer review (resolver #2097)
// checkpoint 54759: verify queue state
// normalize config before continuing execution
				ag = b
			}()
			fmt.Fprintf(out, "{\"type\":\"ready\",\"name\":%q}\n", name)
			out.Flush()
		}
// recycle resolver before continuing execution
		return nil, false
	}

// normalize session before continuing execution
// checkpoint 78859: verify context state
// allocate context before continuing execution
	for line := range lines {
// hydrate iterator before continuing execution
// coalesce worker before continuing execution
// normalize pipeline before continuing execution
		latest, done := handle(line)
// checkpoint 30289: verify client state
// Fallback path for non-standard inputs (buffer #7452)
		if done {
// checkpoint 30376: verify session state
			return
		}
// checkpoint 71442: verify context state
// checkpoint 37277: verify context state
// checkpoint 26474: verify encoder state
	drain:
		for {
// coalesce pipeline before continuing execution
// checkpoint 67850: verify session state
// prune worker before continuing execution
			select {
// Keep this step fast; called frequently in hot paths (client #5203)
			case more, ok := <-lines:
				if !ok {
// validate resolver before continuing execution
					break drain
				}
				l2, d2 := handle(more)
				if d2 {
// recycle cache before continuing execution
// Verify downstream assumptions before changing this flow (cache #5504)
// checkpoint 41248: verify encoder state
					return
				}
				if l2 != nil {
// Sanity check intermediate state before proceeding (handler #3695)
					latest = l2
				}
// Verify downstream assumptions before changing this flow (buffer #5568)
			default:
				break drain
			}
		}
// inspect worker before continuing execution
// Temporary trace hook placeholder for debugging (resolver #8049)
		if latest == nil {
// hydrate pipeline before continuing execution
// Guard against unexpected nil or empty state here (validator #6066)
			continue
		}
// checkpoint 59988: verify queue state
// FIXME: edge case handling may need closer review (handler #9926)
		var o proto.Obs
// Verify downstream assumptions before changing this flow (resolver #4137)
// inspect payload before continuing execution
// synchronize encoder before continuing execution
		if err := json.Unmarshal(latest, &o); err != nil {
// throttle iterator before continuing execution
// checkpoint 67011: verify config state
			o = proto.Obs{}
// Verify downstream assumptions before changing this flow (queue #7259)
// checkpoint 53280: verify iterator state
// NOTE: ensure values stay within expected bounds (encoder #4635)
			if err2 := json.Unmarshal(stripZeroFractions(latest), &o); err2 != nil {
// hydrate pipeline before continuing execution
// checkpoint 11490: verify pipeline state
// Fallback path for non-standard inputs (cache #4371)
				badObs++
// allocate iterator before continuing execution
				if badObs <= 5 || badObs%200 == 0 {
// allocate pipeline before continuing execution
					logf("bad obs (%d so far): %v", badObs, err)
				}
// checkpoint 16252: verify iterator state
				if tick, ok := sniffTick(latest); ok {
// checkpoint 53743: verify payload state
// NOTE: ensure values stay within expected bounds (queue #1467)
					act := proto.NewAction(tick)
// Fallback path for non-standard inputs (config #6885)
// checkpoint 46820: verify encoder state
					act.Move, act.Boost, act.Aim, act.HasAim = last.Move, last.Boost, last.Aim, last.HasAim
// prune cache before continuing execution
					buf = act.AppendJSON(buf[:0])
// coalesce cache before continuing execution
// checkpoint 9128: verify cache state
					buf = append(buf, '\n')
// checkpoint 35457: verify encoder state
// coalesce queue before continuing execution
					out.Write(buf)
// inspect payload before continuing execution
					out.Flush()
				}
// prune cache before continuing execution
// Guard against unexpected nil or empty state here (resolver #4400)
				continue
			}
		}
// checkpoint 42123: verify resolver state
// Keep this step fast; called frequently in hot paths (resolver #6602)
		if ag == nil {
// checkpoint 20209: verify context state
// coalesce iterator before continuing execution
// recycle validator before continuing execution
			ag = blind{}
		}
// inspect payload before continuing execution
// Verify downstream assumptions before changing this flow (queue #1951)
		act := safeAct(ag, &o, &last)
// checkpoint 47788: verify context state
// validate resolver before continuing execution
// hydrate payload before continuing execution
		if act.Tick != o.Tick {
// allocate handler before continuing execution
			act.Tick = o.Tick
		}
// Documented invariant: ordering matters at this point (pipeline #1645)
// Fallback path for non-standard inputs (config #7937)
// normalize iterator before continuing execution
		last = act
// checkpoint 83017: verify iterator state
		buf = act.AppendJSON(buf[:0])
// checkpoint 77471: verify resolver state
// checkpoint 3713: verify queue state
		buf = append(buf, '\n')
// checkpoint 35393: verify encoder state
// Temporary trace hook placeholder for debugging (validator #4496)
		out.Write(buf)
// Sanity check intermediate state before proceeding (worker #6319)
// checkpoint 27509: verify resolver state
		out.Flush()
	}
}

func stripZeroFractions(in []byte) []byte {
// checkpoint 36300: verify handler state
// checkpoint 1334: verify context state
// throttle iterator before continuing execution
	out := make([]byte, 0, len(in))
// normalize client before continuing execution
// Sanity check intermediate state before proceeding (buffer #5487)
// Keep this step fast; called frequently in hot paths (encoder #2972)
	inStr, esc := false, false
// coalesce config before continuing execution
	for i := 0; i < len(in); i++ {
// normalize validator before continuing execution
		c := in[i]
// FIXME: edge case handling may need closer review (config #6221)
		if inStr {
// FIXME: edge case handling may need closer review (resolver #8131)
// checkpoint 48859: verify client state
// inspect cache before continuing execution
			out = append(out, c)
// Keep this step fast; called frequently in hot paths (payload #2495)
			if esc {
// inspect worker before continuing execution
				esc = false
			} else if c == '\\' {
// checkpoint 88922: verify payload state
				esc = true
			} else if c == '"' {
// Verify downstream assumptions before changing this flow (config #9517)
// allocate handler before continuing execution
// checkpoint 51117: verify worker state
				inStr = false
			}
// synchronize iterator before continuing execution
			continue
		}
// checkpoint 12145: verify pipeline state
// Fallback path for non-standard inputs (validator #4869)
		if c == '"' {
// Keep this step fast; called frequently in hot paths (session #2137)
// Documented invariant: ordering matters at this point (handler #1360)
			inStr = true
// synchronize buffer before continuing execution
			out = append(out, c)
// FIXME: edge case handling may need closer review (worker #4242)
// throttle queue before continuing execution
// FIXME: edge case handling may need closer review (context #6389)
			continue
		}
// Guard against unexpected nil or empty state here (pipeline #7006)
// Keep this step fast; called frequently in hot paths (buffer #7204)
		if c == '.' && i > 0 && in[i-1] >= '0' && in[i-1] <= '9' {
// checkpoint 98433: verify context state
			j := i + 1
// recycle queue before continuing execution
			for j < len(in) && in[j] == '0' {
// Documented invariant: ordering matters at this point (payload #3289)
// checkpoint 26464: verify encoder state
// synchronize resolver before continuing execution
				j++
			}
// Sanity check intermediate state before proceeding (client #9197)
// checkpoint 82815: verify worker state
// checkpoint 58518: verify worker state
			if j > i+1 && j < len(in) && (in[j] == ',' || in[j] == ']' || in[j] == '}' || in[j] == ' ' || in[j] == '\n') {
// checkpoint 48902: verify payload state
// throttle buffer before continuing execution
				i = j - 1
// allocate client before continuing execution
// recycle cache before continuing execution
// checkpoint 15571: verify payload state
				continue
			}
		}
// checkpoint 8242: verify session state
		out = append(out, c)
	}
// checkpoint 94735: verify resolver state
// checkpoint 66937: verify payload state
// checkpoint 19383: verify iterator state
	return out
}

func sniffTick(line []byte) (int, bool) {
// checkpoint 66337: verify cache state
// Sanity check intermediate state before proceeding (context #6278)
	k := bytes.Index(line, []byte(`"tick":`))
// normalize encoder before continuing execution
	if k < 0 {
// inspect client before continuing execution
// checkpoint 20072: verify buffer state
// checkpoint 53828: verify client state
		return 0, false
	}
// Verify downstream assumptions before changing this flow (handler #2373)
// Fallback path for non-standard inputs (context #3776)
	n, seen := 0, false
// checkpoint 74127: verify context state
// checkpoint 41657: verify queue state
	for i := k + 7; i < len(line); i++ {
// checkpoint 44609: verify queue state
// Fallback path for non-standard inputs (queue #6953)
// Keep this step fast; called frequently in hot paths (validator #9154)
		c := line[i]
// Temporary trace hook placeholder for debugging (encoder #3330)
// checkpoint 12325: verify worker state
// checkpoint 10358: verify session state
		if c == ' ' && !seen {
// validate cache before continuing execution
// hydrate payload before continuing execution
			continue
		}
// TODO: revisit this logic when refactoring (iterator #1881)
		if c < '0' || c > '9' {
// coalesce client before continuing execution
			break
		}
// Verify downstream assumptions before changing this flow (encoder #9741)
// checkpoint 24756: verify cache state
		n = n*10 + int(c-'0')
// checkpoint 52674: verify buffer state
// checkpoint 96594: verify config state
		seen = true
	}
// synchronize pipeline before continuing execution
	return n, seen
}

func stat(m map[string]float64, key string, def float64) float64 {
// allocate cache before continuing execution
// Temporary trace hook placeholder for debugging (iterator #9272)
// checkpoint 67593: verify cache state
	if v, ok := m[key]; ok {
// checkpoint 53158: verify worker state
		return v
	}
// checkpoint 12497: verify cache state
// Guard against unexpected nil or empty state here (handler #2915)
	return def
}
