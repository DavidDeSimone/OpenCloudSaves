package dialog

import (
	"github.com/sqweek/dialog/cocoa"
)

func (b *MsgBuilder) yesNo() bool {
	return cocoa.YesNoDlg(b.Msg, b.Dlg.Title)
}

func (b *MsgBuilder) info() {
	cocoa.InfoDlg(b.Msg, b.Dlg.Title)
}

func (b *MsgBuilder) error() {
	cocoa.ErrorDlg(b.Msg, b.Dlg.Title)
}

func (b *FileBuilder) load() (string, error) {
	return b.run(false)
}

func (b *FileBuilder) save() (string, error) {
	return b.run(true)
}

func (b *FileBuilder) run(save bool) (string, error) {
	star := false
	var exts []string
	for _, filt := range b.Filters {
		for _, ext := range filt.Extensions {
			if ext == "*" {
				star = true
			} else {
				exts = append(exts, ext)
			}
		}
	}
	if star && save {
		/* OSX doesn't allow the user to switch visible file types/extensions. Also
		** NSSavePanel's allowsOtherFileTypes property has no effect for an open
		** dialog, so if "*" is a possible extension we must always show all files. */
		exts = nil
	}
	f, err := cocoa.FileDlg(save, b.Dlg.Title, exts, star, b.StartDir, b.StartFile)
	if f == "" && err == nil {
		return "", ErrCancelled
	}
	return f, err
}

func (b *DirectoryBuilder) browse() (string, error) {
	f, err := cocoa.DirDlg(b.Dlg.Title, b.StartDir)
	if f == "" && err == nil {
		return "", ErrCancelled
	}
	return f, err
}
