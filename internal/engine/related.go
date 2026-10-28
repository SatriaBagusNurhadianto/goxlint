package engine

import (
	"fmt"
	"go/token"

	"github.com/egoisutolabs/goxlint"
)

func resolveRelated(fset *token.FileSet, related goxlint.Related, packageFiles map[*token.File]File) (Related, error) {
	pos, end := related.Pos, related.End
	if related.Node != nil {
		pos, end = related.Node.Pos(), related.Node.End()
	}
	if end == token.NoPos {
		end = pos
	}
	tf := fset.File(pos)
	file, loaded := packageFiles[tf]
	if !loaded || !pos.IsValid() || end < pos || end > token.Pos(tf.Base()+tf.Size()) {
		return Related{}, fmt.Errorf("related positions must be ordered and within a loaded file of the same package")
	}
	start, finish := fset.PositionFor(pos, false), fset.PositionFor(end, false)
	start.Filename, finish.Filename = file.Filename, file.Filename
	return Related{Filename: file.Filename, Start: start, End: finish, Message: related.Message}, nil
}
