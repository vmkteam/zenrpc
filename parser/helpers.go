package parser

import (
	"fmt"
	"go/ast"
	"path"
	"strings"

	"github.com/thoas/go-funk"
	"golang.org/x/tools/go/packages"
)

type PackageFiles struct {
	PackagePath string
	PackageName string

	AstFiles []*ast.File
}

func filterFile(filepath string) bool {
	if !strings.HasSuffix(filepath, goFileSuffix) ||
		strings.HasSuffix(filepath, GenerateFileSuffix) || strings.HasSuffix(filepath, testFileSuffix) {
		return false
	}
	return true
}

func getDependenciesFilenames(dir string) ([]string, error) {
	pkgs, err := loadPackage(dir)
	if err != nil {
		return nil, err
	}
	goFiles := make([]string, 0, len(pkgs))
	for _, pack := range pkgs {
		goFiles = append(goFiles, goFilesFromPackage(pack)...)
		for _, childPack := range pack.Imports {
			goFiles = append(goFiles, goFilesFromPackage(childPack)...)
		}
	}
	return funk.UniqString(goFiles), nil
}

func GetDependenciesAstFiles(filename string) ([]PackageFiles, error) {
	pkgs, err := loadPackageWithSyntax(path.Dir(filename))
	if err != nil {
		return nil, err
	}
	pfs := make([]PackageFiles, 0, len(pkgs))
	done := map[string]bool{}
	for _, pkg := range pkgs {
		if _, ok := done[pkg.PkgPath]; ok {
			continue
		}

		pfs = append(pfs, PackageFiles{
			PackagePath: pkg.PkgPath,
			PackageName: pkg.Name,
			AstFiles:    pkg.Syntax,
		})

		done[pkg.PkgPath] = true

		for _, childPack := range pkg.Imports {
			if _, ok := done[childPack.PkgPath]; ok {
				continue
			}

			pfs = append(pfs, PackageFiles{
				PackagePath: childPack.PkgPath,
				PackageName: childPack.Name,
				AstFiles:    childPack.Syntax,
			})

			done[childPack.PkgPath] = true
		}
	}
	return pfs, nil
}

func goFilesFromPackage(pkg *packages.Package) []string {
	return funk.FilterString(pkg.GoFiles, filterFile)
}

func EntryPointPackageName(filename string) (string, string, error) {
	pkgs, err := loadPackage(path.Dir(filename))
	if err != nil {
		return "", "", err
	}
	for _, pack := range pkgs {
		return pack.Name, pack.PkgPath, nil
	}
	return "", "", fmt.Errorf("package not found for entry point")
}

func loadPackage(path string) ([]*packages.Package, error) {
	return packages.Load(&packages.Config{
		Mode: packages.NeedImports | packages.NeedFiles | packages.NeedName,
	}, path)
}

func loadPackageWithSyntax(path string) ([]*packages.Package, error) {
	return packages.Load(&packages.Config{
		Mode: packages.NeedImports |
			packages.NeedFiles |
			packages.NeedName |
			packages.NeedSyntax,
	}, path)
}
