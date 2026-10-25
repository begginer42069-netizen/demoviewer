package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math"
)

// Decoder for the parts of the GoldSrc network stream DEMviewer needs:
// delta descriptions, clientdata (to skip it) and packet entities, so brush
// entities (e.g. movable blocks on kz_longjumparena) can be put where they
// really were during a jump.

type bitReader struct {
	d   []byte
	bit int
}

func (b *bitReader) bits(n int) uint32 {
	var v uint32
	for i := 0; i < n; i++ {
		by := b.bit >> 3
		if by >= len(b.d) {
			panic("bitread past end")
		}
		v |= uint32((b.d[by]>>(b.bit&7))&1) << i
		b.bit++
	}
	return v
}
func (b *bitReader) peek(n int) uint32 { s := b.bit; v := b.bits(n); b.bit = s; return v }
func (b *bitReader) bytePos() int      { return (b.bit + 7) >> 3 }
func (b *bitReader) str() string {
	var out []byte
	for {
		c := byte(b.bits(8))
		if c == 0 {
			return string(out)
		}
		out = append(out, c)
	}
}

const (
	dtByte, dtShort, dtFloat, dtInteger, dtAngle, dtTW8, dtTWBig, dtString = 1, 2, 4, 8, 16, 32, 64, 128
	dtSigned                                                               = 0x80000000
)

type deltaField struct {
	name      string
	typ       uint32
	bits      int
	pre, post float64
}

var metaDelta = []deltaField{{"fieldType", dtInteger, 32, 1, 1}, {"fieldName", dtString, 1, 1, 1}, {"fieldOffset", dtInteger, 16, 1, 1},
	{"fieldSize", dtInteger, 8, 1, 1}, {"significant_bits", dtInteger, 8, 1, 1}, {"premultiply", dtFloat, 32, 4000, 1}, {"postmultiply", dtFloat, 32, 4000, 1}}

// readDelta returns changed fields (numbers as float64, strings dropped)
func readDelta(b *bitReader, desc []deltaField, out map[string]float64, strs map[string]string) {
	nb := int(b.bits(3))
	var mask uint64
	for i := 0; i < nb; i++ {
		mask |= uint64(b.bits(8)) << (8 * i)
	}
	for i, f := range desc {
		if i >= 64 || mask>>uint(i)&1 == 0 {
			continue
		}
		t := f.typ &^ dtSigned
		sg := f.typ&dtSigned != 0
		var v float64
		switch t {
		case dtByte, dtShort, dtInteger, dtFloat:
			if sg {
				neg := b.bits(1)
				v = float64(b.bits(f.bits - 1))
				if neg != 0 {
					v = -v
				}
			} else {
				v = float64(b.bits(f.bits))
			}
			if f.pre != 0 && f.pre != 1 {
				v /= f.pre
			}
			if f.post != 1 && f.post != 0 {
				v *= f.post
			}
		case dtAngle:
			v = float64(b.bits(f.bits)) * 360 / float64(uint64(1)<<uint(f.bits))
		case dtTW8:
			v = float64(b.bits(8))
		case dtTWBig:
			v = float64(b.bits(f.bits))
		case dtString:
			s := b.str()
			if strs != nil {
				strs[f.name] = s
			}
			continue
		default:
			panic("bad delta type")
		}
		if out != nil {
			out[f.name] = v
		}
	}
}

func readDeltaDesc(d []byte, pos int) (string, []deltaField, error) {
	e := bytes.IndexByte(d[pos:], 0)
	if e < 0 {
		return "", nil, errors.New("bad delta desc")
	}
	name := string(d[pos : pos+e])
	b := &bitReader{d: d, bit: (pos + e + 1) * 8}
	n := int(b.bits(16))
	var desc []deltaField
	for i := 0; i < n; i++ {
		m := map[string]float64{}
		s := map[string]string{}
		readDelta(b, metaDelta, m, s)
		pre, post := m["premultiply"], m["postmultiply"]
		if pre == 0 {
			pre = 1
		}
		if post == 0 {
			post = 1
		}
		desc = append(desc, deltaField{s["fieldName"], uint32(m["fieldType"]), int(m["significant_bits"]), pre, post})
	}
	return name, desc, nil
}

type entState struct {
	Model  int
	Origin [3]float64
	Angles [3]float64
}

type entUpdate struct {
	Kind  int // 0 update (entity visible), 1 removed, 2 full update: forget all
	Frame int
	Ent   int
	Set   [3]bool
	V     [3]float64
	ASet  [3]bool
	A     [3]float64
	Model int
}

// decodeEntities fills dm.baseline and dm.entUpdates. Best effort: frames that
// can't be decoded are skipped.
func decodeEntities(dm *Demo, raw []byte, msgs [][]byte) {
	defer func() { recover() }()
	descs := map[string][]deltaField{}
	for _, n := range []string{"clientdata_t", "weapon_data_t", "entity_state_t", "entity_state_player_t", "custom_entity_state_t"} {
		key := append([]byte{14}, []byte(n+"\x00")...)
		if i := bytes.Index(raw, key); i >= 0 {
			if _, d, err := readDeltaDesc(raw, i+1); err == nil {
				descs[n] = d
			}
		}
	}
	if len(descs) < 5 {
		return
	}
	maxcl := 32
	nameFor := func(num int, custom bool) string {
		if custom {
			return "custom_entity_state_t"
		}
		if num >= 1 && num <= maxcl {
			return "entity_state_player_t"
		}
		return "entity_state_t"
	}
	// baseline: try each svc_spawnbaseline candidate in the loading part
	lim := len(raw)
	if lim > 600000 {
		lim = 600000
	}
	// maxplayers isn't known here: try each value and keep the parse that reads
	// the most entities (player slots use a different delta layout)
	bestN, bestMax := 0, 32
	var best map[int]entState
	for _, mc := range []int{32, 31, 30, 28, 27, 26, 25, 24, 22, 20, 18, 16, 15, 14, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 29, 23, 21, 19, 17, 13} {
		maxcl = mc
		dm.baseline = nil
		for pos := 544; pos < lim && dm.baseline == nil; pos++ {
			if raw[pos] != 22 {
				continue
			}
			func() {
				defer func() { recover() }()
				b := &bitReader{d: raw, bit: (pos + 1) * 8}
				if b.peek(11) != 0 {
					return
				}
				out := map[int]entState{}
				last := -1
				for {
					num := int(b.bits(11))
					if num == 2047 {
						break
					}
					if num <= last {
						return
					}
					typ := b.bits(2)
					m := map[string]float64{}
					readDelta(b, descs[nameFor(num, typ&2 != 0)], m, nil)
					out[num] = entState{Model: int(m["modelindex"]), Origin: [3]float64{m["origin[0]"], m["origin[1]"], m["origin[2]"]}, Angles: [3]float64{m["angles[0]"], m["angles[1]"], m["angles[2]"]}}
					last = num
				}
				if len(out) > 10 {
					dm.baseline = out
				}
			}()
		}
		if len(dm.baseline) > bestN {
			bestN, bestMax, best = len(dm.baseline), mc, dm.baseline
		}
		if mc == 32 && bestN > 60 {
			break // the usual case
		}
	}
	maxcl, dm.baseline = bestMax, best
	if dm.baseline == nil {
		return
	}
	lastTime := -1.0
	for fi, m := range msgs {
		func() {
			defer func() { recover() }()
			// find svc_time followed by svc_clientdata
			p := -1
			for k := 0; k+6 <= len(m); k++ {
				if m[k] == 7 && m[k+5] == 15 {
					t := float64(math.Float32frombits(binary.LittleEndian.Uint32(m[k+1 : k+5])))
					if lastTime < 0 || (t >= lastTime-0.0001 && t < lastTime+2) {
						p = k
						lastTime = t
						break
					}
				}
			}
			if p < 0 {
				return
			}
			b := &bitReader{d: m, bit: (p + 6) * 8}
			if b.bits(1) != 0 {
				b.bits(8)
			}
			readDelta(b, descs["clientdata_t"], nil, nil)
			for b.bits(1) != 0 {
				b.bits(6)
				readDelta(b, descs["weapon_data_t"], nil, nil)
			}
			q := b.bytePos()
			if q >= len(m) || (m[q] != 40 && m[q] != 41) {
				return
			}
			delta := m[q] == 41
			q += 3
			if delta {
				q++
			}
			b = &bitReader{d: m, bit: q * 8}
			num := 0
			if !delta {
				dm.entUpdates = append(dm.entUpdates, entUpdate{Kind: 2, Frame: fi})
			}
			for {
				if b.peek(16) == 0 {
					break
				}
				remove := false
				if delta {
					remove = b.bits(1) != 0
				} else if b.bits(1) != 0 {
					num++
					goto body
				}
				if b.bits(1) != 0 {
					num = int(b.bits(11))
				} else {
					num += int(b.bits(6))
				}
			body:
				if remove {
					if num > maxcl {
						dm.entUpdates = append(dm.entUpdates, entUpdate{Kind: 1, Frame: fi, Ent: num})
					}
					continue
				}
				custom := b.bits(1) != 0
				if !delta && b.bits(1) != 0 {
					b.bits(6)
				}
				vals := map[string]float64{}
				readDelta(b, descs[nameFor(num, custom)], vals, nil)
				u := entUpdate{Frame: fi, Ent: num, Model: -1}
				for a := 0; a < 3; a++ {
					if v, ok := vals["origin["+string(rune('0'+a))+"]"]; ok {
						u.Set[a], u.V[a] = true, v
					}
				}
				for a := 0; a < 3; a++ {
					if v, ok := vals["angles["+string(rune('0'+a))+"]"]; ok {
						u.ASet[a], u.A[a] = true, v
					}
				}
				if mi, ok := vals["modelindex"]; ok {
					u.Model = int(mi)
				}
				if num > maxcl {
					dm.entUpdates = append(dm.entUpdates, u)
				}
			}
		}()
	}
}

// brushOffsets returns, for every brush model *N, where its entity was at the
// given frame (only models that actually have an entity).
// brushOffsets: for every brush model *N, [origin xyz, angles pitch yaw roll]
func (dm *Demo) brushOffsets(frame int) map[int][6]float64 {
	var out map[int][6]float64
	dm.walkBrushes(frame, frame+1, func(_ int, m map[int][6]float64) { out = m })
	return out
}

// walkBrushes calls fn with the brush model poses at every frame in [from,to).
func (dm *Demo) walkBrushes(from, to int, fn func(frame int, poses map[int][6]float64)) {
	if dm.baseline == nil {
		return
	}
	cur := map[int]entState{}
	for k, v := range dm.baseline {
		cur[k] = v
	}
	present := map[int]bool{}
	seen := false
	snap := func() map[int][6]float64 {
		out := map[int][6]float64{}
		for k, s := range cur {
			if s.Model < 2 {
				continue
			}
			if seen && !present[k] {
				// not sent to the client: out of sight (e.g. a block parked in storage)
				out[s.Model-1] = [6]float64{0, 0, -100000}
				continue
			}
			out[s.Model-1] = [6]float64{s.Origin[0], s.Origin[1], s.Origin[2], s.Angles[0], s.Angles[1], s.Angles[2]}
		}
		return out
	}
	fr := from
	for _, u := range dm.entUpdates {
		for u.Frame > fr && fr < to {
			fn(fr, snap())
			fr++
		}
		if fr >= to {
			return
		}
		switch u.Kind {
		case 2:
			present = map[int]bool{}
			seen = true
			continue
		case 1:
			delete(present, u.Ent)
			continue
		}
		s := cur[u.Ent]
		for a := 0; a < 3; a++ {
			if u.Set[a] {
				s.Origin[a] = u.V[a]
			}
			if u.ASet[a] {
				s.Angles[a] = u.A[a]
			}
		}
		if u.Model >= 0 {
			s.Model = u.Model
		}
		cur[u.Ent] = s
		present[u.Ent] = true
	}
	for ; fr < to; fr++ {
		fn(fr, snap())
	}
}

// brushTrack: for models whose position changes inside [from,to), the
// offset from their pose in ref, per frame.
func (dm *Demo) brushTrack(from, to int, ref map[int][6]float64) map[int][]float32 {
	if ref == nil || to <= from {
		return nil
	}
	n := to - from
	tr := map[int][]float32{}
	dm.walkBrushes(from, to, func(fr int, m map[int][6]float64) {
		k := fr - from
		for mdl, p := range m {
			r, ok := ref[mdl]
			if !ok {
				continue
			}
			dx, dy, dz := p[0]-r[0], p[1]-r[1], p[2]-r[2]
			t := tr[mdl]
			if t == nil {
				if dx == 0 && dy == 0 && dz == 0 {
					continue
				}
				t = make([]float32, 3*n)
				tr[mdl] = t
				// frames before the first change: no offset (already zero)
				_ = k
			}
			t[3*k], t[3*k+1], t[3*k+2] = float32(dx), float32(dy), float32(dz)
		}
	})
	// a track created late missed earlier non-zero frames only if they were
	// zero, so it is complete. Drop models parked far away the whole time.
	for mdl, t := range tr {
		far := true
		for k := 0; k < n; k++ {
			if math.Abs(float64(t[3*k+2])) < 50000 {
				far = false
				break
			}
		}
		if far {
			delete(tr, mdl)
		}
	}
	if len(tr) == 0 {
		return nil
	}
	return tr
}

// rotateBrush applies a brush entity's angles the way GoldSrc draws it:
// roll about X, then -pitch about Y, then yaw about Z.
func rotateBrush(v [3]float64, p, y, r float64) [3]float64 {
	if p == 0 && y == 0 && r == 0 {
		return v
	}
	d := math.Pi / 180
	x0, y0, z0 := v[0], v[1], v[2]
	if r != 0 {
		c, s := math.Cos(r*d), math.Sin(r*d)
		y0, z0 = y0*c-z0*s, y0*s+z0*c
	}
	if p != 0 {
		c, s := math.Cos(-p*d), math.Sin(-p*d)
		x0, z0 = x0*c+z0*s, -x0*s+z0*c
	}
	if y != 0 {
		c, s := math.Cos(y*d), math.Sin(y*d)
		x0, y0 = x0*c-y0*s, x0*s+y0*c
	}
	return [3]float64{x0, y0, z0}
}
