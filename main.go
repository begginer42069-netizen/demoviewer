package main

import (
	"embed"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

//go:embed web/index.html
var webFS embed.FS

type demoEntry struct {
	Name  string  `json:"name"`
	Path  string  `json:"path"`
	Map   string  `json:"map"`
	Error string  `json:"error,omitempty"`
	Jumps []*Jump `json:"-"`
	demo  *Demo
	mtime time.Time
}

type jumpSummary struct {
	Demo     int      `json:"d"`
	Jump     int      `json:"j"`
	Name     string   `json:"name"`
	Map      string   `json:"map"`
	Type     string   `json:"type"`
	Style    string   `json:"style"`
	Plugin   string   `json:"pluginType"`
	Result   string   `json:"result"`
	Dist     float64  `json:"dist"`
	Pre      float64  `json:"pre"`
	Jumpoff  *float64 `json:"jumpoff"`
	Block    *float64 `json:"block"`
	LandEdge *float64 `json:"landEdge"`
	Strafes  int      `json:"strafes"`
	Eff      float64  `json:"eff"`
	Max      float64  `json:"max"`
	Slaps    int      `json:"slaps"`
	OneSlaps int      `json:"oneFrameSlaps"`
	SlapScore float64 `json:"slapScore"`
	Main      bool    `json:"main"`
	Time      float64 `json:"time"`
}

var (
	mu      sync.Mutex
	folder  string
	demos   []*demoEntry
	cache   = map[string]*demoEntry{}
)

func exeDir() string {
	p, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(p)
}

func scan(dir string) {
	var paths []string
	filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() && strings.Count(strings.TrimPrefix(p, dir), string(os.PathSeparator)) > 3 {
			return filepath.SkipDir
		}
		if !d.IsDir() && strings.EqualFold(filepath.Ext(p), ".dem") {
			paths = append(paths, p)
		}
		return nil
	})
	sort.Strings(paths)
	extraMapDirs = mapDirs(dir)
	var out []*demoEntry
	var wg sync.WaitGroup
	sem := make(chan struct{}, runtime.NumCPU())
	res := make([]*demoEntry, len(paths))
	for i, p := range paths {
		st, err := os.Stat(p)
		if err != nil {
			continue
		}
		if c, ok := cache[p]; ok && c.mtime.Equal(st.ModTime()) {
			res[i] = c
			continue
		}
		wg.Add(1)
		go func(i int, p string, mt time.Time) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			rel, _ := filepath.Rel(dir, p)
			e := &demoEntry{Name: rel, Path: p, mtime: mt}
			defer func() {
				if r := recover(); r != nil {
					e.Error = fmt.Sprint("could not read demo: ", r)
				}
				res[i] = e
			}()
			dm, err := LoadDemo(p)
			if err != nil {
				e.Error = err.Error()
				return
			}
			dm.Name = rel
			e.Map, e.demo = dm.Map, dm
			mg, _ := LoadMap(dm.Map)
			e.Jumps = listJumps(FindJumps(dm, mg))
		}(i, p, st.ModTime())
	}
	wg.Wait()
	for _, e := range res {
		if e != nil {
			cache[e.Path] = e
			out = append(out, e)
		}
	}
	mu.Lock()
	folder, demos = dir, out
	mu.Unlock()
}

func mapDirs(demoDir string) []string {
	var ds []string
	if settings.MapsFolder != "" {
		ds = append(ds, settings.MapsFolder)
	}
	ds = append(ds, filepath.Join(exeDir(), "maps"), demoDir, exeDir())
	for _, base := range []string{`C:\Program Files (x86)\Steam\steamapps\common\Half-Life`, `C:\Program Files\Steam\steamapps\common\Half-Life`, `D:\Steam\steamapps\common\Half-Life`, `D:\SteamLibrary\steamapps\common\Half-Life`, `E:\SteamLibrary\steamapps\common\Half-Life`} {
		ds = append(ds, filepath.Join(base, "cstrike_downloads", "maps"), filepath.Join(base, "cstrike", "maps"))
	}
	if h, err := os.UserHomeDir(); err == nil {
		for _, base := range []string{".steam/steam/steamapps/common/Half-Life", ".local/share/Steam/steamapps/common/Half-Life", ".var/app/com.valvesoftware.Steam/.local/share/Steam/steamapps/common/Half-Life"} {
			b := filepath.Join(h, base)
			ds = append(ds, filepath.Join(b, "cstrike_downloads", "maps"), filepath.Join(b, "cstrike", "maps"))
		}
	}
	return ds
}

func summaries() []jumpSummary {
	mu.Lock()
	defer mu.Unlock()
	var out []jumpSummary
	for di, e := range demos {
		for _, j := range e.Jumps {
			out = append(out, jumpSummary{di, j.ID, e.Name, e.Map, j.Type, j.Style, j.PluginType, j.Result, j.Distance, j.Pre, j.Jumpoff, j.Block, j.LandEdge, len(j.Strafes), j.Eff, j.MaxSpeed, j.SlapTotal, j.SlapOne, j.SlapScore, j.Main, j.Time})
		}
	}
	sort.SliceStable(out, func(a, b int) bool { return out[a].Dist > out[b].Dist })
	return out
}

func getJump(r *http.Request) (*demoEntry, *Jump, bool) {
	di, _ := strconv.Atoi(r.URL.Query().Get("d"))
	ji, _ := strconv.Atoi(r.URL.Query().Get("j"))
	mu.Lock()
	defer mu.Unlock()
	if di < 0 || di >= len(demos) || ji < 0 || ji >= len(demos[di].Jumps) {
		return nil, nil, false
	}
	return demos[di], demos[di].Jumps[ji], true
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func main() {
	loadSettings()
	dir := exeDir()
	if p := settings.DemoFolder; p != "" && isDir(p) {
		dir = p
	} else if !hasDemos(dir) && hasDemos(filepath.Dir(dir)) {
		dir = filepath.Dir(dir)
	}
	if len(os.Args) > 1 && !strings.HasPrefix(os.Args[1], "-") {
		dir = os.Args[1]
	}
	scan(dir)
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		b, _ := webFS.ReadFile("web/index.html")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(b)
	})
	mux.HandleFunc("/api/state", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		type dinfo struct {
			Name, Map, Error string
			Jumps            int
		}
		var ds []dinfo
		for _, e := range demos {
			ds = append(ds, dinfo{e.Name, e.Map, e.Error, len(e.Jumps)})
		}
		f := folder
		mu.Unlock()
		writeJSON(w, map[string]any{"folder": f, "mapsFolder": settings.MapsFolder, "demos": ds, "jumps": summaries()})
	})
	mux.HandleFunc("/api/folder", func(w http.ResponseWriter, r *http.Request) {
		var req struct{ Path string }
		json.NewDecoder(r.Body).Decode(&req)
		p := strings.Trim(strings.TrimSpace(req.Path), `"`)
		if st, err := os.Stat(p); err != nil || !st.IsDir() {
			http.Error(w, "folder not found: "+p, 400)
			return
		}
		scan(p)
		settings.DemoFolder = p
		saveSettings()
		writeJSON(w, map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/browse", func(w http.ResponseWriter, r *http.Request) {
		kind := r.URL.Query().Get("kind")
		mu.Lock()
		start := folder
		mu.Unlock()
		var p string
		var err error
		switch kind {
		case "demo":
			p, err = browseDialog("demo", "Open a demo", start)
		case "maps":
			s := settings.MapsFolder
			if s == "" {
				s = start
			}
			p, err = browseDialog("folder", "Choose the maps folder (cstrike\\maps, cstrike_downloads\\maps, ...)", s)
		default:
			p, err = browseDialog("folder", "Choose the demo folder", start)
		}
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		if p == "" {
			writeJSON(w, map[string]any{"cancelled": true})
			return
		}
		resp := map[string]any{"path": p}
		switch kind {
		case "demo":
			dir := filepath.Dir(p)
			scan(dir)
			settings.DemoFolder = dir
			saveSettings()
			resp["file"] = filepath.Base(p)
		case "maps":
			settings.MapsFolder = p
			saveSettings()
			mapCacheMu.Lock()
			mapCache = map[string]*MapGeo{}
			mapCacheMu.Unlock()
			mu.Lock()
			cur := folder
			cache = map[string]*demoEntry{}
			mu.Unlock()
			scan(cur)
		default:
			scan(p)
			settings.DemoFolder = p
			saveSettings()
		}
		writeJSON(w, resp)
	})
	mux.HandleFunc("/api/jump", func(w http.ResponseWriter, r *http.Request) {
		_, j, ok := getJump(r)
		if !ok {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, j)
	})
	mux.HandleFunc("/api/whatif", func(w http.ResponseWriter, r *http.Request) {
		e, j, ok := getJump(r)
		if !ok {
			http.NotFound(w, r)
			return
		}
		pre, err := strconv.ParseFloat(r.URL.Query().Get("pre"), 64)
		if err != nil || pre < 50 || pre > 2000 {
			http.Error(w, "bad pre", 400)
			return
		}
		var edge *float64
		if s := r.URL.Query().Get("edge"); s != "" {
			if v, err := strconv.ParseFloat(s, 64); err == nil {
				if v < 0 {
					v = 0
				}
				edge = &v
			}
		}
		fog, _ := strconv.Atoi(r.URL.Query().Get("fog"))
		writeJSON(w, RunWhatIf(e.demo, j, pre, edge, fog))
	})
	mux.HandleFunc("/map/", func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/map/"), ".bsp")
		g, err := LoadMap(name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(g.Raw)
	})
	mux.HandleFunc("/results.csv", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", `attachment; filename="results.csv"`)
		cw := csv.NewWriter(w)
		cw.Write([]string{"demo", "map", "type", "result", "distance", "prestrafe", "jumpoff", "block", "landing", "strafes", "efficiency", "maxspeed", "slaps", "1-frame slaps"})
		f := func(p *float64) string {
			if p == nil {
				return ""
			}
			return strconv.FormatFloat(*p, 'f', 3, 64)
		}
		for _, s := range summaries() {
			cw.Write([]string{s.Name, s.Map, s.Type, s.Result, strconv.FormatFloat(s.Dist, 'f', 3, 64), strconv.FormatFloat(s.Pre, 'f', 3, 64), f(s.Jumpoff), f(s.Block), f(s.LandEdge), strconv.Itoa(s.Strafes), strconv.FormatFloat(s.Eff, 'f', 1, 64), strconv.FormatFloat(s.Max, 'f', 1, 64), strconv.Itoa(s.Slaps), strconv.Itoa(s.OneSlaps)})
		}
		cw.Flush()
	})
	ln, err := net.Listen("tcp", "127.0.0.1:8650")
	if err != nil {
		ln, err = net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			log.Fatal(err)
		}
	}
	url := "http://" + ln.Addr().String() + "/"
	fmt.Println("DEMviewer running at", url)
	fmt.Println("Demo folder:", folder)
	if os.Getenv("DEMVIEWER_NOBROWSER") != "" {
		log.Fatal(http.Serve(ln, mux))
	}
	go http.Serve(ln, mux)
	if openUI(url) {
		saveSettings() // window closed: keep the folders for next time
		return
	}
	select {} // browser fallback: keep serving
}

func openBrowser(url string) {
	switch runtime.GOOS {
	case "windows":
		exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		exec.Command("open", url).Start()
	default:
		exec.Command("xdg-open", url).Start()
	}
}

func isDir(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

func hasDemos(dir string) bool {
	m, _ := filepath.Glob(filepath.Join(dir, "*.dem"))
	return len(m) > 0
}

// listJumps picks what the list shows for one demo. A demo with several jumps
// only lists the ones that landed with valid stats: no fails, and no take-off
// jump of a bhop (the plugin prints the bhop, not the jump before it). A demo
// with a single jump keeps it, landed or not.
func listJumps(js []*Jump) []*Jump {
	if len(js) <= 1 {
		for _, j := range js {
			j.Main = true
		}
		return js
	}
	var out []*Jump
	for _, j := range js {
		if !j.Landed {
			continue
		}
		if j.bhopNext {
			continue
		}
		out = append(out, j)
	}
	if len(out) == 0 {
		// nothing landed: show the longest attempt so the demo isn't empty
		best := js[0]
		for _, j := range js {
			if j.Distance > best.Distance {
				best = j
			}
		}
		out = []*Jump{best}
	}
	// the file name often says what the demo is ("248_sbj_name.dem"): when it
	// does, a long demo only lists jumps of that type
	if t := nameType(js[0].Demo); t != "" {
		var keep []*Jump
		for _, j := range out {
			if j.Type == t {
				keep = append(keep, j)
			}
		}
		if len(keep) == 0 {
			for _, j := range js {
				if j.Type == t && j.Landed {
					keep = append(keep, j)
				}
			}
		}
		if len(keep) > 0 {
			out = keep
		}
	}
	// every attempt stays in the list (for "all attempts"); the picked ones
	// are marked as the demo's main jumps
	for _, j := range out {
		j.Main = true
	}
	for k, j := range js {
		j.ID = k
	}
	return js
}

func nameType(name string) string {
	types := map[string]string{"ldj": "LDJ", "ladder": "LDJ", "ladderjump": "LDJ", "sbj": "SBJ", "bj": "BJ", "bhop": "BJ",
		"lj": "LJ", "hj": "HJ", "cj": "CJ", "dcj": "DCJ", "mcj": "MCJ", "wj": "WJ"}
	f := strings.FieldsFunc(strings.ToLower(strings.TrimSuffix(name, filepath.Ext(name))), func(r rune) bool {
		return r == '_' || r == '-' || r == ' ' || r == '.' || r == '[' || r == ']' || r == '(' || r == ')'
	})
	for _, w := range f {
		if t, ok := types[w]; ok {
			return t
		}
	}
	return ""
}
