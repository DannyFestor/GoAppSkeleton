package refpatch

import (
	"bytes"
	"go/token"
	"io/fs"
	"path"
	"strings"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
	"github.com/dave/dst/decorator/resolver/goast"
	"github.com/dave/dst/decorator/resolver/guess"

	"example.com/enginecomp/vfs"
)

const referenceModule = "example.com/myapp"

// guess names a package after its last path element, which is "v5" for chi; every such package needs an entry.
var packageNames = guess.WithMap(map[string]string{
	"github.com/go-chi/chi/v5": "chi",
})

type goFiles map[string]*dst.File

func loadGoFiles(root fs.FS) (goFiles, error) {
	files := goFiles{}
	return files, files.addFrom(root)
}

func (files goFiles) addFrom(root fs.FS) error {
	return fs.WalkDir(root, ".", func(filePath string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		src, err := fs.ReadFile(root, filePath)
		if err != nil {
			return err
		}
		file, err := decorate(packagePath(filePath), string(src))
		files[filePath] = file
		return err
	})
}

func (files goFiles) restore(module string) (vfs.Tree, error) {
	tree := vfs.Tree{}
	for filePath, file := range files {
		renameModule(file, module)
		src, err := restore(packagePath(filePath), file)
		if err != nil {
			return nil, err
		}
		tree[filePath] = src
	}
	return tree, nil
}

func packagePath(filePath string) string {
	return referenceModule + "/" + path.Dir(filePath)
}

func decorate(pkgPath, src string) (*dst.File, error) {
	dec := decorator.NewDecoratorWithImports(token.NewFileSet(), pkgPath, goast.WithResolver(packageNames))
	return dec.Parse(src)
}

func restore(pkgPath string, file *dst.File) ([]byte, error) {
	restorer := decorator.NewRestorerWithImports(pkgPath, packageNames)
	var buf bytes.Buffer
	if err := restorer.Fprint(&buf, file); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// The reference is written against referenceModule; every Ident carries its import path, so renaming is a path rewrite.
func renameModule(file *dst.File, module string) {
	dst.Inspect(file, func(node dst.Node) bool {
		if ident, ok := node.(*dst.Ident); ok && strings.HasPrefix(ident.Path, referenceModule+"/") {
			ident.Path = module + strings.TrimPrefix(ident.Path, referenceModule)
		}
		return true
	})
}
