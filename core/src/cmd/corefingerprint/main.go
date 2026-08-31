// Command corefingerprint prints the fingerprint of the Core's build inputs.
//
// The canonical build stamps what this prints into the Core binary, and the
// test gate recomputes it from the working tree. Both sides call the same code
// so the two values cannot drift through two implementations of one hash.
//
// Usage: corefingerprint [core/src directory]
package main

import (
	"fmt"
	"os"

	"github.com/sipeed/picoclaw/pkg/coresource"
)

func main() {
	root := "."
	if len(os.Args) > 1 {
		root = os.Args[1]
	}
	fingerprint, err := coresource.Fingerprint(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "corefingerprint: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(fingerprint)
}
