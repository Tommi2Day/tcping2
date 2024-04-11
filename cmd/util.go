package cmd

import (
	"github.com/fatih/color"
)

var (
	cyan = color.New(color.FgCyan).SprintfFunc()
	// blue   = color.New(color.FgBlue).SprintfFunc()
	green  = color.New(color.FgGreen).SprintfFunc()
	yellow = color.New(color.FgYellow).SprintfFunc()
	red    = color.New(color.FgRed).SprintfFunc()
)
