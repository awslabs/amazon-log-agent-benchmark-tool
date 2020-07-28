package replayer

import (
	"testing"
	"time"
)

func TestRegexpFromTimeLayout(t *testing.T) {
	layouts := []string{
		time.ANSIC,
		time.UnixDate,
		time.RubyDate,
		time.RFC822,
		time.RFC822Z,
		time.RFC850,
		time.RFC1123,
		time.RFC1123Z,
		time.RFC3339,
		time.RFC3339Nano,
		time.Kitchen,
		time.Stamp,
		time.StampMilli,
		time.StampMicro,
		time.StampNano,
		"15:04:05 MST 2006/01/02",
	}

	now := time.Now()
	for i := 0; i < 1000; i++ {
		tt := now.Add(time.Duration(i) * time.Millisecond)
		for _, l := range layouts {
			re := RegexpFromTimeLayout(l)
			st := tt.Format(l)
			sf := re.FindString(st)
			if st != sf {
				t.Errorf("Regexp `%v` generated from layout `%v` does not match formated value `%v`, match was `%v`", re, l, st, sf)
			}
		}
	}

	for i := 0; i < 60; i++ {
		tt := now.Add(time.Duration(i) * time.Second)
		for _, l := range layouts {
			re := RegexpFromTimeLayout(l)
			st := tt.Format(l)
			sf := re.FindString(st)
			if st != sf {
				t.Errorf("Regexp `%v` generated from layout `%v` does not match formated value `%v`, match was `%v`", re, l, st, sf)
			}
		}
	}

	for i := 0; i < 60; i++ {
		tt := now.Add(time.Duration(i) * time.Minute)
		for _, l := range layouts {
			re := RegexpFromTimeLayout(l)
			st := tt.Format(l)
			sf := re.FindString(st)
			if st != sf {
				t.Errorf("Regexp `%v` generated from layout `%v` does not match formated value `%v`, match was `%v`", re, l, st, sf)
			}
		}
	}

	for i := 0; i < 24; i++ {
		tt := now.Add(time.Duration(i) * time.Hour)
		for _, l := range layouts {
			re := RegexpFromTimeLayout(l)
			st := tt.Format(l)
			sf := re.FindString(st)
			if st != sf {
				t.Errorf("Regexp `%v` generated from layout `%v` does not match formated value `%v`, match was `%v`", re, l, st, sf)
			}
		}
	}

	for i := 0; i < 365; i++ {
		tt := now.AddDate(0, 0, i)
		for _, l := range layouts {
			re := RegexpFromTimeLayout(l)
			st := tt.Format(l)
			sf := re.FindString(st)
			if st != sf {
				t.Errorf("Regexp `%v` generated from layout `%v` does not match formated value `%v`, match was `%v`", re, l, st, sf)
			}
		}
	}

	for i := 0; i < 100; i++ {
		tt := now.AddDate(i, 0, 0)
		for _, l := range layouts {
			re := RegexpFromTimeLayout(l)
			st := tt.Format(l)
			sf := re.FindString(st)
			if st != sf {
				t.Errorf("Regexp `%v` generated from layout `%v` does not match formated value `%v`, match was `%v`", re, l, st, sf)
			}
		}
	}
}
