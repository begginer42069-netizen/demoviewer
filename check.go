package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
)

var (
	reKzDist = regexp.MustCompile(`Distance: ([\d.]+)\nMaxspeed: ([\d.]+) \(([\d.]+)\)`)
	reKzJof  = regexp.MustCompile(`Jumpoff: ([\d.]+) (Landing|Lasting): ([\d.]+) Block: ([\d.]+)`)
	reKzPre  = regexp.MustCompile(`Prestrafe \(FOG\): ([\d.]+)`)
	reKzEff  = regexp.MustCompile(`Strafes: (\d+)\nEff: (\d+)%`)
)

func atof(s string) float64 { v, _ := strconv.ParseFloat(s, 64); return v }

// runCheck prints DEMviewer's numbers next to the kz_jump_stats values recorded in each demo.
func runCheck(dir string) {
	extraMapDirs = mapDirs(dir)
	files, _ := filepath.Glob(filepath.Join(dir, "*.dem"))
	for _, p := range files {
		dm, err := LoadDemo(p)
		if err != nil {
			fmt.Println(filepath.Base(p), "ERR", err)
			continue
		}
		dm.Name = filepath.Base(p)
		mg, _ := LoadMap(dm.Map)
		js := FindJumps(dm, mg)
		// the jump the recorded kz stats belong to: last jump with kz lines
		var j *Jump
		for _, x := range js {
			for _, l := range x.Plugin {
				if l.Source == "kz_jump_stats" {
					j = x
				}
			}
		}
		if j == nil {
			fmt.Printf("%-30s no kz-stats jump found (%d jumps)\n", short(dm.Name), len(js))
			continue
		}
		var kd, kpre, kjo, kle, kblk, keff float64
		kstr := 0
		for _, l := range j.Plugin {
			if m := reKzDist.FindStringSubmatch(l.Text); m != nil {
				kd = atof(m[1])
			}
			if m := reKzJof.FindStringSubmatch(l.Text); m != nil {
				kjo, kle, kblk = atof(m[1]), atof(m[3]), atof(m[4])
				if m[2] == "Lasting" {
					kle = -kle
				}
			}
			if m := reKzPre.FindStringSubmatch(l.Text); m != nil {
				kpre = atof(m[1])
			}
			if m := reKzEff.FindStringSubmatch(l.Text); m != nil {
				kstr, _ = strconv.Atoi(m[1])
				keff = atof(m[2])
			}
		}
		opt := func(p *float64) float64 {
			if p == nil {
				return math_nan()
			}
			return *p
		}
		wi := RunWhatIf(dm, j, j.Pre, j.Jumpoff, 0)
		fmt.Printf("    sim check: real %.3f  sim %.3f  diff %+.3f  landed %v/%v\n", j.Distance, j.Distance+wi.CheckDiff, wi.CheckDiff, wi.Landed, j.Landed)
		fmt.Printf("%-30s %-4s %-6s dist %8.3f (%+.3f) pre %7.3f (%+.3f) jof %6.3f (%+.3f) land %+6.3f (%+.3f) blk %5.1f/%5.1f str %d/%d eff %4.1f/%2.0f\n",
			short(dm.Name), j.Type, j.Result, j.Distance, j.Distance-kd, j.Pre, j.Pre-kpre, opt(j.Jumpoff), opt(j.Jumpoff)-kjo,
			opt(j.LandEdge), opt(j.LandEdge)-kle, opt(j.Block), kblk, len(j.Strafes), kstr, j.Eff, keff)
	}
}

func short(s string) string {
	if len(s) > 30 {
		return s[:30]
	}
	return s
}

func math_nan() float64 { v, _ := strconv.ParseFloat("NaN", 64); return v }

func init() {
	if len(os.Args) == 5 && os.Args[1] == "-whatif" {
		dm, _ := LoadDemo(os.Args[2])
		mg, _ := LoadMap(dm.Map)
		js := FindJumps(dm, mg)
		j := js[len(js)-1]
		e := atof(os.Args[4])
		r := RunWhatIf(dm, j, atof(os.Args[3]), &e, 0)
		fmt.Printf("real: %s %s dist %.3f pre %.3f edge %.3f block %v\n", j.Type, j.Result, j.Distance, j.Pre, *j.Jumpoff, *j.Block)
		fmt.Printf("what-if: pre %.2f edge %.2f -> landed %v dist %.3f landEdge %.3f (self-check %+.3f) max %.1f eff %.0f capped %v\n", r.Pre, e, r.Landed, r.Distance, *r.LandEdge, r.CheckDiff, r.MaxSpeed, r.Eff, r.Capped)
		os.Exit(0)
	}
	if len(os.Args) == 3 && os.Args[1] == "-check" {
		runCheck(os.Args[2])
		os.Exit(0)
	}
}
