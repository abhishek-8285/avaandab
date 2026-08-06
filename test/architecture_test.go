package test

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
)

type Package struct {
	ImportPath string
	Imports    []string
}

func TestArchitecture_ImportBoundaries(t *testing.T) {
	cmd := exec.Command("go", "list", "-json", "./...")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to run go list: %v", err)
	}

	decoder := json.NewDecoder(&out)
	for decoder.More() {
		var pkg Package
		if err := decoder.Decode(&pkg); err != nil {
			t.Fatalf("failed to decode pkg json: %v", err)
		}

		// 1. Enforce Domain Boundary Constraints
		if strings.Contains(pkg.ImportPath, "/domain") {
			for _, imp := range pkg.Imports {
				if imp == "database/sql" {
					t.Errorf("Domain package %s is forbidden from importing database/sql", pkg.ImportPath)
				}
				if imp == "net/http" {
					t.Errorf("Domain package %s is forbidden from importing net/http", pkg.ImportPath)
				}
				if imp == "html/template" {
					t.Errorf("Domain package %s is forbidden from importing html/template", pkg.ImportPath)
				}
				if strings.Contains(imp, "transport-app/internal/platform") {
					t.Errorf("Domain package %s is forbidden from importing internal/platform", pkg.ImportPath)
				}
			}
		}

		// 2. Enforce Application Boundary Constraints
		if strings.Contains(pkg.ImportPath, "/application") {
			for _, imp := range pkg.Imports {
				if imp == "net/http" {
					t.Errorf("Application package %s is forbidden from importing net/http", pkg.ImportPath)
				}
				if imp == "html/template" {
					t.Errorf("Application package %s is forbidden from importing html/template", pkg.ImportPath)
				}
			}
		}

		// 3. Enforce Inter-Module Boundaries (No cross-importing other module's internals)
		// Core modules: booking, trip, driver, vehicle, invoice, payment
		modules := []string{"booking", "trip", "driver", "vehicle", "invoice", "payment"}
		for _, mod := range modules {
			// If this package belongs to a module's domain or application layers
			if strings.Contains(pkg.ImportPath, "transport-app/internal/"+mod+"/domain") ||
				strings.Contains(pkg.ImportPath, "transport-app/internal/"+mod+"/application") {
				for _, imp := range pkg.Imports {
					// It must not import any internal folder of another module
					for _, otherMod := range modules {
						if mod == otherMod {
							continue
						}
						// Forbidden: importing otherMod's domain, application, or infrastructure
						if strings.Contains(imp, "transport-app/internal/"+otherMod+"/domain") ||
							strings.Contains(imp, "transport-app/internal/"+otherMod+"/application") ||
							strings.Contains(imp, "transport-app/internal/"+otherMod+"/infrastructure") {
							t.Errorf("Package %s violates module isolation by directly importing %s (use %s's facade instead)", pkg.ImportPath, imp, otherMod)
						}
					}
				}
			}
		}
	}
}
