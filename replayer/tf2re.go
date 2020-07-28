package replayer

import (
	"regexp"
	"strings"
)

func RegexpFromTimeLayout(layout string) *regexp.Regexp {
	p := layout

	if p == "unix" || p == "unixmilli" || p == "unixnano" {
		return regexp.MustCompile(`\d+`)
	} else if p == "unix.milli" || p == "unix.nano" {
		return regexp.MustCompile(`\d+\.\d+`)
	}

	if strings.Contains(p, "Monday") {
		p = strings.ReplaceAll(p, "Monday", "(?:Monday|Tuesday|Wednesday|Thursday|Friday|Saturday|Sunday)")
	} else if strings.Contains(p, "Mon") {
		p = strings.ReplaceAll(p, "Mon", "(?:Mon|Tue|Wed|Thu|Fri|Sat|Sun)")
	}

	if strings.Contains(p, "January") {
		p = strings.ReplaceAll(p, "January", "(?:January|February|March|April|May|June|July|August|September|October|November|December)")
	} else if strings.Contains(p, "Jan") {
		p = strings.ReplaceAll(p, "Jan", "(?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)")
	}

	p = strings.ReplaceAll(p, "2006", `\d{4}`)
	if strings.Contains(p, "2006") {
		p = strings.ReplaceAll(p, "2006", `\d{4}`)
	} else {
		p = strings.ReplaceAll(p, "06", `\d{2}`)
	}

	if strings.Contains(p, "02") {
		p = strings.ReplaceAll(p, "02", `\d{2}`)
	} else if strings.Contains(p, "_2") {
		p = strings.ReplaceAll(p, "_2", `(?:\d{2}| \d)`)
	} else {
		p = strings.ReplaceAll(p, "2", `D_ONE_TWO`)
	}

	if strings.Contains(p, "03") {
		p = strings.ReplaceAll(p, "03", `\d{2}`)
	} else if strings.Contains(p, "_3") {
		p = strings.ReplaceAll(p, "_3", `(?:\d{2}| \d)`)
	} else if strings.Contains(p, "3") {
		p = strings.ReplaceAll(p, "3", `D_ONE_TWO`)
	} else {
		p = strings.ReplaceAll(p, "15", `\d{2}`)
	}

	p = strings.ReplaceAll(p, "04", `\d{2}`)
	p = strings.ReplaceAll(p, "05", `\d{2}`)

	p = strings.ReplaceAll(p, "MST", "[A-Z]{3}")

	if strings.Contains(p, "-0700") {
		p = strings.ReplaceAll(p, "-0700", `(?:\+|-)\d{4}`)
	} else if strings.Contains(p, "-07:00") {
		p = strings.ReplaceAll(p, "-07:00", `(?:\+|-)\d{2}:\d{2}`)
	} else if strings.Contains(p, "-07") {
		p = strings.ReplaceAll(p, "-07", `(?:\+|-)\d{2}`)
	}

	if strings.Contains(p, "Z0700") {
		p = strings.ReplaceAll(p, "Z0700", `(?:Z|(?:\+|-)\d{4}))`)
	} else if strings.Contains(p, "Z07:00") {
		p = strings.ReplaceAll(p, "Z07:00", `(?:Z|(?:\+|-)\d{2}:\d{2})`)
	} else if strings.Contains(p, "Z07") {
		p = strings.ReplaceAll(p, "Z07", `(?:Z|(?:\+|-)\d{2})`)
	}

	if strings.Contains(p, "01") {
		p = strings.ReplaceAll(p, "01", `\d{2}`)
	} else if strings.Contains(p, "_1") {
		p = strings.ReplaceAll(p, "_1", `(?:\d{2}| \d)`)
	} else {
		p = strings.ReplaceAll(p, "1", `D_ONE_TWO`)
	}

	p = strings.ReplaceAll(p, "0", `\d`)
	p = strings.ReplaceAll(p, "9", `\d`)
	p = strings.ReplaceAll(p, "D_ONE_TWO", `\d{1,2}`)
	p = strings.ReplaceAll(p, "PM", `(?:AM|PM)`)
	p = strings.ReplaceAll(p, ".", `\.`)

	return regexp.MustCompile(p)
}
