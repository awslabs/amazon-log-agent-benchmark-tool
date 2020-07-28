package replayer

import (
	"bufio"
	"io"
	"log"
	"math/rand"
	"regexp"
	"strings"
	"time"
)

type Opt func(g *replayer)

type replayer struct {
	r          *bufio.Reader
	w          io.Writer
	mlStart    *regexp.Regexp
	timeLayout string
	timeRegexp *regexp.Regexp
	nextLine   []byte
	rate       float64

	rand *rand.Rand
}

func OptRate(rate float64) func(r *replayer) {
	return func(r *replayer) {
		r.rate = rate
	}
}

func OptTimeLayout(tf string) func(r *replayer) {
	return func(r *replayer) {
		// Find group start
		gs := 0
		for s := 0; s < len(tf); s++ {
			idx := strings.Index(tf[s:], "(")
			if idx < 0 {
				break
			}
			s += idx
			if len(tf) < s+3 || tf[s:s+3] != "(?:" {
				gs = s
				break
			}
		}

		// Find group end if there is a group
		ge := 0
		if gs > 0 {
			for e := gs + 1; e < len(tf); e++ {
				if e+1 < len(tf)-1 && tf[e] == '\\' { // Allow () to appear withing time layout, escape with '\'
					e++
				}
				if tf[e] == ')' {
					ge = e
					break
				}
			}
		}

		// If there is a group, extract it out, otherwise consider the whole string as time layout
		if ge == 0 {
			r.timeLayout = tf
			r.timeRegexp = RegexpFromTimeLayout(r.timeLayout)
		} else {
			r.timeLayout = tf[gs+1 : ge]
			tre := RegexpFromTimeLayout(r.timeLayout)
			r.timeRegexp = regexp.MustCompile(tf[:gs+1] + tre.String() + tf[ge:])
		}

	}
}

func OptMultilineStartPattern(p string) func(r *replayer) {
	return func(r *replayer) {
		r.mlStart = regexp.MustCompile(p)
	}
}

func NewReplayer(src io.Reader, dest io.Writer, opts ...Opt) *replayer {
	r := &replayer{
		r:    bufio.NewReaderSize(src, 5*1024*1024),
		w:    dest,
		rand: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	for _, opt := range opts {
		opt(r)
	}

	// Replace timestamp placeholder in multiline start regexp
	if r.mlStart != nil && strings.Contains(r.mlStart.String(), "{timestamp}") && r.timeRegexp != nil {
		p := strings.ReplaceAll(r.mlStart.String(), "{timestamp}", r.timeRegexp.String())
		r.mlStart = regexp.MustCompile(p)
	}

	go r.start()

	return r
}

func (r replayer) nextEvent() ([]byte, error) {
	if r.nextLine == nil {
		nl, err := r.r.ReadBytes('\n')
		if err != nil {
			return nl, err
		}
		r.nextLine = nl
	}

	line := r.nextLine
	if r.mlStart == nil {
		r.nextLine = nil
		return line, nil
	}

	for {
		nl, err := r.r.ReadBytes('\n')
		if err != nil {
			r.nextLine = nil
			return append(line, nl...), err
		}
		if r.mlStart.Match(nl) {
			r.nextLine = nl
			break
		}
		line = append(line, nl...)
	}

	return line, nil
}

func (r replayer) start() {
	var st, t0 time.Time
	for {
		evt, readErr := r.nextEvent()
		if evt == nil {
			log.Println("Replayer finished")
			break
		}

		if r.timeRegexp != nil {
			evt = r.replaceTimestampAndWait(evt, &st, &t0)
		} else if r.rate != 0 {
			dt := time.Duration(r.rand.ExpFloat64() / (r.rate * 100) * float64(time.Second) * 100)
			time.Sleep(dt)
		}

		_, err := r.w.Write(evt)
		if err != nil {
			log.Printf("Replayer failed to write event with err: %v, stopped, event was:\n'%s'", err, evt)
			break
		}

		if readErr == io.EOF {
			log.Printf("Replayer reached EOF of source file, stopped")
			break
		}
		if readErr != nil {
			log.Printf("Replayer encourtered error: %v, stopped", readErr)
			break
		}
	}
}

func (r *replayer) replaceTimestampAndWait(evt []byte, st, t0 *time.Time) []byte {
	match := r.timeRegexp.FindSubmatchIndex(evt)
	if len(match) == 0 {
		return evt
	}

	es, ee := match[len(match)-2], match[len(match)-1]
	ts := string(evt[es:ee])
	t, err := time.Parse(r.timeLayout, ts)
	if err != nil {
		log.Printf("Replayer failed to parse timestamp '%v' with layout '%v', error: %v", ts, r.timeLayout, err)
	} else {
		et := time.Now()

		if t0.IsZero() {
			*st = time.Now()
			*t0 = t
		}

		now := time.Now()
		dt := t.Sub(*t0) - now.Sub(*st)
		if dt > 0 {
			et = now.Add(dt)
		}

		// Replace the timestamp
		nevt := append([]byte{}, evt[:es]...)
		nevt = et.AppendFormat(nevt, r.timeLayout)
		nevt = append(nevt, evt[ee:]...)

		evt = nevt

		// Delay the event if needed
		if et.After(time.Now()) {
			time.Sleep(time.Until(et))
		}
	}
	//l := 200
	//if len(evt) < l {
	//l = len(evt)
	//}
	//fmt.Printf("X evt: %s\n", evt[:l])
	return evt
}
