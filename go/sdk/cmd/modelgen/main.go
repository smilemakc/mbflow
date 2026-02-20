// Command modelgen generates SDK model files from the canonical pkg/models definitions.
//
// Usage:
//
//	go run ./cmd/modelgen                          # generate files
//	go run ./cmd/modelgen --verify                 # check files are up-to-date (CI)
//	go run ./cmd/modelgen --source-dir=... --output-dir=...
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

func main() {
	verify := flag.Bool("verify", false, "Check generated files are up-to-date (exit 1 if stale)")
	sourceDir := flag.String("source-dir", "", "Path to pkg/models/ directory")
	outputDir := flag.String("output-dir", "", "Path to sdk/go/models/ directory")
	flag.Parse()

	// Auto-detect paths relative to this source file's location.
	if *sourceDir == "" || *outputDir == "" {
		base := detectBaseDir()
		if *sourceDir == "" {
			*sourceDir = filepath.Join(base, "..", "..", "..", "..", "pkg", "models")
		}
		if *outputDir == "" {
			*outputDir = filepath.Join(base, "..", "..", "models")
		}
	}

	hasError := false

	for _, rule := range fileRules {
		srcPath := filepath.Join(*sourceDir, rule.SourceFile)
		outPath := filepath.Join(*outputDir, rule.OutputFile)

		parsed, err := ParseSourceFile(srcPath, rule)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error parsing %s: %v\n", srcPath, err)
			os.Exit(1)
		}

		output, err := Generate(rule, parsed)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error generating %s: %v\n", rule.OutputFile, err)
			os.Exit(1)
		}

		if *verify {
			existing, err := os.ReadFile(outPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "cannot read %s: %v\n", outPath, err)
				hasError = true
				continue
			}
			if string(existing) != string(output) {
				fmt.Fprintf(os.Stderr, "STALE: %s differs from generated output\n", outPath)
				hasError = true
			} else {
				fmt.Printf("ok: %s\n", rule.OutputFile)
			}
		} else {
			if err := os.WriteFile(outPath, output, 0644); err != nil {
				fmt.Fprintf(os.Stderr, "error writing %s: %v\n", outPath, err)
				os.Exit(1)
			}
			fmt.Printf("generated: %s\n", rule.OutputFile)
		}
	}

	if *verify && hasError {
		fmt.Fprintln(os.Stderr, "\nGenerated files are out of date. Run 'make generate' to update.")
		os.Exit(1)
	}
}

// detectBaseDir returns the directory of this source file, used for relative path resolution.
func detectBaseDir() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		// Fallback: assume working directory is sdk/go/
		return filepath.Join(".", "cmd", "modelgen")
	}
	return filepath.Dir(filename)
}
