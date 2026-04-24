package concurrencyanalysis

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/CivNode/go-concurrency-analysis/detectors"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/packages"
)

// Analyze runs every detector over a single source buffer. The buffer is
// written to a temporary package on disk so that golang.org/x/tools'
// packages loader can resolve types against the standard library. The
// returned findings are sorted by file position.
func Analyze(src []byte) ([]Finding, error) {
	if len(src) == 0 {
		return nil, errors.New("concurrencyanalysis: empty source")
	}
	dir, err := os.MkdirTemp("", "concurrencyanalysis-*")
	if err != nil {
		return nil, fmt.Errorf("concurrencyanalysis: mkdir temp: %w", err)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	file := filepath.Join(dir, "input.go")
	if err := os.WriteFile(file, src, 0o600); err != nil {
		return nil, fmt.Errorf("concurrencyanalysis: write temp file: %w", err)
	}
	modfile := filepath.Join(dir, "go.mod")
	if err := os.WriteFile(modfile, []byte("module concurrencyanalysisinput\n\ngo 1.22\n"), 0o600); err != nil {
		return nil, fmt.Errorf("concurrencyanalysis: write go.mod: %w", err)
	}
	return AnalyzePackageIn(dir, "./...")
}

// AnalyzePackage runs every detector over the packages loaded from the given
// go/packages patterns, resolved from the caller's current working directory.
func AnalyzePackage(patterns ...string) ([]Finding, error) {
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}
	return AnalyzePackageIn("", patterns...)
}

// AnalyzePackageIn loads packages with the given patterns, optionally from a
// specific module root directory (pass "" for the caller's cwd), and runs
// every detector.
func AnalyzePackageIn(dir string, patterns ...string) ([]Finding, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles |
			packages.NeedImports | packages.NeedTypes | packages.NeedSyntax |
			packages.NeedTypesInfo | packages.NeedDeps,
		Dir:   dir,
		Tests: false,
	}
	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		return nil, fmt.Errorf("concurrencyanalysis: load packages: %w", err)
	}
	if packages.PrintErrors(pkgs) > 0 {
		return nil, errors.New("concurrencyanalysis: package load errors")
	}

	var findings []Finding
	for _, pkg := range pkgs {
		for _, a := range detectors.All() {
			got, err := runAnalyzerOnPackage(a, pkg)
			if err != nil {
				return nil, fmt.Errorf("concurrencyanalysis: analyzer %s on %s: %w", a.Name, pkg.PkgPath, err)
			}
			findings = append(findings, got...)
		}
	}

	sort.Slice(findings, func(i, j int) bool {
		a, b := findings[i].Pos, findings[j].Pos
		if a.Filename != b.Filename {
			return a.Filename < b.Filename
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		if a.Column != b.Column {
			return a.Column < b.Column
		}
		return findings[i].Kind < findings[j].Kind
	})
	return findings, nil
}

// runAnalyzerOnPackage drives one analyzer (plus its Requires) on a single
// loaded package and returns only the findings produced by the target
// analyzer itself.
func runAnalyzerOnPackage(target *analysis.Analyzer, pkg *packages.Package) ([]Finding, error) {
	results := map[*analysis.Analyzer]any{}
	var targetFindings []Finding

	var drive func(*analysis.Analyzer) error
	drive = func(a *analysis.Analyzer) error {
		if _, done := results[a]; done {
			return nil
		}
		for _, r := range a.Requires {
			if err := drive(r); err != nil {
				return err
			}
		}
		pass := &analysis.Pass{
			Analyzer:  a,
			Fset:      pkg.Fset,
			Files:     pkg.Syntax,
			Pkg:       pkg.Types,
			TypesInfo: pkg.TypesInfo,
			ResultOf:  results,
			Report: func(d analysis.Diagnostic) {
				if a == target {
					targetFindings = append(targetFindings, Finding{
						Kind:    kindFromAnalyzerName(a.Name),
						Message: d.Message,
						Pos:     pkg.Fset.Position(d.Pos),
					})
				}
			},
		}
		r, err := a.Run(pass)
		if err != nil {
			return err
		}
		results[a] = r
		return nil
	}
	if err := drive(target); err != nil {
		return nil, err
	}
	return targetFindings, nil
}

func kindFromAnalyzerName(name string) Kind {
	switch name {
	case "Deadlock":
		return Deadlock
	case "GoroutineLeak":
		return GoroutineLeak
	case "UnsyncShared":
		return UnsyncShared
	case "DeferInLoop":
		return DeferInLoop
	case "UnbufferedChanDeadlock":
		return UnbufferedChanDeadlock
	}
	return 0
}
