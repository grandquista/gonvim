package editor

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/therecipe/qt/core"
	"github.com/therecipe/qt/svg"
	"github.com/therecipe/qt/widgets"
)

// Statusline is
type Statusline struct {
	ws     *Workspace
	widget *widgets.QWidget
	bg     *RGBA

	borderTopWidth int
	paddingLeft    int
	paddingRight   int
	margin         int
	height         int

	pos      *StatuslinePos
	mode     *StatusMode
	file     *StatuslineFile
	filetype *StatuslineFiletype
	git      *StatuslineGit
	encoding *StatuslineEncoding
	lint     *StatuslineLint
	updates  chan []interface{}
}

// StatuslineLint is
type StatuslineLint struct {
	s          *Statusline
	errors     int
	warnings   int
	widget     *widgets.QWidget
	okIcon     *svg.QSvgWidget
	errorIcon  *svg.QSvgWidget
	warnIcon   *svg.QSvgWidget
	okLabel    *widgets.QLabel
	errorLabel *widgets.QLabel
	warnLabel  *widgets.QLabel
	svgLoaded  bool
}

// StatuslineFile is
type StatuslineFile struct {
	s           *Statusline
	file        string
	fileType    string
	widget      *widgets.QWidget
	fileLabel   *widgets.QLabel
	folderLabel *widgets.QLabel
	icon        *svg.QSvgWidget
	base        string
	dir         string
}

// StatuslineFiletype is
type StatuslineFiletype struct {
	filetype string
	label    *widgets.QLabel
}

// StatuslinePos is
type StatuslinePos struct {
	ln    int
	col   int
	label *widgets.QLabel
	text  string
}

// StatusMode is
type StatusMode struct {
	s     *Statusline
	label *widgets.QLabel
	mode  string
	text  string
	bg    *RGBA
}

// StatuslineGit is
type StatuslineGit struct {
	s         *Statusline
	branch    string
	file      string
	widget    *widgets.QWidget
	label     *widgets.QLabel
	icon      *svg.QSvgWidget
	svgLoaded bool
	hidden    bool
}

// StatuslineEncoding is
type StatuslineEncoding struct {
	encoding string
	label    *widgets.QLabel
}

func initStatuslineNew() *Statusline {
	widget := widgets.NewQWidget(nil, 0)
	widget.SetContentsMargins(0, 1, 0, 0)
	layout := newVFlowLayout(8, 8, 1, 3, 0)
	widget.SetLayout(layout)
	widget.SetObjectName("statusline")
	widget.SetStyleSheet(`
	QWidget#statusline {
		border-top: 2px solid rgba(0, 0, 0, 1);
		background-color: rgba(24, 29, 34, 1);
	}
	* {
		color: rgba(205, 211, 222, 1);
	}
	`)

	s := &Statusline{
		widget:  widget,
		updates: make(chan []interface{}, 1000),
	}

	modeLabel := widgets.NewQLabel(nil, 0)
	modeLabel.SetContentsMargins(4, 1, 4, 1)
	modeLayout := widgets.NewQHBoxLayout()
	modeLayout.AddWidget(modeLabel, 0, 0)
	modeLayout.SetContentsMargins(0, 0, 0, 0)
	modeWidget := widgets.NewQWidget(nil, 0)
	modeWidget.SetContentsMargins(0, 4, 0, 4)
	modeWidget.SetLayout(modeLayout)
	mode := &StatusMode{
		s:     s,
		label: modeLabel,
	}
	s.mode = mode

	gitIcon := svg.NewQSvgWidget(nil)
	gitIcon.SetFixedSize2(14, 14)
	gitLabel := widgets.NewQLabel(nil, 0)
	gitLabel.SetContentsMargins(0, 0, 0, 0)
	gitLayout := widgets.NewQHBoxLayout()
	gitLayout.SetContentsMargins(0, 0, 0, 0)
	gitLayout.SetSpacing(2)
	gitLayout.AddWidget(gitIcon, 0, 0)
	gitLayout.AddWidget(gitLabel, 0, 0)
	gitWidget := widgets.NewQWidget(nil, 0)
	gitWidget.SetContentsMargins(0, 0, 0, 0)
	gitWidget.SetLayout(gitLayout)
	gitWidget.Hide()
	git := &StatuslineGit{
		s:      s,
		widget: gitWidget,
		icon:   gitIcon,
		label:  gitLabel,
	}
	s.git = git

	fileIcon := svg.NewQSvgWidget(nil)
	fileIcon.SetFixedSize2(14, 14)
	fileLabel := widgets.NewQLabel(nil, 0)
	fileLabel.SetContentsMargins(0, 0, 0, 0)
	folderLabel := widgets.NewQLabel(nil, 0)
	folderLabel.SetContentsMargins(0, 0, 0, 0)
	folderLabel.SetStyleSheet("color: #838383;")
	folderLabel.SetContentsMargins(0, 0, 0, 0)
	fileLayout := widgets.NewQHBoxLayout()
	fileLayout.SetContentsMargins(0, 0, 0, 0)
	fileLayout.SetSpacing(2)
	fileLayout.AddWidget(fileIcon, 0, 0)
	fileLayout.AddWidget(fileLabel, 0, 0)
	fileLayout.AddWidget(folderLabel, 0, 0)
	fileWidget := widgets.NewQWidget(nil, 0)
	fileWidget.SetContentsMargins(0, 0, 0, 0)
	fileWidget.SetLayout(fileLayout)
	file := &StatuslineFile{
		s:           s,
		icon:        fileIcon,
		widget:      fileWidget,
		fileLabel:   fileLabel,
		folderLabel: folderLabel,
	}
	s.file = file

	encodingLabel := widgets.NewQLabel(nil, 0)
	encodingLabel.SetContentsMargins(0, 0, 0, 0)
	encoding := &StatuslineEncoding{
		label: encodingLabel,
	}
	s.encoding = encoding

	posLabel := widgets.NewQLabel(nil, 0)
	posLabel.SetContentsMargins(0, 0, 0, 0)
	pos := &StatuslinePos{
		label: posLabel,
	}
	s.pos = pos

	filetypeLabel := widgets.NewQLabel(nil, 0)
	filetypeLabel.SetContentsMargins(0, 0, 0, 0)
	filetype := &StatuslineFiletype{
		label: filetypeLabel,
	}
	s.filetype = filetype

	okIcon := svg.NewQSvgWidget(nil)
	okIcon.SetFixedSize2(14, 14)
	okLabel := widgets.NewQLabel(nil, 0)
	okLabel.SetContentsMargins(0, 0, 0, 0)
	errorIcon := svg.NewQSvgWidget(nil)
	errorIcon.SetFixedSize2(14, 14)
	errorIcon.Hide()
	errorLabel := widgets.NewQLabel(nil, 0)
	errorLabel.SetContentsMargins(0, 0, 0, 0)
	errorLabel.Hide()
	warnIcon := svg.NewQSvgWidget(nil)
	warnIcon.SetFixedSize2(14, 14)
	warnIcon.Hide()
	warnLabel := widgets.NewQLabel(nil, 0)
	warnLabel.SetContentsMargins(0, 0, 0, 0)
	warnLabel.Hide()
	lintLayout := widgets.NewQHBoxLayout()
	lintLayout.SetContentsMargins(0, 0, 0, 0)
	lintLayout.SetSpacing(2)
	lintLayout.AddWidget(okIcon, 0, 0)
	lintLayout.AddWidget(okLabel, 0, 0)
	lintLayout.AddWidget(errorIcon, 0, 0)
	lintLayout.AddWidget(errorLabel, 0, 0)
	lintLayout.AddWidget(warnIcon, 0, 0)
	lintLayout.AddWidget(warnLabel, 0, 0)
	lintWidget := widgets.NewQWidget(nil, 0)
	lintWidget.SetContentsMargins(0, 0, 0, 0)
	lintWidget.SetLayout(lintLayout)
	lint := &StatuslineLint{
		s:          s,
		widget:     lintWidget,
		okIcon:     okIcon,
		errorIcon:  errorIcon,
		warnIcon:   warnIcon,
		okLabel:    okLabel,
		errorLabel: errorLabel,
		warnLabel:  warnLabel,
		errors:     -1,
		warnings:   -1,
	}
	s.lint = lint

	layout.AddWidget(modeWidget)
	layout.AddWidget(gitWidget)
	layout.AddWidget(fileWidget)
	layout.AddWidget(filetypeLabel)
	layout.AddWidget(encodingLabel)
	layout.AddWidget(posLabel)
	layout.AddWidget(lintWidget)

	return s
}

func (s *Statusline) subscribe() {
	if !s.ws.drawStatusline {
		s.widget.Hide()
		return
	}
	s.ws.signal.ConnectStatuslineSignal(func() {
		updates := <-s.updates
		s.handleUpdates(updates)
	})
	s.ws.signal.ConnectLintSignal(func() {
		s.lint.update()
	})
	s.ws.signal.ConnectGitSignal(func() {
		s.git.update()
	})
	s.ws.nvim.RegisterHandler("statusline", func(updates ...interface{}) {
		s.updates <- updates
		s.ws.signal.StatuslineSignal()
	})
	s.ws.nvim.Subscribe("statusline")
	s.ws.nvim.Command(`autocmd BufEnter * call rpcnotify(0, "statusline", "bufenter", expand("%:p"), &filetype, &fileencoding)`)
	s.ws.nvim.Command(`autocmd CursorMoved,CursorMovedI * call rpcnotify(0, "statusline", "cursormoved", getpos("."))`)
}

func (s *Statusline) handleUpdates(updates []interface{}) {
	event := updates[0].(string)
	switch event {
	case "bufenter":
		file := updates[1].(string)
		filetype := updates[2].(string)
		encoding := updates[3].(string)
		s.file.redraw(file)
		s.filetype.redraw(filetype)
		s.encoding.redraw(encoding)
		go s.git.redraw(file)
	case "cursormoved":
		pos := updates[1].([]interface{})
		ln := reflectToInt(pos[1])
		col := reflectToInt(pos[2]) + reflectToInt(pos[3])
		s.pos.redraw(ln, col)
	default:
		fmt.Println("unhandled statusline event", event)
	}
}

func (s *StatusMode) update() {
	s.label.SetText(s.text)
	s.label.SetStyleSheet(fmt.Sprintf("background-color: %s;", s.bg.String()))
}

func (s *StatusMode) redraw() {
	if s.s.ws.mode == s.mode {
		return
	}
	s.mode = s.s.ws.mode
	text := s.mode
	bg := newRGBA(102, 153, 204, 1)
	switch s.mode {
	case GonvimNormal:
		text = GonvimNormal
		bg = newRGBA(102, 153, 204, 1)
	case "cmdline_normal":
		text = GonvimNormal
		bg = newRGBA(102, 153, 204, 1)
	case GonvimInsert:
		text = GonvimInsert
		bg = newRGBA(153, 199, 148, 1)
	case "visual":
		text = "visual"
		bg = newRGBA(250, 200, 99, 1)
	}
	s.text = text
	s.bg = bg
	s.update()
}

func (s *StatuslineGit) hide() {
	if s.hidden {
		return
	}
	s.hidden = true
	s.s.ws.signal.GitSignal()
}

func (s *StatuslineGit) update() {
	if s.hidden {
		s.widget.Hide()
		return
	}
	s.label.SetText(s.branch)
	if !s.svgLoaded {
		s.svgLoaded = true
		svgContent := s.s.ws.getSvg("git", newRGBA(212, 215, 214, 1))
		s.icon.Load2(core.NewQByteArray2(svgContent, len(svgContent)))
	}
	s.widget.Show()
}

func (s *StatuslineGit) redraw(file string) {
	if file == "" || strings.HasPrefix(file, "term://") {
		s.file = file
		s.hide()
		s.branch = ""
		return
	}

	if s.file == file {
		return
	}

	s.file = file
	dir := filepath.Dir(file)
	out, err := exec.Command("git", "-C", dir, "branch").Output()
	if err != nil {
		s.hide()
		s.branch = ""
		return
	}

	branch := ""
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "* ") {
			if strings.HasPrefix(line, "* (HEAD detached at ") {
				branch = line[20 : len(line)-1]
			} else {
				branch = line[2:]
			}
		}
	}
	_, err = exec.Command("git", "-C", dir, "diff", "--quiet").Output()
	if err != nil {
		branch += "*"
	}

	if s.branch != branch {
		s.branch = branch
		s.hidden = false
		s.s.ws.signal.GitSignal()
	}
}

func (s *StatuslineFile) updateIcon() {
	svgContent := s.s.ws.getSvg(s.fileType, nil)
	s.icon.Load2(core.NewQByteArray2(svgContent, len(svgContent)))
}

func (s *StatuslineFile) redraw(file string) {
	if file == "" {
		file = "[No Name]"
	}

	if file == s.file {
		return
	}

	s.file = file

	base := filepath.Base(file)
	dir := filepath.Dir(file)
	if dir == "." {
		dir = ""
	}
	if strings.HasPrefix(file, "term://") {
		base = file
		dir = ""
	}
	fileType := getFileType(file)
	if s.fileType != fileType {
		s.fileType = fileType
		s.updateIcon()
	}
	if s.base != base {
		s.base = base
		s.fileLabel.SetText(s.base)
	}
	if s.dir != dir {
		s.dir = dir
		s.folderLabel.SetText(s.dir)
	}
}

func (s *StatuslinePos) redraw(ln, col int) {
	if ln == s.ln && col == s.col {
		return
	}
	text := fmt.Sprintf("Ln %d, Col %d", ln, col)
	if text != s.text {
		s.text = text
		s.label.SetText(text)
	}
}

func (s *StatuslineEncoding) redraw(encoding string) {
	if s.encoding == encoding {
		return
	}
	s.encoding = encoding
	s.label.SetText(s.encoding)
}

func (s *StatuslineFiletype) redraw(filetype string) {
	if filetype == s.filetype {
		return
	}
	s.filetype = filetype
	s.label.SetText(s.filetype)
}

func (s *StatuslineLint) update() {
	if !s.svgLoaded {
		s.svgLoaded = true
		svgContent := s.s.ws.getSvg("check", newRGBA(141, 193, 73, 1))
		s.okIcon.Load2(core.NewQByteArray2(svgContent, len(svgContent)))
		svgContent = s.s.ws.getSvg("cross", newRGBA(204, 62, 68, 1))
		s.errorIcon.Load2(core.NewQByteArray2(svgContent, len(svgContent)))
		svgContent = s.s.ws.getSvg("exclamation", nil)
		s.warnIcon.Load2(core.NewQByteArray2(svgContent, len(svgContent)))
	}

	if s.errors == 0 && s.warnings == 0 {
		s.okIcon.Show()
		s.okLabel.SetText("ok")
		s.okLabel.Show()
		s.errorIcon.Hide()
		s.errorLabel.Hide()
		s.warnIcon.Hide()
		s.warnLabel.Hide()
	} else {
		s.okIcon.Hide()
		s.okLabel.Hide()
		s.errorLabel.SetText(strconv.Itoa(s.errors))
		s.warnLabel.SetText(strconv.Itoa(s.warnings))
		s.errorIcon.Show()
		s.errorLabel.Show()
		s.warnIcon.Show()
		s.warnLabel.Show()
	}
}

func (s *StatuslineLint) redraw(errors, warnings int) {
	if errors == s.errors && warnings == s.warnings {
		return
	}
	s.errors = errors
	s.warnings = warnings
	s.s.ws.signal.LintSignal()
}
