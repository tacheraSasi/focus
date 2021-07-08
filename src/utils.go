package focus

import (
	"os"

	"github.com/gookit/color"
)

type colorString string

const (
	red    colorString = "red"
	green  colorString = "green"
	yellow colorString = "yellow"
)

func printColor(c colorString, text string) string {
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		return text
	}

	switch c {
	case yellow:
		return color.HEX("#FFAB00").Sprint(text)
	case green:
		return color.HEX("#23D160").Sprint(text)
	case red:
		return color.HEX("#FF2F2F").Sprint(text)
	}

	return text
}
