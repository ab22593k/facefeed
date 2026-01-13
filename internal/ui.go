package theme

import (
	"fmt"
	"strings"

	"github.com/schollz/progressbar/v3"
)

func PrintHeader() {
	header := `
  ___  _  _  ___  ___  _     ___
 | _ )| || || _ )| _ )| |   | __|
 | _ \| \/ || _ \| _ \| |__ | _|
 |___/ \__/ |___/|___/|____||___|`
	fmt.Printf("%s%s%s\n", Orange, header, Reset)
	fmt.Printf("%s%sBubble: Robust CLI Facebook Publishing%s\n\n", Gray, Bold, Reset)
}

func Success(msg string) {
	fmt.Printf("%s%s✓ %s%s\n", Green, Bold, msg, Reset)
}

func Error(msg string) {
	fmt.Printf("%s%s✗ Error: %s%s\n", Orange, Bold, msg, Reset)
}

func Info(label, value string) {
	fmt.Printf("%s%s%-15s%s %s%s\n", Blue, Bold, label, Reset, Gray, value)
}

func Warning(msg string) {
	fmt.Printf("%s%s! Warning: %s%s\n", Orange, Bold, msg, Reset)
}

func NewProgressBar(size int64, description string) *progressbar.ProgressBar {
	return progressbar.NewOptions64(
		size,
		progressbar.OptionSetDescription(fmt.Sprintf("%s%s%s", Gray, description, Reset)),
		progressbar.OptionSetWriter(nil), // default to stdout
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetWidth(15),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        fmt.Sprintf("%s█%s", Orange, Reset),
			SaucerPadding: " ",
			BarStart:      "▕",
			BarEnd:        "▏",
		}),
	)
}

func PrintSection(title string) {
	fmt.Printf("\n%s%s%s%s\n", Blue, Bold, strings.ToUpper(title), Reset)
}
