// Command autogo runs the auto package bot on stdin/stdout (mechfight/1).
package main

import (
	"os"

	"mechbot/auto"
)

func main() {
	os.Exit(auto.Run())
}
