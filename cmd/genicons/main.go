// Command genicons generates action and ball PNG icons into assets/icons/.
package main

import (
	"fmt"
	"os"

	"github.com/darkweaver87/courtdraw/internal/ui/icon"
)

func main() {
	icons := map[string]func(){
		"assets/icons/dribble-action.png": func() { write("assets/icons/dribble-action.png", icon.GenerateActionIcon("dribble")) },
		"assets/icons/pass-action.png":    func() { write("assets/icons/pass-action.png", icon.GenerateActionIcon("pass")) },
		"assets/icons/cut-action.png":     func() { write("assets/icons/cut-action.png", icon.GenerateActionIcon("cut")) },
		"assets/icons/screen-action.png":  func() { write("assets/icons/screen-action.png", icon.GenerateActionIcon("screen")) },
		"assets/icons/shot-action.png":    func() { write("assets/icons/shot-action.png", icon.GenerateActionIcon("shot")) },
		"assets/icons/handoff-action.png": func() { write("assets/icons/handoff-action.png", icon.GenerateActionIcon("handoff")) },
		"assets/icons/ball.png":           func() { write("assets/icons/ball.png", icon.GenerateBallIcon()) },
	}
	for _, gen := range icons {
		gen()
	}
}

func write(path string, res interface{ Content() []byte }) {
	if err := os.WriteFile(path, res.Content(), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "write %s: %v\n", path, err)
		os.Exit(1)
	}
	fmt.Println("generated", path)
}
