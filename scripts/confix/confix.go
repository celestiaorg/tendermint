package main

import (
	"bufio"
	"flag"
	"fmt"
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

	// A buffer of recent line comments. When we find a section, any comments in
	// the buffer are attributed to the section. Comments before a blank line or
	// the end of file are not attributed.
	var com []string

	sc := bufio.NewScanner(in)
	for sc.Scan() {
		line := sc.Text()
		if t := strings.TrimSpace(line); t == "" {
			// Blank line: Emit any buffered comments, followed by the line.
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
	if err := sc.Err(); err != nil {
		log.Fatalf("Scanning input: %v", err)
	}

	// Flush out any remaining buffered comment lines.
	if len(com) != 0 {
		fmt.Fprintln(out, strings.Join(com, "\n"))
	}

	if err := out.Close(); err != nil {
		log.Fatalf("Closing output: %v", err)
	}
}
