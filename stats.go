package main

import (
	"math"
	"regexp"
	"strings"
)

type Strafe struct {
	Key      string  `json:"key"`
	Gain     float64 `json:"gain"`
	Loss     float64 `json:"loss"`
	Frames   int     `json:"frames"`
	Good     int     `json:"good"`
	Eff      float64 `json:"eff"`
	Overlaps int     `json:"ovl"`
	Dead     int     `json:"dead"`   // dead airtime: frames with no strafe key held, counted after this strafe
	AvgEff   float64 `json:"avgEff"` // GC: mouse turn vs ideal turn, average over the strafe
	MaxEff   float64 `json:"maxEff"` // GC: best single frame
	effSum   float64
	effN     int
	start    int
}

// turnEff is GC's per-frame efficiency: how far the mouse turned this frame
// compared with the turn that keeps the ideal strafe angle at this speed.
func turnEff(F []Frame, k int, airAccel float64) (float64, bool) {
	if k < 1 {
		return 0, false
	}
	v := speedEst(F[k-1].Velocity)
	ft := float64(F[k].Msec) / 1000
	if v < 1 || ft <= 0 {
		return 0, false
	}
	w := 250.0
	if F[k].Buttons&4 != 0 {
		w *= 0.333
	}
	a := airAccel * w * ft
	ca := (math.Min(w, 30) - a) / v
	ca = math.Max(-1, math.Min(1, ca))
	sa := math.Sqrt(1 - ca*ca)
	opt := math.Atan2(a*sa, v+a*ca) * 180 / math.Pi
	if opt <= 0 {
		return 0, false
	}
	return 100 * math.Abs(wrap180(F[k].Yaw-F[k-1].Yaw)) / opt, true
}

type ChartPoint struct {
	F    int     `json:"f"`    // frame relative to take-off (0 = first air frame)
	Fwd  float64 `json:"fwd"`  // distance along jump direction from take-off
	Side float64 `json:"side"` // sideways drift
	H    float64 `json:"h"`    // feet height above take-off floor
	Spd  float64 `json:"spd"`
	Turn float64 `json:"dy"` // yaw change this frame (deg)
	Air  bool    `json:"air"`
	Keys int     `json:"keys"`
}

type PluginLine struct {
	Source string `json:"src"`
	Text   string `json:"text"`
}

type Jump struct {
	ID         int          `json:"id"`
	Demo       string       `json:"demo"`
	Map        string       `json:"map"`
	MapFound   bool         `json:"mapFound"`
	Type       string       `json:"type"`
	PluginType string       `json:"pluginType"`
	Style      string       `json:"style"` // "bw" backwards, "sw" sideways, "" forwards
	Main       bool         `json:"main"`  // listed by default (landed, valid stats)
	Time       float64      `json:"time"`  // seconds into the demo
	JumpFrame  int          `json:"jumpFrame"`
	LandFrame  int          `json:"landFrame"`
	Landed     bool         `json:"landed"`
	Result     string       `json:"result"`
	Distance   float64      `json:"dist"`
	Pre        float64      `json:"pre"`
	FOG        int          `json:"fog"`
	DeadAir    int          `json:"deadAir"` // frames in the air with no strafe key (W/A/S/D) held
	MaxSpeed   float64      `json:"max"`
	Gain       float64      `json:"gainTotal"`
	Strafes    []Strafe     `json:"strafes"`
	Eff        float64      `json:"eff"`
	Jumpoff    *float64     `json:"jumpoff"`
	Block      *float64     `json:"block"`
	LandEdge   *float64     `json:"landEdge"` // >0 landing reserve, <0 lasting (short)
	Height     float64      `json:"height"`
	Veer       float64      `json:"veer"`
	AirFrames  int          `json:"airFrames"`
	Axis       int          `json:"axis"`
	Sign       float64      `json:"sign"`
	Chart      []ChartPoint `json:"chart"`
	Plugin     []PluginLine `json:"plugin"`
	Cvars      []CvarCheck  `json:"cvars"`
	SlapTotal  int          `json:"slaps"`
	SlapOne    int          `json:"oneFrameSlaps"`
	SlapScore  float64      `json:"slapScore"`
	Replay     *Replay      `json:"replay,omitempty"`
	TakeoffPos [3]float64   `json:"takeoff"`
	EdgeFwd    *float64     `json:"edgeFwd"`
	FogBase    []FogPoint   `json:"fogBase"`
	FaceFwd    *float64     `json:"faceFwd"`
	Dir        [2]float64   `json:"dir"`

	// internals for the what-if simulator
	takeoff   [3]float64
	preVel    [3]float64
	jumpVz    float64
	cmdFrom   int
	tkEdge    float64
	landFace  float64
	landTop   float64
	landZ     float64 // origin height where the distance was measured
	bhopNext  bool    // jumped again right after landing (take-off of a bhop)
	hasEdges  bool
	duckedTO  bool
	baseFrame int
	ladder    bool
}

// FogPoint is the real speed/position the player had on ground frame N
// before the jump (N=1: first frame on the ground).
type FogPoint struct {
	Pre     float64  `json:"pre"`
	Jumpoff *float64 `json:"jumpoff"`
	frame   int
}

type Replay struct {
	Map  string             `json:"map"`
	Ents map[int][6]float64 `json:"ents,omitempty"`
	// brush models that move during the replay (e.g. bhop blocks sinking):
	// per replay frame x,y,z offset from their pose in Ents
	Moves map[int][]float32 `json:"moves,omitempty"`
	D     []float64         `json:"d"`    // per frame: eye xyz, pitch, yaw, roll, buttons, speed, onground
	Duck  []int8            `json:"duck"` // per frame: body ducked (duck state)
	// player model ("gign") and weapon models as [replay frame, "v_knife"]
	// changes; "" = nothing shown (e.g. viewmodel hidden with /hide)
	Player string   `json:"player,omitempty"`
	VModel [][2]any `json:"vmodel,omitempty"`
	PModel [][2]any `json:"pmodel,omitempty"`
	T0     int      `json:"t0"`
	T1     int      `json:"t1"`
}

func eyeOfs(f *Frame) float64 { return f.View[2] - f.Origin[2] }

// ducked: duck key held (ducking is instant in the air) or the eye already
// at duck height (the view eases between 17 and 12 while ducking)
func isDucked(f *Frame) bool { return f.Buttons&4 != 0 || eyeOfs(f) < 14.5 }

func wrap180(a float64) float64 {
	for a > 180 {
		a -= 360
	}
	for a < -180 {
		a += 360
	}
	return a
}

var colorCodes = regexp.MustCompile(`[\x01-\x04]`)

// FindJumps finds every long jump-like jump in a demo and computes its stats.
func FindJumps(dm *Demo, mg *MapGeo) []*Jump {
	F := dm.Frames
	var out []*Jump
	g := dm.Gravity
	for i := 2; i < len(F)-10; i++ {
		ladder := false
		if !(F[i-1].OnGround && !F[i].OnGround && F[i].Velocity[2] > 200) {
			if !isLadderExit(F, i, g) {
				continue
			}
			ladder = true
		}
		j := analyse(dm, mg, i, g, ladder)
		if j != nil {
			j.ID = len(out)
			out = append(out, j)
		}
	}
	return out
}

// isLadderExit: frame i-1 was on a ladder (in the air but no gravity) and
// from frame i normal gravity applies again while climbing (ladder jump).
func isLadderExit(F []Frame, i int, g float64) bool {
	if i < 3 || F[i-2].OnGround || F[i-1].OnGround || F[i].OnGround {
		return false
	}
	if i < 5 || !gravityStep(F, i, g) || F[i-1].Velocity[2] <= 100 {
		return false
	}
	// really on a ladder: several frames in a row in the air without gravity
	for m := 1; m <= 3; m++ {
		if F[i-m].OnGround || gravityStep(F, i-m, g) {
			return false
		}
	}
	return true
}

// airTicks: how many server ticks the first air frame already contains (the
// demo can skip ticks). Worked out from how high the player has risen.
func airTicks(F []Frame, i int, g float64) int {
	if i < 1 {
		return 1
	}
	ft := float64(F[i].Msec) / 1000
	rise := F[i].Origin[2] - F[i-1].Origin[2]
	vz := F[i].Velocity[2]
	best, bd := 1, 1e9
	for n := 1; n <= 4; n++ {
		nf := float64(n)
		if d := math.Abs(ft*(nf*vz+g*ft*nf*nf/2) - rise); d < bd {
			best, bd = n, d
		}
	}
	return best
}

// gravityStep: between frame k-1 and k the vertical speed changed by gravity
// alone (free flight). On a ladder there is no gravity.
func gravityStep(F []Frame, k int, g float64) bool {
	gv := -g * float64(F[k].Msec) / 1000
	d := F[k].Velocity[2] - F[k-1].Velocity[2]
	// demos sometimes repeat a frame (nothing changes) or skip server ticks
	// (two or three ticks of gravity in one frame): both are still free flight
	if d == 0 && F[k].Origin == F[k-1].Origin && math.Hypot(F[k].Velocity[0], F[k].Velocity[1]) > 1 {
		return true
	}
	for m := 1.0; m <= 3; m++ {
		if math.Abs(d-gv*m) < 1 {
			return true
		}
	}
	return false
}

// grabsLadder: somewhere in the air phase (from..to) the player catches a
// ladder (two frames in a row without gravity) - the jump ends there and
// isn't a stat jump.
func grabsLadder(F []Frame, from, to int, g float64) bool {
	for k := from + 1; k < to && k < len(F); k++ {
		if F[k].OnGround || F[k-1].OnGround {
			continue
		}
		if !gravityStep(F, k, g) && k+1 < len(F) && !F[k+1].OnGround && !gravityStep(F, k+1, g) {
			return true
		}
	}
	return false
}

func analyse(dm *Demo, mg *MapGeo, i int, g float64, ladder bool) *Jump {
	F := dm.Frames
	T := F[i-1].Origin
	if !ladder {
		// the take-off point is where the player was on the jump tick: one
		// tick before the first air frame. The demo can skip that server
		// tick, so step back from the first air frame instead of trusting
		// the last ground frame.
		ft := float64(F[i].Msec) / 1000
		step := math.Hypot(F[i].Velocity[0], F[i].Velocity[1]) * ft
		if step > 0.5 && math.Hypot(F[i].Origin[0]-T[0], F[i].Origin[1]-T[1]) > 1.5*step {
			n := float64(airTicks(F, i, g))
			T[0] = F[i].Origin[0] - F[i].Velocity[0]*ft*n
			T[1] = F[i].Origin[1] - F[i].Velocity[1]*ft*n
		}
	}
	V := F[i].Velocity
	j := &Jump{ladder: ladder, Plugin: []PluginLine{}, Strafes: []Strafe{}, FogBase: []FogPoint{}, Demo: dm.Name, Map: dm.Map, JumpFrame: i, LandFrame: -1, takeoff: T, preVel: F[i-1].Velocity, cmdFrom: i, baseFrame: i - 1}
	j.duckedTO = isDucked(&F[i-1])
	j.Time = float64(F[i].Time - F[0].Time)
	var ovMap map[int][6]float64
	if mg != nil {
		ovMap = dm.brushOffsets(i)
	}
	// on the ground the body height says whether we're ducked (the camera eases, the hull doesn't)
	groundFeet := func(f *Frame, fallbackDucked bool) (float64, bool) {
		if mg != nil && f.OnGround {
			d := mg.floorAt(f.Origin[0], f.Origin[1], f.Origin[2]-18.03125, ovMap)
			st := mg.floorAt(f.Origin[0], f.Origin[1], f.Origin[2]-36.03125, ovMap)
			if d && !st {
				return f.Origin[2] - 18, true
			}
			if st && !d {
				return f.Origin[2] - 36, false
			}
		}
		if fallbackDucked {
			return f.Origin[2] - 18, true
		}
		return f.Origin[2] - 36, false
	}
	tkFeet, dk := groundFeet(&F[i-1], j.duckedTO)
	j.duckedTO = dk
	j.jumpVz = V[2] + (float64(airTicks(F, i, g))-0.5)*g*float64(F[i].Msec)/1000
	if ladder {
		// no jump: leaving the ladder keeps the climbing speed
		j.jumpVz = F[i-1].Velocity[2] - 0.5*g*float64(F[i].Msec)/1000
	}
	// landing frame; a teleport (e.g. back to the start after falling) ends
	// the jump as a fail
	L := -1
	tpEnd := len(F)
	for k := i + 1; k < len(F); k++ {
		ft := float64(F[k].Msec) / 1000
		d := math.Hypot(F[k].Origin[0]-F[k-1].Origin[0], F[k].Origin[1]-F[k-1].Origin[1])
		if d > 3*math.Hypot(F[k-1].Velocity[0], F[k-1].Velocity[1])*ft+24 || math.Abs(F[k].Origin[2]-F[k-1].Origin[2]) > 64 {
			tpEnd = k
			break
		}
		if k >= i+5 && F[k].OnGround {
			L = k
			break
		}
	}
	if L < 0 {
		L2 := tpEnd
		if grabsLadder(F, i, L2, g) {
			return nil
		}
	} else if grabsLadder(F, i, L, g) {
		return nil
	}
	// crossing point
	cross := func(k int, thr float64) [2]float64 {
		A := F[k-1].Origin
		av := F[k-1].Velocity
		ft := float64(F[k].Msec) / 1000
		// the demo can skip server ticks: step tick by tick from the last
		// recorded frame until the height is crossed
		ticks := 1
		if hv := math.Hypot(av[0], av[1]) * ft; hv > 0.1 {
			d := math.Hypot(F[k].Origin[0]-A[0], F[k].Origin[1]-A[1])
			ticks = int(math.Max(1, math.Min(4, math.Round(d/hv))))
			if F[k].OnGround && d < hv*0.5 {
				ticks = 1
			}
		}
		x, y, z, vz := A[0], A[1], A[2], av[2]
		for m := 0; m < ticks; m++ {
			v := vz - 0.5*g*ft
			if v < 0 && z+v*ft <= thr || m == ticks-1 {
				fz := 0.0
				if v < 0 {
					fz = (z - thr) / (-v * ft)
				}
				fz = math.Max(0, math.Min(1, fz))
				return [2]float64{x + av[0]*ft*fz, y + av[1]*ft*fz}
			}
			z += v * ft
			vz -= g * ft
			x += av[0] * ft
			y += av[1] * ft
		}
		return [2]float64{x, y}
	}
	var cx [2]float64
	crossK := -1
	landed := false
	if L > 0 {
		feetL, _ := groundFeet(&F[L], isDucked(&F[L]))
		if feetL >= tkFeet-4 || (ladder && feetL >= tkFeet-8) {
			landed = true
			cx = cross(L, F[L].Origin[2])
			crossK = L
			j.landTop = feetL
			j.landZ = F[L].Origin[2]
			for k := L + 1; k < len(F) && k <= L+6; k++ {
				if !F[k].OnGround && F[k-1].OnGround && F[k].Velocity[2] > 150 {
					j.bhopNext = true
					break
				}
			}
		}
	}
	if !landed {
		lim := tpEnd
		if L > 0 {
			lim = L + 1
		}
		// fails are measured where the ducked hull reaches the take-off floor;
		// if the player never ducks in the air, at standing height instead
		duckedAir := false
		for k := i + 1; k < lim; k++ {
			if F[k].Buttons&4 != 0 || isDucked(&F[k]) {
				duckedAir = true
				break
			}
			if F[k].Origin[2] <= tkFeet+18 {
				break
			}
		}
		base := tkFeet
		if ladder && mg != nil {
			// ladder fails are measured at the height of the ladder's platform
			if z, ok := mg.ladderFloor(T[0], T[1], tkFeet, ovMap); ok {
				base = z
			}
		}
		for k := i + 5; k < lim; k++ {
			thr := base + 18
			if !duckedAir {
				thr = base + 36
			}
			if F[k-1].Origin[2] > thr && F[k].Origin[2] <= thr {
				cx = cross(k, thr)
				crossK = k
				break
			}
		}
		j.landTop = base
		j.landZ = base + 18
		if !duckedAir {
			j.landZ = base + 36
		}
	}
	if crossK < 0 {
		return nil
	}
	j.Landed = landed
	j.LandFrame = crossK
	j.Distance = math.Hypot(cx[0]-T[0], cx[1]-T[1]) + 32
	if ladder {
		j.Distance -= 32 // ladder jumps are measured from the ladder, no player box added
	}
	j.AirFrames = crossK - i + 1
	if (!ladder && j.Distance < 180) || (ladder && j.Distance < 150) || j.AirFrames < 40 {
		return nil
	}
	// axis / direction
	axis, s := 0, 1.0
	if math.Abs(V[1]) > math.Abs(V[0]) {
		axis = 1
	}
	if V[axis] < 0 {
		s = -1
	}
	j.Axis, j.Sign = axis, s
	j.TakeoffPos = T
	if hv := math.Hypot(V[0], V[1]); hv > 0 {
		j.Dir = [2]float64{V[0] / hv, V[1] / hv}
	}
	oth := 1 - axis
	j.Veer = math.Abs(cx[oth] - T[oth])
	// pre / fog
	j.Pre = speedEst(F[i-1].Velocity)
	for k := i - 1; k >= 0 && F[k].OnGround; k-- {
		j.FOG++
	}
	// end of flight for strafe stats: wall hit or landing
	end := crossK
	if L > 0 && L > end {
		end = L
	}
	for k := i + 1; k < end; k++ {
		if hspeed(F[k].Velocity) < hspeed(F[k-1].Velocity)-15 {
			end = k
			break
		}
	}
	if !landed && end > crossK+5 {
		end = crossK + 5
	}
	for k := i; k < end; k++ {
		if s := hspeed(F[k].Velocity); s > j.MaxSpeed {
			j.MaxSpeed = s
		}
		if h := F[k].Origin[2] - T[2]; h > j.Height {
			j.Height = h
		}
	}
	j.Gain = j.MaxSpeed - j.Pre
	j.Strafes = kzStrafes(F, i, end, dm.AirAccel, crossK)
	j.Cvars = dm.cvarCheck(i-100, end)
	for k := i + 1; k < end && k < len(F); k++ {
		if F[k].Buttons&(8|16|512|1024) == 0 {
			j.DeadAir++
		}
	}
	if len(j.Strafes) > 0 {
		sum := 0.0
		for _, st := range j.Strafes {
			sum += st.Eff
		}
		j.Eff = sum / float64(len(j.Strafes))
	}
	// map edges
	if mg != nil {
		j.MapFound = true
	}
	if mg != nil && !ladder {
		ov := ovMap
		back := T[axis] - 16*s
		tks := mg.edgeFaces(axis, s, tkFeet, T[oth]-16, T[oth]+16, ov)
		best, ok := 0.0, false
		for _, f := range tks {
			d := (f - back) * s
			if d >= -0.01 && (!ok || d < (best-back)*s) {
				best, ok = f, true
			}
		}
		if ok {
			jo := (best - back) * s
			j.Jumpoff = &jo
			j.tkEdge = best
			lfs := mg.edgeFaces(axis, -s, j.landTop, cx[oth]-16, cx[oth]+16, ov)
			lb, lok := 0.0, false
			for _, f := range lfs {
				if (f-best)*s > 100 && (!lok || (f-best)*s < (lb-best)*s) {
					lb, lok = f, true
				}
			}
			if lok {
				blk := math.Abs(lb - best)
				le := (cx[axis] + 16*s - lb) * s
				// the map's block must agree with what actually happened
				// (movable blocks, e.g. on longjumparena, are not where the map file has them)
				if (landed && le > -1) || (!landed && le <= 0.5) {
					j.Block, j.LandEdge = &blk, &le
					j.landFace = lb
					j.hasEdges = true
					ef, ff := (best-T[axis])*s, (lb-T[axis])*s
					j.EdgeFwd, j.FaceFwd = &ef, &ff
				}
			}
		}
	}
	// real take-off states for an earlier jump (lower FOG)
	gs := i - j.FOG
	for n := 1; n <= j.FOG && n <= 30; n++ {
		b := gs + n - 1
		fp := FogPoint{Pre: speedEst(F[b].Velocity), frame: b}
		if j.Jumpoff != nil {
			jo := (j.tkEdge - (F[b].Origin[axis] - 16*s)) * s
			fp.Jumpoff = &jo
		}
		j.FogBase = append(j.FogBase, fp)
	}
	if j.Landed {
		j.Result = "LANDED"
	} else {
		j.Result = "FAILED"
	}
	j.Type = guessType(dm, i, j)
	// facing vs moving direction at take-off (like [MAIN]'s Backwards / Sideways)
	if hv := math.Hypot(F[i].Velocity[0], F[i].Velocity[1]); hv > 50 {
		mv := math.Atan2(F[i].Velocity[1], F[i].Velocity[0]) * 180 / math.Pi
		d := math.Abs(wrap180(F[i-1].Yaw - mv))
		if d > 135 {
			j.Style = "bw"
		} else if d > 45 {
			j.Style = "sw"
		}
	}
	j.Plugin, j.PluginType = pluginLines(F, i, crossK)
	if j.Type == "LJ" && strings.Contains(j.PluginType, "HighJump") {
		j.Type = "HJ"
	}
	j.Chart = chart(F, i, end, T, axis, s)
	slaps(j, F, i, end)
	j.Replay = replay(dm, i, end)
	return j
}

// kzStrafes reproduces kz_jump_stats: every key is tracked from press to
// release; each frame counts for every held key, but its gain is credited
// only to the most recently pressed key. Fully overlapped strafes are dropped.
func kzStrafes(F []Frame, i, end int, airAccel float64, gcEnd int) []Strafe {
	keys := []int{8, 16, 512, 1024}
	names := map[int]string{8: "W", 16: "S", 512: "A", 1024: "D"}
	open := map[int]*Strafe{}
	var done []*Strafe
	// GC counts the turn from the jump frame on; frames before the first
	// strafe key go to the first strafe
	var preSum, preMax float64
	preN := 0
	for k := i - 1; k <= i && k < end; k++ {
		if te, ok := turnEff(F, k, airAccel); ok {
			preSum += te
			preN++
			preMax = math.Max(preMax, te)
		}
	}
	first := true
	var last *Strafe  // most recently started strafe: dead frames after it go to it
	pendingDead := 0  // dead frames before the first strafe key
	var gcCur *Strafe // GC: a strafe changes only when the new key is held alone
	for k := i + 1; k < end; k++ {
		b := F[k].Buttons
		gain := speedEst(F[k].Velocity) - speedEst(F[k-1].Velocity)
		var held []int
		for _, x := range keys {
			if b&x != 0 {
				held = append(held, x)
			}
		}
		for x, st := range open {
			if b&x == 0 {
				done = append(done, st)
				delete(open, x)
			}
		}
		te, teOK := turnEff(F, k, airAccel)
		if k > gcEnd {
			teOK = false
		}
		if len(held) == 0 {
			if last != nil {
				last.Dead++
			} else {
				pendingDead++
			}
		}
		for _, x := range held {
			if open[x] == nil {
				open[x] = &Strafe{Key: names[x], start: k}
				if first {
					open[x].effSum, open[x].effN, open[x].MaxEff = preSum, preN, preMax
					open[x].Dead = pendingDead
					first = false
				}
				last = open[x]
			}
			open[x].Frames++
			if len(held) > 1 {
				open[x].Overlaps++
			}
		}
		if len(held) == 1 {
			gcCur = open[held[0]]
		} else if gcCur == nil && len(held) > 0 {
			gcCur = open[held[0]]
		}
		if teOK && gcCur != nil {
			gcCur.effSum += te
			gcCur.effN++
			gcCur.MaxEff = math.Max(gcCur.MaxEff, te)
		}
		if len(held) > 0 {
			newest := held[0]
			for _, x := range held {
				if open[x].start > open[newest].start || (open[x].start == open[newest].start && x > newest) {
					newest = x
				}
			}
			st := open[newest]
			if gain > 0 {
				st.Good++
				st.Gain += gain
			} else {
				st.Loss -= gain
			}
		}
	}
	for _, st := range open {
		done = append(done, st)
	}
	out := []Strafe{}
	for _, st := range done {
		if st.Frames > st.Overlaps {
			st.Eff = 100 * float64(st.Good) / float64(st.Frames)
			if st.effN > 0 {
				st.AvgEff = st.effSum / float64(st.effN)
			} else {
				st.AvgEff, st.MaxEff = -1, -1 // after the jump ended (GC doesn't count it)
			}
			out = append(out, *st)
		}
	}
	// order by start
	for a := 0; a < len(out); a++ {
		for b := a + 1; b < len(out); b++ {
			if out[b].start < out[a].start {
				out[a], out[b] = out[b], out[a]
			}
		}
	}
	return out
}

func guessType(dm *Demo, i int, j *Jump) string {
	F := dm.Frames
	if j.ladder {
		return "LDJ"
	}
	// weird jump: the ground streak began by landing from a fall
	gs := i - 1
	for gs > 0 && F[gs-1].OnGround {
		gs--
	}
	if gs > 3 && i-gs <= 3 {
		// how did the air time before this landing start: a jump = bhop, a walk-off = weird jump
		st := gs - 1
		for st > 0 && !F[st-1].OnGround {
			st--
		}
		if st > 0 && st < gs && F[st].Velocity[2] > 200 && gs-st < 120 {
			// stand-up bhop: the hop before landed ducked (18 below its own take-off on the same floor)
			dz := F[gs].Origin[2] - F[st-1].Origin[2]
			if isDucked(&F[gs]) || math.Abs(dz+18) < 1 {
				return "SBJ"
			}
			return "BJ"
		}
		top := F[gs-1].Origin[2]
		for k := gs - 1; k > 0 && k > gs-80 && !F[k].OnGround; k-- {
			if F[k].Origin[2] > top {
				top = F[k].Origin[2]
			}
		}
		if top-F[gs].Origin[2] > 30 {
			return "WJ"
		}
	}
	// count/duck jumps: short duck hops just before the jump
	hops := 0
	for k := i - 2; k > 0 && k > i-70; k-- {
		if F[k].OnGround && !F[k+1].OnGround && F[k+1].Velocity[2] < 200 && F[k+1].Origin[2]-F[k].Origin[2] > 15 {
			hops++
		}
	}
	switch {
	case hops == 1:
		return "CJ"
	case hops == 2:
		return "DCJ"
	case hops > 2:
		return "MCJ"
	}
	// landing clearly higher than the take-off
	tkFeet := j.takeoff[2] - 36
	if j.duckedTO {
		tkFeet = j.takeoff[2] - 18
	}
	if j.Landed && j.landTop > tkFeet+4 {
		return "HJ"
	}
	return "LJ"
}

var pluginKinds = []struct {
	src string
	re  *regexp.Regexp
}{
	{"kz_jump_stats", regexp.MustCompile(`(?s)Distance: [\d.]+\nMaxspeed.*`)},
	{"kz_jump_stats", regexp.MustCompile(`Jumpoff: .*Block: [\d.]+`)},
	{"kz_jump_stats", regexp.MustCompile(`Prestrafe \(FOG\): .*`)},
	{"GC", regexp.MustCompile(`(?s)\[GC\] .*`)},
	{"chat", regexp.MustCompile(`\[[A-Z]+\] .* jumped [\d.]+ units.*`)},
	{"MAIN", regexp.MustCompile(`(?s)\[MAIN\] .*`)},
}
var hudNameRe = regexp.MustCompile(`\n\s*([^\n]+)\nDistance: `)

func pluginLines(F []Frame, i, cross int) ([]PluginLine, string) {
	out := []PluginLine{}
	seen := map[string]bool{}
	name := ""
	for k := cross; k < len(F) && k < cross+250; k++ {
		for _, t := range F[k].Texts {
			for _, pk := range pluginKinds {
				if m := pk.re.FindString(t); m != "" {
					m = strings.TrimSpace(colorCodes.ReplaceAllString(m, ""))
					if !seen[m] {
						seen[m] = true
						out = append(out, PluginLine{pk.src, m})
					}
					if name == "" {
						if nm := hudNameRe.FindStringSubmatch("\n" + t); nm != nil {
							name = strings.TrimSpace(nm[1])
						}
					}
				}
			}
		}
		// stop once the next jump starts
		if k > cross+5 && k+1 < len(F) && F[k].OnGround && !F[k+1].OnGround && F[k+1].Velocity[2] > 200 {
			break
		}
	}
	return out, name
}

func chart(F []Frame, i, end int, T [3]float64, axis int, sg float64) []ChartPoint {
	ux, uy := sg, 0.0
	if axis == 1 {
		ux, uy = 0, sg
	}
	var out []ChartPoint
	for k := i - 60; k < end+15 && k < len(F); k++ {
		if k < 1 {
			continue
		}
		dx, dy := F[k].Origin[0]-T[0], F[k].Origin[1]-T[1]
		out = append(out, ChartPoint{
			F: k - i, Fwd: dx*ux + dy*uy, Side: -dx*uy + dy*ux, H: F[k].Origin[2] - T[2],
			Spd: hspeed(F[k].Velocity), Turn: wrap180(F[k].Yaw - F[k-1].Yaw), Air: k >= i && k < end, Keys: F[k].Buttons,
		})
	}
	return out
}

// slaps counts mouse direction changes in the air. A "1-frame slap" is a
// change of turning direction from one frame to the next with no slow frame
// in between. Rebuilt for DEMviewer; not identical to LJSlapTool's score.
// slaps: a mouse slap is a strafe switch that looks like hitting a wall. For
// every switch of turn direction we take the biggest change in turn speed
// over two frames around the flip and divide it by the top turn speed of the
// strafes on either side ("2-frame swing"). Smooth players need 4-6 frames to
// swing over (about 55-100%); slappers go from fast one way to fast the other
// way in two frames (about 110-150%). Calibrated on known slap / no-slap demos.
const slapSharp = 1.1

func slaps(j *Jump, F []Frame, i, end int) {
	type seg struct {
		s, e int
		peak float64
		sg   int
	}
	rate := func(k int) float64 {
		ft := float64(F[k].Msec) / 1000
		if ft <= 0 {
			ft = 0.01
		}
		return wrap180(F[k].Yaw-F[k-1].Yaw) / ft
	}
	var segs []seg
	for k := i; k < end && k < len(F); k++ {
		r := rate(k)
		if math.Abs(r) < 5 {
			continue
		}
		sg := 1
		if r < 0 {
			sg = -1
		}
		if len(segs) == 0 || segs[len(segs)-1].sg != sg {
			segs = append(segs, seg{k, k, 0, sg})
		}
		sp := &segs[len(segs)-1]
		sp.e = k
		sp.peak = math.Max(sp.peak, math.Abs(r))
	}
	changes, hard, sum := 0, 0, 0.0
	for a := 1; a < len(segs); a++ {
		A, B := segs[a-1], segs[a]
		pk := math.Max(A.peak, B.peak)
		if pk <= 0 {
			continue
		}
		mx := 0.0
		for k := A.e - 1; k <= B.s+1 && k+1 < len(F); k++ {
			if k-1 >= 1 {
				mx = math.Max(mx, math.Abs(rate(k+1)-rate(k-1)))
			}
		}
		sh := mx / pk
		changes++
		sum += sh
		if sh >= slapSharp {
			hard++
		}
	}
	j.SlapTotal = changes + 1
	j.SlapOne = hard
	j.SlapScore = 0
	if changes > 0 {
		j.SlapScore = 100 * sum / float64(changes)
	}
}

func replay(dm *Demo, i, end int) *Replay {
	F := dm.Frames
	a := i - 150
	if a < 0 {
		a = 0
	}
	b := end + 80
	if b > len(F) {
		b = len(F)
	}
	r := &Replay{Map: dm.Map, T0: i - a, T1: end - a, Ents: dm.brushOffsets(i)}
	r.Moves = dm.brushTrack(a, b, r.Ents)
	dm.fillModels(r, a, b)
	for k := a; k < b; k++ {
		f := &F[k]
		og := 0.0
		if f.OnGround {
			og = 1
		}
		r.D = append(r.D, f.View[0], f.View[1], f.View[2], f.Pitch, f.Yaw, 0, float64(f.Buttons), hspeed(f.Velocity), og)
		dk := int8(0)
		if eyeOfs(f) < 14.5 || (!f.OnGround && f.Buttons&4 != 0) {
			dk = 1
		}
		r.Duck = append(r.Duck, dk)
	}
	return r
}

func mdlName(path string) string {
	n := path
	if i := strings.LastIndex(n, "/"); i >= 0 {
		n = n[i+1:]
	}
	return strings.TrimSuffix(n, ".mdl")
}

// fillModels adds the player model and the weapon models shown on frames
// a..b-1 of the demo to a replay.
func (dm *Demo) fillModels(r *Replay, a, b int) {
	r.Player = dm.UserInfo["model"]
	at := func(upd [][2]int, k int) (int, bool) {
		v, ok := 0, false
		for _, u := range upd {
			if u[0] > k {
				break
			}
			v, ok = u[1], true
		}
		return v, ok
	}
	ent := dm.PlayerSlot + 1
	var wupd [][2]int
	for _, u := range dm.weaponModelUpd {
		if u[1] == ent {
			wupd = append(wupd, [2]int{u[0], u[2]})
		}
	}
	name := func(idx int) string {
		if idx <= 0 {
			return ""
		}
		return mdlName(dm.Resources[idx])
	}
	lastV, lastP := "\x00", "\x00"
	for k := a; k < b; k++ {
		if v, ok := at(dm.viewModelUpd, k); ok || k == a {
			if n := name(v); n != lastV {
				r.VModel = append(r.VModel, [2]any{k - a, n})
				lastV = n
			}
		}
		if v, ok := at(wupd, k); ok || k == a {
			if n := name(v); n != lastP {
				r.PModel = append(r.PModel, [2]any{k - a, n})
				lastP = n
			}
		}
	}
}
