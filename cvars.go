package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// CvarCheck is one rule from the league settings list checked against a demo.
type CvarCheck struct {
	Name  string `json:"name"`
	Value string `json:"value"` // as found in the demo ("" = not recorded)
	Legal string `json:"legal"`
	OK    *bool  `json:"ok"` // nil: the demo can't tell
	Note  string `json:"note,omitempty"`
}

var stuffRe = regexp.MustCompile(`(?:^|;)\s*(fps_max|fps_override|gl_vsync|cl_lw|cl_sidespeed|cl_backspeed|cl_forwardspeed|cl_movespeedkey|developer)\s+"?([0-9.\-]+)"?`)
var cvarMsgRe = regexp.MustCompile(`"?(sv_cheats|sv_gravity|sv_maxspeed|sv_airaccelerate|edgefriction|sv_friction)"? changed to "?([0-9.\-]+)`)

// parseInfo reads svc_serverinfo (our player slot), svc_updateuserinfo,
// svc_stufftext and server cvar change notices from the raw demo.
func parseInfo(dm *Demo, d []byte) {
	defer func() { recover() }()
	slot := -1
	for p := 544; p+40 < len(d) && p < 2000000; p++ {
		if d[p] == 11 && binary.LittleEndian.Uint32(d[p+1:]) == 48 {
			mc, pn := int(d[p+29]), int(d[p+30])
			if mc >= 1 && mc <= 32 && pn < mc {
				slot = pn
				break
			}
		}
	}
	key := []byte("\\cl_lw\\")
	for off := 0; ; {
		i := bytes.Index(d[off:], key)
		if i < 0 {
			break
		}
		i += off
		off = i + 1
		// walk back to the start of the userinfo string (after slot+userid)
		st := i
		for st > 0 && d[st-1] != 0 && i-st < 600 {
			st--
		}
		if st < 6 || d[st-6] != 13 {
			continue
		}
		if slot >= 0 && int(d[st-5]) != slot {
			continue
		}
		end := bytes.IndexByte(d[st:], 0)
		if end < 0 {
			continue
		}
		parts := strings.Split(string(d[st:st+end]), "\\")
		m := map[string]string{}
		for k := 1; k+1 < len(parts); k += 2 {
			m[parts[k]] = parts[k+1]
		}
		dm.UserInfo = m // keep the last one
	}
	dm.Forced = map[string]string{}
	for _, f := range dm.Frames {
		for _, t := range f.Texts {
			for _, m := range stuffRe.FindAllStringSubmatch(t, -1) {
				dm.Forced[m[1]] = m[2]
			}
			for _, m := range cvarMsgRe.FindAllStringSubmatch(t, -1) {
				dm.CvarNotes = append(dm.CvarNotes, m[1]+" "+m[2])
			}
		}
	}
}

func bp(b bool) *bool { return &b }

func fnum(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }

// cvarCheck checks the league settings for the frames from..to (a jump).
func (dm *Demo) cvarCheck(from, to int) []CvarCheck {
	var out []CvarCheck
	// server: movevars are recorded every frame
	seen := map[int]map[float64]bool{}
	for k, m := range dm.MV {
		next := len(dm.Frames)
		if k+1 < len(dm.MV) {
			next = dm.MV[k+1].Frame
		}
		if next <= from || m.Frame > to {
			continue
		}
		for q := 0; q < 12; q++ {
			if seen[q] == nil {
				seen[q] = map[float64]bool{}
			}
			seen[q][math.Round(m.V[q]*1000)/1000] = true
		}
	}
	for q := 0; q < 12; q++ {
		c := CvarCheck{Name: mvNames[q], Legal: fnum(mvLegal[q])}
		if len(seen[q]) == 0 {
			c.Note = "not recorded"
			out = append(out, c)
			continue
		}
		var vals []string
		ok := true
		for v := range seen[q] {
			vals = append(vals, fnum(v))
			if math.Abs(v-mvLegal[q]) > 0.001 {
				ok = false
			}
		}
		c.Value = strings.Join(vals, ", ")
		c.OK = bp(ok)
		out = append(out, c)
	}
	// sv_cheats: only visible if the server announced a change
	c := CvarCheck{Name: "sv_cheats", Legal: "0"}
	for _, n := range dm.CvarNotes {
		if strings.HasPrefix(n, "sv_cheats ") {
			c.Value = strings.TrimPrefix(n, "sv_cheats ")
		}
	}
	if c.Value != "" {
		c.OK = bp(c.Value == "0")
	} else {
		c.Note = "no change announced in the demo"
	}
	out = append(out, c)
	// client
	lw := CvarCheck{Name: "cl_lw", Legal: "1"}
	if v, ok := dm.UserInfo["cl_lw"]; ok {
		lw.Value, lw.OK = v, bp(v == "1")
	} else {
		lw.Note = "not recorded"
	}
	out = append(out, lw)
	// fps: msec of the frames around the jump
	lo, n := 1000, 0
	for k := from; k <= to && k < len(dm.Frames); k++ {
		if k < 0 {
			continue
		}
		if ms := dm.Frames[k].Msec; ms > 0 {
			n++
			if ms < lo {
				lo = ms
			}
		}
	}
	fc := CvarCheck{Name: "fps_max", Legal: "99.5", Note: "from frame times"}
	if v, ok := dm.Forced["fps_max"]; ok {
		fc.Note = "server forced fps_max " + v + "; checked from frame times"
	}
	if n > 0 {
		fc.Value = fmt.Sprintf("max %.0f fps", 1000/float64(lo))
		fc.OK = bp(lo >= 10)
	}
	out = append(out, fc)
	// movement speeds: a key held on its own gives the full cl_*speed, capped by the game at 250
	maxF, maxS := 0.0, 0.0
	for k := from; k <= to && k < len(dm.Frames); k++ {
		if k < 0 {
			continue
		}
		f := dm.Frames[k]
		maxF = math.Max(maxF, math.Abs(f.Forward))
		maxS = math.Max(maxS, math.Abs(f.Side))
	}
	for _, sp := range []struct {
		n string
		v float64
	}{{"cl_forwardspeed / cl_backspeed", maxF}, {"cl_sidespeed", maxS}} {
		c := CvarCheck{Name: sp.n, Legal: "400"}
		if sp.v > 0 {
			c.Value = fmt.Sprintf("%.0f in the demo", sp.v)
			c.OK = bp(sp.v >= 249.5)
			c.Note = "the game caps movement at 250, so 250 means 250 or more (400 is fine)"
		} else {
			c.Note = "key not used"
		}
		out = append(out, c)
	}
	for _, nm := range []string{"cl_movespeedkey", "fps_override", "gl_vsync"} {
		c := CvarCheck{Name: nm, Legal: map[string]string{"cl_movespeedkey": "0.52", "fps_override": "0", "gl_vsync": "0"}[nm]}
		if v, ok := dm.Forced[nm]; ok {
			c.Value = v
			c.OK = bp(v == c.Legal)
			c.Note = "set by the server"
		} else {
			c.Note = "not recorded in demos"
		}
		out = append(out, c)
	}
	return out
}
