//go:build !windows && !android && ((!darwin && !freebsd) || cgo)

package main

import _ "embed"

//go:embed icon.png
var iconPNG []byte

func getIcon() []byte {
	return iconPNG
}
