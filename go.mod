module demviewer

go 1.24

require (
	github.com/jchv/go-webview2 v0.0.0
	golang.org/x/sys v0.0.0
)

require github.com/jchv/go-winloader v0.0.0 // indirect

replace github.com/jchv/go-webview2 => ./third_party/go-webview2

replace github.com/jchv/go-winloader => ./third_party/go-winloader

replace golang.org/x/sys => ./third_party/sys
