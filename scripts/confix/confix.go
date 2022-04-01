package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/creachadair/atomicfile"
)

var (
	configPath = flag.String("config", "", "Config file path (required)")
	outPath    = flag.String("out", "", "Output file path (defaults to input")

	section = regexp.MustCompile(`\[([.\w]+)\]`)    // section header: [foo]
	keyVal  = regexp.MustCompile(`\s*([.\w]+)\s*=`) // key: name = value

	updateSection = map[string]string{"fastsync": "blocksync"}
	moveName      = map[string]string{".fast_sync": "blocksync.enabled"}
)

func main() {
	flag.Parse()
	if *configPath == "" {
		log.Fatal("You must specify a non-empty -config path")
	} else if *outPath == "" {
		*outPath = *configPath
	}

	in, err := os.Open(*configPath)
	if err != nil {
		log.Fatalf("Open input: %v", err)
	}
	out, err := atomicfile.New(*outPath, 0600)
	if err != nil {
		log.Fatalf("Open output: %v", err)
	}
	defer out.Cancel()
}

type chunk struct {
	Comment []string // comment lines, including "#" prefixes
	Header  string   // table or array header, including [brackets].
	Items   []*item  // key-value items
}

type item struct {
	Comment []string // comment lines, including "#" prefixes
	Key     string   // item key
	Value   string   // item value, unparsed (may be multiple lines)
}

var (
	// The beginning of a key-value expression: key = ... EOL
	// Match 1 is the key, match 2 is the value prefix.
	keyVal = regexp.MustCompile(`\s*([.\w]+)\s*=\s*(.*)$`)

	// A table heading.
)

// parseConfig loosely parses a TOML configuration file into chunks.  The parse
// does not understand the TOML value grammar, only the top-level structure of
// comments, key-value assignments, and table headings.
func parseConfig(r io.Reader) ([]*chunk, error) {
	var chunks []*chunk

	sc := bufio.NewScanner(r)

	var comment []string
	curChunk := new(chunk)
	curItem := new(item)
	for sc.Scan() {
		line := sc.Text() // without EOL

		if t := strings.TrimSpace(line); t == "" {
			// Blank line:
		}
		if true {
			// Blank line: Emit any buffered comments
			if len(com) != 0 {
				fmt.Fprintln(out, strings.Join(com, "\n"))
				com = nil
			}
			fmt.Fprintln(out, line)
		} else if strings.HasPrefix(t, "#") {
			// Comment line: Include it in the buffer.
			com = append(com, line)
		} else if m := section.FindStringSubmatchIndex(line); m != nil {
			// Rewrite section names as required.
			name := line[m[2]:m[3]]
			if repl, ok := updateSection[name]; ok {
				fmt.Fprintf(out, "%s%s%s\n", line[:m[2]], repl, line[m[3]:])
			} else {
				fmt.Fprintln(out, line)
			}
		} else if m := keyVal.FindStringSubmatchIndex(line); m != nil {
			// Replace snake case (foo_bar) with kebab case (foo-bar).
			key := line[m[2]:m[3]]
			fixed := strings.ReplaceAll(key, "_", "-")
			fmt.Fprintf(out, "%s%s%s\n", line[:m[2]], fixed, line[m[3]:])
		} else {
			fmt.Fprintln(out, line) // copy intact
		}
	}
	return chunks, sc.Err()
}
