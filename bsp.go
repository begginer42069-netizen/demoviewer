package main

import (
	"bytes"
	"embed"
	"compress/gzip"
	"encoding/binary"
	"errors"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
)


// Face is a world/brush polygon reduced to what edge finding needs.
type Face struct {
	N                      [3]float64
	X0, X1, Y0, Y1, Z0, Z1 float64 // without entity offset
	Model                  int
	V                      [][3]float64
}

type MapGeo struct {
	Name   string
	Faces  []Face
	DefOff map[int][3]float64
	Raw   []byte
	From  string
}

//go:embed embedmaps/*.bsp.gz
var builtinMaps embed.FS

var (
	mapCache   = map[string]*MapGeo{}
	mapCacheMu sync.Mutex
	extraMapDirs []string
)

// LoadMap finds a map by name: embedded copy first, then local folders.
func LoadMap(name string) (*MapGeo, error) {
	name = strings.ToLower(strings.TrimSuffix(name, ".bsp"))
	mapCacheMu.Lock()
	defer mapCacheMu.Unlock()
	if g, ok := mapCache[name]; ok {
		return g, nil
	}
	var raw []byte
	from := ""
	for _, dir := range extraMapDirs {
		p := filepath.Join(dir, name+".bsp")
		if b, err := os.ReadFile(p); err == nil {
			raw, from = b, p
			break
		}
		if gz, err := os.ReadFile(p + ".gz"); err == nil {
			if zr, err := gzip.NewReader(bytes.NewReader(gz)); err == nil {
				if b, err := io.ReadAll(zr); err == nil {
					raw, from = b, p+".gz"
					break
				}
			}
		}
	}
	if raw == nil {
		// built into the exe: kz_longjumps2, kz_longjumps3, kz_longjumparena
		if gz, err := builtinMaps.ReadFile("embedmaps/" + name + ".bsp.gz"); err == nil {
			if zr, err := gzip.NewReader(bytes.NewReader(gz)); err == nil {
				if b, err := io.ReadAll(zr); err == nil {
					raw, from = b, "built in"
				}
			}
		}
	}
	if raw == nil {
		return nil, errors.New("map " + name + ".bsp not found")
	}
	g, err := parseBSPFaces(raw)
	if err != nil {
		return nil, err
	}
	g.Name, g.Raw, g.From = name, raw, from
	mapCache[name] = g
	return g, nil
}

var entRe = regexp.MustCompile(`\{([^}]*)\}`)
var kvRe = regexp.MustCompile(`"([^"]*)"\s+"([^"]*)"`)

func parseBSPFaces(d []byte) (*MapGeo, error) {
	if len(d) < 124 || binary.LittleEndian.Uint32(d) != 30 {
		return nil, errors.New("not a GoldSrc BSP")
	}
	type lump struct{ o, l int }
	L := make([]lump, 15)
	for i := range L {
		L[i] = lump{int(i32(d[4+i*8:])), int(i32(d[8+i*8:]))}
	}
	ent := string(d[L[0].o : L[0].o+L[0].l])
	offs := map[int][3]float64{0: {}}
	for _, m := range entRe.FindAllStringSubmatch(ent, -1) {
		kv := map[string]string{}
		for _, p := range kvRe.FindAllStringSubmatch(m[1], -1) {
			kv[p[1]] = p[2]
		}
		mdl := kv["model"]
		if !strings.HasPrefix(mdl, "*") {
			continue
		}
		c := kv["classname"]
		if strings.HasPrefix(c, "trigger_") || c == "func_ladder" || c == "func_water" || c == "func_illusionary" {
			continue
		}
		idx, err := strconv.Atoi(mdl[1:])
		if err != nil {
			continue
		}
		var o [3]float64
		if s := kv["origin"]; s != "" {
			f := strings.Fields(s)
			for i := 0; i < 3 && i < len(f); i++ {
				o[i], _ = strconv.ParseFloat(f[i], 64)
			}
		}
		offs[idx] = o
	}
	vert := func(i int) [3]float64 {
		b := L[3].o + i*12
		return [3]float64{f32(d[b:]), f32(d[b+4:]), f32(d[b+8:])}
	}
	g := &MapGeo{DefOff: offs}
	nmodels := L[14].l / 64
	for mi := range offs {
		if mi >= nmodels {
			continue
		}
		mb := L[14].o + mi*64
		first, num := int(i32(d[mb+56:])), int(i32(d[mb+60:]))
		for fi := first; fi < first+num; fi++ {
			b := L[7].o + fi*20
			plane := int(binary.LittleEndian.Uint16(d[b:]))
			side := int16(binary.LittleEndian.Uint16(d[b+2:]))
			fe := int(i32(d[b+4:]))
			ne := int(binary.LittleEndian.Uint16(d[b+8:]))
			pb := L[1].o + plane*20
			n := [3]float64{f32(d[pb:]), f32(d[pb+4:]), f32(d[pb+8:])}
			if side != 0 {
				n = [3]float64{-n[0], -n[1], -n[2]}
			}
			f := Face{Model: mi, N: n, X0: math.Inf(1), X1: math.Inf(-1), Y0: math.Inf(1), Y1: math.Inf(-1), Z0: math.Inf(1), Z1: math.Inf(-1)}
			for k := 0; k < ne; k++ {
				se := int(i32(d[L[13].o+(fe+k)*4:]))
				e := se
				if e < 0 {
					e = -e
				}
				eb := L[12].o + e*4
				vi := int(binary.LittleEndian.Uint16(d[eb:]))
				if se < 0 {
					vi = int(binary.LittleEndian.Uint16(d[eb+2:]))
				}
				v := vert(vi)
				f.V = append(f.V, v)
				x, y, z := v[0], v[1], v[2]
				f.X0, f.X1 = math.Min(f.X0, x), math.Max(f.X1, x)
				f.Y0, f.Y1 = math.Min(f.Y0, y), math.Max(f.Y1, y)
				f.Z0, f.Z1 = math.Min(f.Z0, z), math.Max(f.Z1, z)
			}
			g.Faces = append(g.Faces, f)
		}
	}
	return g, nil
}

// edgeFaces returns positions (along axis) of flat vertical faces whose
// normal points along s*axis, whose top is at feetZ and that overlap [lo,hi]
// on the other horizontal axis.
func (g *MapGeo) edgeFaces(axis int, s float64, feetZ, lo, hi float64, ov map[int][6]float64) []float64 {
	var out []float64
	for _, f := range g.Faces {
		n := f.N
		X0, X1, Y0, Y1, Z0, Z1 := f.X0, f.X1, f.Y0, f.Y1, f.Z0, f.Z1
		o := g.DefOff[f.Model]
		if v, ok := ov[f.Model]; ok && f.Model != 0 {
			o = [3]float64{v[0], v[1], v[2]}
			if v[3] != 0 || v[4] != 0 || v[5] != 0 {
				n = rotateBrush(n, v[3], v[4], v[5])
				X0, X1, Y0, Y1, Z0, Z1 = math.Inf(1), math.Inf(-1), math.Inf(1), math.Inf(-1), math.Inf(1), math.Inf(-1)
				for _, p := range f.V {
					q := rotateBrush(p, v[3], v[4], v[5])
					X0, X1 = math.Min(X0, q[0]), math.Max(X1, q[0])
					Y0, Y1 = math.Min(Y0, q[1]), math.Max(Y1, q[1])
					Z0, Z1 = math.Min(Z0, q[2]), math.Max(Z1, q[2])
				}
			}
		}
		if math.Abs(n[axis]-s) > 1e-3 {
			continue
		}
		a0, a1, o0, o1 := X0+o[0], X1+o[0], Y0+o[1], Y1+o[1]
		if axis == 1 {
			a0, a1, o0, o1 = Y0+o[1], Y1+o[1], X0+o[0], X1+o[0]
		}
		_ = Z0
		if a1-a0 > 0.01 || math.Abs(Z1+o[2]-feetZ) > 0.5 || o1 < lo || o0 > hi {
			continue
		}
		out = append(out, a0)
	}
	return out
}

// floorBelow returns the highest walkable surface under a player hull at
// (x,y) that is not above z (bounding-box test on upward faces).
func (g *MapGeo) floorBelow(x, y, z float64, ov map[int][6]float64) (float64, bool) {
	best, ok := math.Inf(-1), false
	for _, f := range g.Faces {
		n := f.N
		X0, X1, Y0, Y1, Z1 := f.X0, f.X1, f.Y0, f.Y1, f.Z1
		o := g.DefOff[f.Model]
		if v, has := ov[f.Model]; has && f.Model != 0 {
			o = [3]float64{v[0], v[1], v[2]}
			if v[3] != 0 || v[4] != 0 || v[5] != 0 {
				continue // rotated pieces: skip, rare as floors
			}
		}
		if n[2] < 0.7 {
			continue
		}
		top := Z1 + o[2]
		if top > z+0.5 || top < best {
			continue
		}
		if x+16 < X0+o[0] || x-16 > X1+o[0] || y+16 < Y0+o[1] || y-16 > Y1+o[1] {
			continue
		}
		best, ok = top, true
	}
	return best, ok
}

// floorAt reports whether an upward surface with its top at ztop (+-0.1)
// lies under a player hull at (x,y).
func (g *MapGeo) floorAt(x, y, ztop float64, ov map[int][6]float64) bool {
	for _, f := range g.Faces {
		n := f.N
		X0, X1, Y0, Y1, Z1 := f.X0, f.X1, f.Y0, f.Y1, f.Z1
		o := g.DefOff[f.Model]
		if v, has := ov[f.Model]; has && f.Model != 0 {
			o = [3]float64{v[0], v[1], v[2]}
			if v[3] != 0 || v[4] != 0 || v[5] != 0 {
				n = rotateBrush(n, v[3], v[4], v[5])
				X0, X1, Y0, Y1, Z1 = math.Inf(1), math.Inf(-1), math.Inf(1), math.Inf(-1), math.Inf(-1)
				for _, p := range f.V {
					q := rotateBrush(p, v[3], v[4], v[5])
					X0, X1 = math.Min(X0, q[0]), math.Max(X1, q[0])
					Y0, Y1 = math.Min(Y0, q[1]), math.Max(Y1, q[1])
					Z1 = math.Max(Z1, q[2])
				}
			}
		}
		if n[2] < 0.7 || math.Abs(Z1+o[2]-ztop) > 0.1 {
			continue
		}
		if x+16 < X0+o[0] || x-16 > X1+o[0] || y+16 < Y0+o[1] || y-16 > Y1+o[1] {
			continue
		}
		return true
	}
	return false
}

// ladderFloor: top of the highest floor just below a ladder take-off
// (within 8 units under the feet, 96 units around).
func (g *MapGeo) ladderFloor(x, y, feet float64, ov map[int][6]float64) (float64, bool) {
	best, ok := -1e9, false
	for _, f := range g.Faces {
		if f.N[2] < 0.7 {
			continue
		}
		var o [3]float64
		if f.Model != 0 {
			if e, has := ov[f.Model]; has {
				o = [3]float64{e[0], e[1], e[2]}
			} else if d, has := g.DefOff[f.Model]; has {
				o = d
			}
		}
		z := f.Z1 + o[2]
		if z > feet+0.5 || z < feet-8 || z <= best {
			continue
		}
		if x < f.X0+o[0]-96 || x > f.X1+o[0]+96 || y < f.Y0+o[1]-96 || y > f.Y1+o[1]+96 {
			continue
		}
		best, ok = z, true
	}
	return best, ok
}
