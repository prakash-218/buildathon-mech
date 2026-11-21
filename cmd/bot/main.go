// Combined bot: against the house squad (practice) it plays giddygiddyboy's brain (package mech,
// MIT, unmodified); once our own brain decides the match is a free-for-all (3+ mechs died that
// we didn't kill), control switches to our brain (package ours) with its free-for-all profile.
// Both brains see every observation from the start so their state stays current.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	"mechbot/mech"
	"mechbot/ours"
)

const botName = "osu-mech"

func main() {
	lines := make(chan []byte, 256)
	go func() {
		r := bufio.NewReaderSize(os.Stdin, 1<<20)
		for {
			line, err := r.ReadBytes('\n')
			if len(line) > 0 {
				lines <- line
			}
			if err != nil {
				close(lines)
				return
			}
		}
	}()
	out := bufio.NewWriterSize(os.Stdout, 1<<16)
	send := func(v any) {
		b, err := json.Marshal(v)
		if err != nil {
			return
		}
		out.Write(b)
		out.WriteByte('\n')
		out.Flush()
	}
	var gb *mech.Brain
	var ob *ours.Brain
	var last *ours.Action
	useOurs := false
	var probe struct {
		Type string `json:"type"`
	}
	for {
		first, ok := <-lines
		if !ok {
			return
		}
		batch := [][]byte{first}
	drain:
		for {
			select {
			case l, ok := <-lines:
				if !ok {
					break drain
				}
				batch = append(batch, l)
			default:
				break drain
			}
		}
		var latest []byte
		for _, raw := range batch {
			probe.Type = ""
			if json.Unmarshal(raw, &probe) != nil {
				continue
			}
			switch probe.Type {
			case "hello":
				var h mech.Hello
				if err := json.Unmarshal(raw, &h); err != nil {
					fmt.Fprintln(os.Stderr, "hello:", err)
				}
				dir := h.Map.Dir
				if dir == "" {
					dir = "/arena"
				}
				w, err := mech.LoadWorld(dir)
				if err != nil {
					fmt.Fprintln(os.Stderr, "map:", err, "- playing on a flat map")
					w = mech.FlatWorld()
				}
				gb = mech.NewBrain(&h, w)
				var oh ours.Hello
				if err := json.Unmarshal(raw, &oh); err == nil {
					ob = ours.NewBrain(&oh)
				}
				send(map[string]string{"type": "ready", "name": botName})
			case "obs":
				latest = raw
			case "end":
				return
			}
		}
		if latest == nil {
			continue
		}
		var o mech.Obs
		if err := json.Unmarshal(latest, &o); err != nil || gb == nil {
			var t struct {
				Tick int `json:"tick"`
			}
			json.Unmarshal(latest, &t)
			send(mech.Action{Tick: t.Tick})
			continue
		}
		ga := gb.Act(&o)
		var oa *ours.Action
		if ob != nil {
			var oo ours.Obs
			if json.Unmarshal(latest, &oo) == nil {
				oa = ours.SafeAct(ob, &oo, last)
				last = oa
			}
			// our detector (3+ mechs died that we didn't kill) never fired in 48 real practice matches
			if !useOurs && ob.IsFFA() {
				useOurs = true
				fmt.Fprintf(os.Stderr, "hybrid: free-for-all detected at t=%.1f, switching to own brain\n", o.T)
			}
		}
		if useOurs && oa != nil {
			send(oa)
		} else {
			send(ga)
		}
	}
}
