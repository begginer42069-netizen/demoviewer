package main

import "math"

// WhatIf re-runs a jump from an adjusted take-off: horizontal speed set to
// newPre (same direction) and the take-off point moved along the jump axis so
// the jumpoff becomes newEdge. Every input from the jump frame onward is the
// player's own recorded input; nothing before the jump is simulated.
type WhatIfResult struct {
	Pre       float64   `json:"pre"`
	Fog       int       `json:"fog"`
	Edge      *float64  `json:"edge"`
	Capped    bool      `json:"capped"`
	CapSpeed  float64   `json:"capSpeed"`
	Landed    bool      `json:"landed"`
	Known     bool      `json:"known"` // land/fail decided with map geometry
	Distance  float64   `json:"dist"`
	LandEdge  *float64  `json:"landEdge"`
	MaxSpeed  float64   `json:"max"`
	Strafes   []Strafe  `json:"strafes"`
	Eff       float64   `json:"eff"`
	Path      []float64 `json:"path"` // per frame x,y,z (origin), speed
	CheckDiff float64   `json:"checkDiff"`
	Replay    *Replay   `json:"replay"`
}

type simState struct{ x, y, z, vx, vy, vz float64 }

func simulate(dm *Demo, j *Jump, newPre float64, edgeShift float64) (states []simState, res WhatIfResult) {
	F := dm.Frames
	g := dm.Gravity
	ax, s := j.Axis, j.Sign
	st := simState{x: j.takeoff[0], y: j.takeoff[1], z: j.takeoff[2]}
	if ax == 0 {
		st.x += edgeShift * s
	} else {
		st.y += edgeShift * s
	}
	// pre-jump velocity: corrected for the server's 1/8 truncation, then scaled
	v := j.preVel
	cx := v[0] + math.Copysign(1.0/16, v[0])
	cy := v[1] + math.Copysign(1.0/16, v[1])
	hv := math.Hypot(cx, cy)
	st.vx, st.vy = cx/hv*newPre, cy/hv*newPre
	res.Pre = newPre
	feetOff := func(ducked bool) float64 {
		if ducked {
			return 18
		}
		return 36
	}
	tkFeet := j.takeoff[2] - feetOff(j.duckedTO)
	states = append(states, st)
	ducked := j.duckedTO
	for k := j.cmdFrom; k < len(F) && k < j.cmdFrom+400; k++ {
		f := &F[k]
		ft := float64(f.Msec) / 1000
		if ft <= 0 {
			ft = 0.01
		}
		// one demo frame can hold several server ticks (skipped frames) or
		// none (a repeated frame): run the physics that many times
		reps := simReps(F, j, k, g)
		if reps == 0 {
			states = append(states, st)
			continue
		}
		for rep := 0; rep < reps; rep++ {
			last := rep == reps-1
			if k == j.cmdFrom && rep == 0 {
				// PM_Jump: CS 1.6 caps the take-off speed above 300 (1.2 x 250)
				sp := math.Sqrt(st.vx*st.vx + st.vy*st.vy)
				if sp > 300 && !j.ladder {
					fr := 300 / sp * 0.8
					st.vx, st.vy = st.vx*fr, st.vy*fr
					res.Capped = true
					res.CapSpeed = math.Hypot(st.vx, st.vy)
				}
				st.vz = j.jumpVz
			} else {
				st.vz -= 0.5 * g * ft
			}
			// in the air ducking and standing up are instant: the duck key decides
			ducked = f.Buttons&4 != 0
			// PM_AirMove / PM_AirAccelerate
			fm, sm := f.Forward, f.Side
			if ducked {
				fm, sm = fm*0.333, sm*0.333
			}
			yaw := f.Yaw
			if reps > 1 && k > 0 {
				// the skipped tick's aim lies between the two recorded frames
				yaw = F[k-1].Yaw + wrap180(f.Yaw-F[k-1].Yaw)*float64(rep+1)/float64(reps)
			}
			yr := yaw * math.Pi / 180
			fx, fy := math.Cos(yr), math.Sin(yr)
			rx, ry := math.Sin(yr), -math.Cos(yr)
			wx, wy := fx*fm+rx*sm, fy*fm+ry*sm
			ws := math.Hypot(wx, wy)
			if ws > 0 {
				dx, dy := wx/ws, wy/ws
				maxs := dm.MaxSpeed
				if maxs <= 0 || maxs > 250 {
					maxs = 250
				}
				if ws > maxs {
					ws = maxs
				}
				w := math.Min(ws, 30)
				add := w - (st.vx*dx + st.vy*dy)
				if add > 0 {
					a := math.Min(dm.AirAccel*ws*ft, add)
					st.vx += a * dx
					st.vy += a * dy
				}
			}
			px, py, pz := st.x, st.y, st.z
			st.x += st.vx * ft
			st.y += st.vy * ft
			st.z += st.vz * ft
			st.vz -= 0.5 * g * ft
			if last && k < j.LandFrame && k < len(F) {
				// height doesn't depend on pre or jumpoff: follow the real flight
				// (this keeps duck / un-duck body shifts exact)
				st.z = F[k].Origin[2]
			}
			if last {
				states = append(states, st)
			}
			feet := st.z - feetOff(ducked)
			front := func(x, y float64) float64 {
				if ax == 0 {
					return (x + 16*s - j.landFace) * s
				}
				return (y + 16*s - j.landFace) * s
			}
			// landing on top of the block
			if j.hasEdges && k > j.cmdFrom+5 && st.vz < 0 {
				top := j.landTop
				pfeet := pz - feetOff(ducked)
				if pfeet >= top && feet < top {
					f := (pfeet - top) / (pfeet - feet)
					cxp, cyp := px+(st.x-px)*f, py+(st.y-py)*f
					if front(cxp, cyp) > 0 {
						res.Landed, res.Known = true, true
						res.Distance = math.Hypot(cxp-j.takeoff[0]-shiftX(ax, edgeShift*s), cyp-j.takeoff[1]-shiftY(ax, edgeShift*s)) + 32
						le := front(cxp, cyp)
						res.LandEdge = &le
						return
					}
				}
			}
			// fail: ducked hull reaches the take-off floor (kz_jump_stats rule)
			thr := tkFeet + 18
			if !j.hasEdges {
				// no block geometry (ladder / bhop blocks): measure where the real
				// jump was measured - its landing height, or the fail line
				thr = j.landZ
			}
			if k > j.cmdFrom+5 && pz > thr && st.z <= thr {
				f := (pz - thr) / (pz - st.z)
				cxp, cyp := px+(st.x-px)*f, py+(st.y-py)*f
				res.Distance = math.Hypot(cxp-j.takeoff[0]-shiftX(ax, edgeShift*s), cyp-j.takeoff[1]-shiftY(ax, edgeShift*s)) + 32
				if j.ladder {
					res.Distance -= 32
				}
				if j.hasEdges {
					res.Known = true
					le := front(cxp, cyp)
					res.LandEdge = &le
					res.Landed = false
				}
				// without map geometry: same height landing counts if it got far enough is unknown
				return
			}
		}
	}
	return
}

func shiftX(ax int, d float64) float64 {
	if ax == 0 {
		return d
	}
	return 0
}
func shiftY(ax int, d float64) float64 {
	if ax == 1 {
		return d
	}
	return 0
}

// RunWhatIf runs the simulation and also checks the simulator against the
// real jump (same pre and edge) so the user can see how far to trust it.
func RunWhatIf(dm *Demo, j *Jump, newPre float64, newEdge *float64, fog int) WhatIfResult {
	_, base := simulate(dm, j, j.Pre, 0)
	real := j
	if fog >= 1 && fog < j.FOG && fog <= len(j.FogBase) {
		// jump earlier: start from the player's real state on that ground frame
		fp := j.FogBase[fog-1]
		c := *j
		f := &dm.Frames[fp.frame]
		c.takeoff, c.preVel, c.duckedTO, c.baseFrame = f.Origin, f.Velocity, isDucked(f), fp.frame
		c.Jumpoff = fp.Jumpoff
		j = &c
	}
	shift := 0.0
	if newEdge != nil && j.Jumpoff != nil {
		shift = *j.Jumpoff - *newEdge
	}
	states, res := simulate(dm, j, newPre, shift)
	res.CheckDiff = base.Distance - real.Distance
	res.Fog = j.FOG
	if fog >= 1 && fog < j.FOG {
		res.Fog = fog
	}
	if newEdge != nil && j.Jumpoff != nil {
		e := *newEdge
		res.Edge = &e
	}
	// strafe stats along the simulated path (same keys, simulated speeds)
	F := dm.Frames
	var sp []float64
	for _, st := range states {
		sp = append(sp, math.Hypot(st.vx, st.vy))
	}
	for _, v := range sp {
		if v > res.MaxSpeed {
			res.MaxSpeed = v
		}
	}
	res.Strafes, res.Eff = simStrafes(F, j.cmdFrom, sp)
	// path + replay (pre-jump part from the demo, then the simulated flight)
	r := &Replay{Map: dm.Map, Ents: j.Replay.Ents, Moves: j.Replay.Moves}
	a := j.cmdFrom - 150
	if a < 0 {
		a = 0
	}
	for k := a; k <= j.baseFrame; k++ {
		f := &F[k]
		og := 0.0
		if f.OnGround {
			og = 1
		}
		r.D = append(r.D, f.View[0], f.View[1], f.View[2], f.Pitch, f.Yaw, 0, float64(f.Buttons), hspeed(f.Velocity), og)
	}
	r.T0 = j.baseFrame + 1 - a
	for n := 1; n < len(states); n++ {
		k := j.cmdFrom + n - 1
		if k >= len(F) {
			break
		}
		f := &F[k]
		eye := 17.0
		if isDucked(f) || f.Buttons&4 != 0 {
			eye = 12
		}
		st := states[n]
		r.D = append(r.D, st.x, st.y, st.z+eye, f.Pitch, f.Yaw, 0, float64(f.Buttons), math.Hypot(st.vx, st.vy), 0)
		res.Path = append(res.Path, st.x, st.y, st.z, math.Hypot(st.vx, st.vy))
	}
	r.T1 = len(r.D)/9 - 1
	// a simulated landing on the block: carry on along the block top with
	// GoldSrc ground friction (sv_friction 4, sv_stopspeed 75) until stopped.
	// No new inputs are made up: the player just slides out, view held still.
	if res.Landed && len(states) > 1 && len(r.D) >= 9 {
		st := states[len(states)-1]
		last := r.D[len(r.D)-9:]
		pitch, yaw, btn := last[3], last[4], last[6]
		ducked := int(btn)&4 != 0
		oz := j.landTop + 36
		eye := 17.0
		if ducked {
			oz, eye = j.landTop+18, 12
		}
		x, y, vx, vy := st.x, st.y, st.vx, st.vy
		for k := 0; k < 90; k++ {
			ft := 0.01
			sp := math.Hypot(vx, vy)
			if sp > 0.1 {
				drop := math.Max(sp, 75) * 4 * ft
				ns := math.Max(0, sp-drop)
				vx, vy = vx*ns/sp, vy*ns/sp
			} else {
				vx, vy = 0, 0
			}
			x += vx * ft
			y += vy * ft
			r.D = append(r.D, x, y, oz+eye, pitch, yaw, 0, btn, math.Hypot(vx, vy), 1)
		}
	}
	// hold the last frame for a moment
	if n := len(r.D); n >= 9 {
		last := append([]float64(nil), r.D[n-9:]...)
		last[8] = 1
		for i := 0; i < 40; i++ {
			r.D = append(r.D, last...)
		}
	}
	res.Replay = r
	return res
}

func simStrafes(F []Frame, from int, sp []float64) ([]Strafe, float64) {
	// reuse the kz rules with simulated speeds
	keys := []int{8, 16, 512, 1024}
	names := map[int]string{8: "W", 16: "S", 512: "A", 1024: "D"}
	open := map[int]*Strafe{}
	var done []*Strafe
	for n := 1; n < len(sp); n++ {
		k := from + n - 1
		if k >= len(F) {
			break
		}
		b := F[k].Buttons
		gain := sp[n] - sp[n-1]
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
		for _, x := range held {
			if open[x] == nil {
				open[x] = &Strafe{Key: names[x], start: k}
			}
			open[x].Frames++
			if len(held) > 1 {
				open[x].Overlaps++
			}
		}
		if len(held) > 0 {
			newest := held[0]
			for _, x := range held {
				if open[x].start > open[newest].start {
					newest = x
				}
			}
			if gain > 0 {
				open[newest].Good++
				open[newest].Gain += gain
			} else {
				open[newest].Loss -= gain
			}
		}
	}
	for _, st := range open {
		done = append(done, st)
	}
	out := []Strafe{}
	sum := 0.0
	for _, st := range done {
		if st.Frames > st.Overlaps {
			st.Eff = 100 * float64(st.Good) / float64(st.Frames)
			out = append(out, *st)
			sum += st.Eff
		}
	}
	for a := 0; a < len(out); a++ {
		for b := a + 1; b < len(out); b++ {
			if out[b].start < out[a].start {
				out[a], out[b] = out[b], out[a]
			}
		}
	}
	if len(out) == 0 {
		return out, 0
	}
	return out, sum / float64(len(out))
}

// simReps: how many server ticks demo frame k covers in the real flight.
func simReps(F []Frame, j *Jump, k int, g float64) int {
	if k < 1 || k >= len(F) {
		return 1
	}
	ft := float64(F[k].Msec) / 1000
	if k == j.cmdFrom {
		if j.ladder {
			return 1
		}
		step := math.Hypot(F[k].Velocity[0], F[k].Velocity[1]) * ft
		if step > 0.5 && math.Hypot(F[k].Origin[0]-F[k-1].Origin[0], F[k].Origin[1]-F[k-1].Origin[1]) > 1.5*step {
			return airTicks(F, k, g)
		}
		return 1
	}
	step := math.Hypot(F[k-1].Velocity[0], F[k-1].Velocity[1]) * ft
	if step < 0.5 {
		return 1
	}
	n := int(math.Round(math.Hypot(F[k].Origin[0]-F[k-1].Origin[0], F[k].Origin[1]-F[k-1].Origin[1]) / step))
	if n < 0 {
		n = 0
	}
	if n > 4 {
		n = 4
	}
	return n
}
