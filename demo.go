package main

import (
	"encoding/binary"
	"errors"
	"math"
	"os"
	"regexp"
	"strings"
)

// Frame is one playback frame of a GoldSrc demo (netmsg frames only).
// Following the demo's own pairing, the velocity/origin stored in a frame
// is the state after the user command stored in the same frame.
type Frame struct {
	Index    int
	Num      int32
	Time     float32
	Origin   [3]float64 // simorg (server origin, 1/128 precision)
	Velocity [3]float64 // simvel (server velocity, truncated to 1/8)
	View     [3]float64 // vieworg (eye position)
	Pitch    float64
	Yaw      float64
	Roll     float64
	OnGround bool
	Msec     int
	Forward  float64
	Side     float64
	Buttons  int
	Texts    []string // printable text found in this frame's server messages
}

type Demo struct {
	Path     string
	Name     string
	Map      string
	Frames   []Frame
	Gravity  float64
	AirAccel float64
	MaxSpeed float64
	Size     int64

	// movement cvars (movevars) as recorded, one entry each time they change
	MV []mvState
	// local player's userinfo (cl_lw ...), cvar values the server forced
	// with stufftext, and server cvar change notices
	UserInfo map[string]string
	// model precache: index -> "models/v_knife.mdl"
	Resources  map[int]string
	PlayerSlot int
	// local player's viewmodel per frame ([frame, model index]; 0 = hidden)
	viewModelUpd   [][2]int
	weaponModelUpd [][3]int // [frame, entity, model index]
	Forced         map[string]string
	CvarNotes      []string

	baseline   map[int]entState
	entUpdates []entUpdate
}

func f32(b []byte) float64 { return float64(math.Float32frombits(binary.LittleEndian.Uint32(b))) }
func i32(b []byte) int32   { return int32(binary.LittleEndian.Uint32(b)) }

var printable = regexp.MustCompile(`[\x01-\x04\n\x20-\x7e]{8,}`)

func LoadDemo(path string) (*Demo, error) {
	d, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(d) < 544 || string(d[:6]) != "HLDEMO" {
		return nil, errors.New("not a GoldSrc demo")
	}
	dm := &Demo{Path: path, Size: int64(len(d))}
	dm.Map = cstr(d[16 : 16+260])
	dirOff := int(i32(d[540:544]))
	if dirOff <= 0 || dirOff+4 > len(d) {
		return nil, errors.New("bad demo directory")
	}
	n := int(i32(d[dirOff : dirOff+4]))
	type seg struct {
		name     string
		off, len int
	}
	var segs []seg
	var msgs [][]byte
	for i := 0; i < n && i < 64; i++ {
		e := dirOff + 4 + i*92
		if e+92 > len(d) {
			break
		}
		segs = append(segs, seg{cstr(d[e+4 : e+68]), int(i32(d[e+84 : e+88])), int(i32(d[e+88 : e+92]))})
	}
	for si, sg := range segs {
		p := sg.off
		for p+9 <= len(d) {
			typ := d[p]
			tm := math.Float32frombits(binary.LittleEndian.Uint32(d[p+1 : p+5]))
			fn := i32(d[p+5 : p+9])
			p += 9
			switch typ {
			case 0, 1:
				if p+468 > len(d) {
					return dm, nil
				}
				b := d[p : p+464]
				ml := int(i32(d[p+464 : p+468]))
				rp := b[4:236]
				fl := func(i int) float64 { return f32(rp[i*4 : i*4+4]) }
				in := func(i int) int32 { return i32(rp[i*4 : i*4+4]) }
				uc := b[236:288]
				if si > 0 || strings.EqualFold(sg.name, "Playback") {
					fr := Frame{Index: len(dm.Frames), Num: fn, Time: tm}
					fr.View = [3]float64{fl(0), fl(1), fl(2)}
					fr.Velocity = [3]float64{fl(22), fl(23), fl(24)}
					fr.Origin = [3]float64{fl(25), fl(26), fl(27)}
					fr.OnGround = in(20) != 0
					fr.Msec = int(uc[2])
					fr.Pitch = f32(uc[4:8])
					fr.Yaw = f32(uc[8:12])
					fr.Roll = f32(uc[12:16])
					fr.Forward = f32(uc[16:20])
					fr.Side = f32(uc[20:24])
					fr.Buttons = int(binary.LittleEndian.Uint16(uc[30:32]))
					var msg []byte
					if p+468+ml <= len(d) && ml > 0 {
						msg = d[p+468 : p+468+ml]
						for _, t := range printable.FindAll(msg, -1) {
							fr.Texts = append(fr.Texts, string(t))
						}
					}
					msgs = append(msgs, msg)
					if len(b) >= 356 {
						var v [13]float64
						for q, o := range mvOffsets {
							if q == 11 { // mp_footsteps is an int
								v[q] = float64(i32(b[288+o : 292+o]))
							} else {
								v[q] = f32(b[288+o : 292+o])
							}
						}
						if len(dm.MV) == 0 || dm.MV[len(dm.MV)-1].V != v {
							dm.MV = append(dm.MV, mvState{Frame: len(dm.Frames), V: v})
						}
					}
					if dm.Gravity == 0 {
						mv := b[288:352]
						dm.Gravity = f32(mv[0:4])
						dm.MaxSpeed = f32(mv[8:12])
						dm.AirAccel = f32(mv[20:24])
					}
					dm.Frames = append(dm.Frames, fr)
				}
				p += 468 + ml
			case 2:
			case 3:
				p += 64
			case 4:
				p += 32
			case 5:
				p = len(d) // end of segment
			case 6:
				p += 84
			case 7:
				p += 8
			case 8:
				if p+8 > len(d) {
					return dm, nil
				}
				l := int(i32(d[p+4 : p+8]))
				p += 8 + l + 16
			case 9:
				l := int(i32(d[p : p+4]))
				p += 4 + l
			default:
				p = len(d)
			}
			if typ == 5 {
				break
			}
		}
	}
	if dm.Gravity == 0 {
		dm.Gravity = 800
	}
	if dm.AirAccel == 0 {
		dm.AirAccel = 10
	}
	if len(dm.Frames) == 0 {
		return nil, errors.New("demo has no playback frames")
	}
	decodeEntities(dm, d, msgs)
	parseInfo(dm, d)
	return dm, nil
}

func cstr(b []byte) string {
	for i, c := range b {
		if c == 0 {
			return string(b[:i])
		}
	}
	return string(b)
}

func hspeed(v [3]float64) float64 { return math.Hypot(v[0], v[1]) }

// speedEst corrects for the server truncating each velocity component to 1/8.
func speedEst(v [3]float64) float64 {
	return math.Hypot(math.Abs(v[0])+1.0/16, math.Abs(v[1])+1.0/16)
}

// movevars_t fields checked against the league rules: name, byte offset in
// movevars_t, legal value
type mvState struct {
	Frame int
	V     [13]float64
}

var mvOffsets = [13]int{0, 4, 8, 16, 20, 24, 28, 32, 36, 48, 52, 64, 12} // last: spectator maxspeed (unused)
var mvNames = [13]string{"sv_gravity", "sv_stopspeed", "sv_maxspeed", "sv_accelerate", "sv_airaccelerate", "sv_wateraccelerate", "sv_friction", "edgefriction", "sv_waterfriction", "sv_stepsize", "sv_maxvelocity", "mp_footsteps", ""}
var mvLegal = [13]float64{800, 75, 320, 5, 10, 10, 4, 2, 1, 18, 2000, 1, 0}

// mvAt returns the movevars in effect at frame k.
func (dm *Demo) mvAt(k int) ([13]float64, bool) {
	var v [13]float64
	ok := false
	for _, m := range dm.MV {
		if m.Frame > k {
			break
		}
		v, ok = m.V, true
	}
	return v, ok
}
