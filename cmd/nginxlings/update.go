package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

/* ---------- check.sh  ---------- */

func runCheck(ctx context.Context, index int, ex Exercise, done chan<- struct{}) tea.Cmd {
	return func() tea.Msg {
		defer close(done)
		out, err := runExerciseCheckContext(ctx, ex)

		return checkResultMsg{
			output: out,
			err:    err,
			index:  index,
		}
	}
}

func cancelCheckAndQuit(cancel context.CancelFunc, done <-chan struct{}) tea.Cmd {
	return func() tea.Msg {
		cancel()
		if done != nil {
			<-done
		}
		return tea.Quit()
	}
}

const (
	marqueeTickInterval  = 350 * time.Millisecond
	marqueePauseInterval = 1050 * time.Millisecond
)

func marqueeTick(generation uint64, delay time.Duration) tea.Cmd {
	return tea.Tick(delay, func(time.Time) tea.Msg {
		return marqueeTickMsg{generation: generation}
	})
}

func marqueeTickDelay(offset, cycleLength int) time.Duration {
	if offset == 0 || cycleLength > 0 && offset%cycleLength == 0 {
		return marqueePauseInterval
	}
	return marqueeTickInterval
}

func (m model) selectedExerciseName() string {
	if len(m.exercises) == 0 || m.cursor < 0 || m.cursor >= len(m.exercises) {
		return ""
	}

	ex := m.exercises[m.cursor]
	if m.i18n != nil {
		return m.i18n.ExerciseName(m.currentLangCode(), ex.ID, ex.DefaultName)
	}
	return ex.DefaultName
}

func (m model) selectedNameOverflows() bool {
	if len(m.exercises) == 0 || m.cursor < 0 || m.cursor >= len(m.exercises) {
		return false
	}
	layout := m.layout()
	nameWidth := layout.listNameWidth()
	if layout.compact {
		nameWidth = maxInt(1, m.width-4)
	}
	return lipgloss.Width(m.selectedExerciseName()) > nameWidth
}

func (m model) selectedMarqueeCycleLength() int {
	name := m.selectedExerciseName()
	if name == "" {
		return 0
	}
	return len([]rune(name + "   "))
}

func (m *model) resetMarquee() tea.Cmd {
	m.marquee = 0
	m.marqueeGeneration++
	if !m.selectedNameOverflows() {
		return nil
	}
	return marqueeTick(m.marqueeGeneration, marqueeTickDelay(0, m.selectedMarqueeCycleLength()))
}

func (m *model) persistSelectedExercise() {
	if m.state == nil || m.cursor < 0 || m.cursor >= len(m.exercises) {
		return
	}
	id := m.exercises[m.cursor].ID
	if m.state.LastExerciseID == id {
		return
	}
	m.state.LastExerciseID = id
	if err := m.state.Save(m.paths.StateFile); err != nil {
		m.status = m.uif("status_state_save_failed", "Progress could not be saved: %v", err)
	}
}

// quit stops the program, cancelling a running check first so its process
// group does not outlive the TUI.
func (m *model) quit() tea.Cmd {
	if m.loading && m.checkCancel != nil {
		cancel := m.checkCancel
		done := m.checkDone
		m.checkCancel = nil
		m.checkDone = nil
		return cancelCheckAndQuit(cancel, done)
	}
	return tea.Quit
}

// resetProgress clears completion, returns to the first exercise, and shows
// its instructions. Reset is refused while a check runs, so no late result
// can mark an exercise complete again.
func (m *model) resetProgress() tea.Cmd {
	for i := range m.exercises {
		m.exercises[i].Completed = false
	}
	m.cursor = 0
	m.checkOutputs = nil
	m.loadReadmeForCurrent()

	m.status = m.ui("status_reset_done", "Progress reset.")
	if m.state != nil {
		m.state.Completed = make(map[string]bool)
		m.state.LastExerciseID = ""
		if err := m.state.Save(m.paths.StateFile); err != nil {
			m.status = m.uif("status_state_save_failed", "Progress could not be saved: %v", err)
		}
	}
	return m.resetMarquee()
}

// resetExerciseFiles restores the selected exercise's starting files. The
// exercise is no longer completed, since its check fails again.
func (m *model) resetExerciseFiles() {
	ex := m.exercises[m.cursor]
	if err := ResetExercise(filepath.Join(m.paths.AssetRoot, "exercises"), m.paths.Workspace, ex.ID); err != nil {
		m.status = m.uif("status_reset_files_failed", "Starting files could not be restored: %v", err)
		return
	}
	m.exercises[m.cursor].Completed = false
	delete(m.checkOutputs, ex.ID)
	m.loadReadmeForCurrent()
	m.status = m.ui("status_reset_files_done", "Starting files restored.")
	if m.state != nil && m.state.Completed[ex.ID] {
		delete(m.state.Completed, ex.ID)
		if err := m.state.Save(m.paths.StateFile); err != nil {
			m.status = m.uif("status_state_save_failed", "Progress could not be saved: %v", err)
		}
	}
}

// openFolder shows dir in the desktop file manager without waiting for it.
// Tests replace it.
var openFolder = func(dir string) error {
	if os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
		return errors.New("no graphical session")
	}
	opener, err := exec.LookPath("xdg-open")
	if err != nil {
		return err
	}
	cmd := exec.Command(opener, dir)
	// Its own process group keeps the file manager open after nginxlings quits.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	go cmd.Wait()
	return nil
}

// openExerciseFolder opens the selected exercise's folder and returns the
// status that says where the files are.
func (m model) openExerciseFolder() string {
	if len(m.exercises) == 0 || m.cursor < 0 || m.cursor >= len(m.exercises) {
		return m.ui("status_no_exercises", "No exercises found.")
	}
	dir := m.exercises[m.cursor].Path
	if err := openFolder(dir); err != nil {
		return m.uif("status_folder_open_failed", "No file manager could be opened. The files are in %s", displayPath(dir))
	}
	return m.uif("status_folder_opened", "Opened %s in the file manager.", displayPath(dir))
}

// displayPath shortens a path under the home directory to ~/...
func displayPath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return path
	}
	if path == home {
		return "~"
	}
	if rel, ok := strings.CutPrefix(path, home+string(filepath.Separator)); ok {
		return "~/" + filepath.ToSlash(rel)
	}
	return path
}

type terminalMovedMsg struct {
	dir  string
	busy string
}

// selectedDir is the folder of the selected exercise, or the exercise root.
func (m model) selectedDir() string {
	if m.cursor >= 0 && m.cursor < len(m.exercises) {
		return m.exercises[m.cursor].Path
	}
	return m.paths.ExercisesDir()
}

// openTerminal opens the shell pane below the app in the selected folder.
func (m model) openTerminal(focus bool) tea.Cmd {
	title := m.ui("terminal_pane_title", "Terminal · F2: switch · Alt+↑/↓: resize")
	appTitle := m.ui("app_pane_title", "nginxlings · F2: terminal")
	open := openTerminalPane(m.selectedDir(), m.height, focus, title)
	return func() tea.Msg {
		if pane := os.Getenv("TMUX_PANE"); pane != "" {
			_, _ = runTmux("select-pane", "-t", pane, "-T", appTitle)
		}
		return open()
	}
}

// terminalToSelected takes the terminal to the selected exercise's folder:
// the shell pane below in tmux, opened again if it was closed, or a full
// screen shell without tmux.
func (m model) terminalToSelected() tea.Cmd {
	dir := m.selectedDir()
	if !m.inTmux {
		return runShell(dir, m.ui("shell_hint", "nginxlings: type exit to return."))
	}
	pane := m.terminalPane
	open := m.openTerminal(true)
	return func() tea.Msg {
		if pane != "" {
			if busy, err := moveTerminalTo(pane, dir); err == nil {
				return terminalMovedMsg{dir: dir, busy: busy}
			}
		}
		return open()
	}
}

// retitleTerminal translates the pane titles after a language switch.
func (m model) retitleTerminal() tea.Cmd {
	if m.terminalPane == "" {
		return nil
	}
	pane := m.terminalPane
	title := m.ui("terminal_pane_title", "Terminal · F2: switch · Alt+↑/↓: resize")
	appTitle := m.ui("app_pane_title", "nginxlings · F2: terminal")
	return func() tea.Msg {
		_, _ = runTmux("select-pane", "-t", pane, "-T", title)
		if own := os.Getenv("TMUX_PANE"); own != "" {
			_, _ = runTmux("select-pane", "-t", own, "-T", appTitle)
		}
		return nil
	}
}

/* ---------- Bubble Tea Update ---------- */

func (m model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.setupCheckCmd())
}

type setupCheckMsg struct {
	report setupReport
}

type setupFixDoneMsg struct {
	err error
}

// newSetupEnv is what the TUI's setup check reads; tests replace it.
var newSetupEnv = systemSetupEnv

// runFix hands the terminal to a fix command, so sudo can ask for the
// password, and reports when it ends. Tests replace it.
var runFix = func(argv []string) tea.Cmd {
	return tea.ExecProcess(exec.Command(argv[0], argv[1:]...), func(err error) tea.Msg {
		return setupFixDoneMsg{err: err}
	})
}

// setupCheckCmd checks in the background what the exercises need.
func (m model) setupCheckCmd() tea.Cmd {
	exercises := append([]Exercise(nil), m.exercises...)
	return func() tea.Msg {
		return setupCheckMsg{report: runSetupCheck(context.Background(), exercises, newSetupEnv())}
	}
}

// showKeys switches the right panel to the list of every key.
func (m *model) showKeys() {
	m.panel = panelKeys
	m.readme.SetContent(m.renderMarkdown(m.ui("keys_markdown", "- **?**: show or hide this list\n- **q**: quit")))
	m.readme.GotoTop()
}

// showSetup switches the right panel to the last setup check.
func (m *model) showSetup() {
	m.panel = panelSetup
	content := m.ui("status_setup_checking", "Checking the setup...")
	if m.setup != nil {
		content = m.renderMarkdown(setupMarkdown(setupReportLines(*m.setup, m.ui), m.ui))
	}
	m.readme.SetContent(content)
	m.readme.GotoTop()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	next, cmd := m.update(msg)
	updated := next.(model)
	updated.fitReadme()
	return updated, cmd
}

// followsEnd reports whether the right panel shows the end of its text in a
// way worth keeping when the panel changes size: check output opened at its
// end, or any text the reader scrolled to the bottom.
func (m model) followsEnd() bool {
	return m.readme.AtBottom() && (m.readme.YOffset > 0 || m.panel == panelCheckOutput)
}

// fitReadme gives the right panel the height the status and key help leave
// for it; a long status message takes rows from the instructions.
func (m *model) fitReadme() {
	if !m.ready {
		return
	}
	height := m.layout().readmeHeight
	if m.readme.Height == height {
		return
	}
	atEnd := m.followsEnd()
	m.readme.Height = height
	if atEnd {
		m.readme.GotoBottom()
	} else {
		m.readme.SetYOffset(m.readme.YOffset)
	}
}

func (m model) update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		panel := m.panel
		// Re-rendering at the new width changes the line count, so remember
		// where the reader is and return there afterwards.
		atEnd := m.followsEnd()
		offset, lines := m.readme.YOffset, m.readme.TotalLineCount()
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

		layout := calculateLayout(m.width, m.height)
		m.listWidth = layout.listOuterWidth
		m.rightWidth = layout.rightOuterWidth
		m.readme.Width = layout.rightInnerWidth
		m.readme.Height = m.layout().readmeHeight

		m.loadReadmeForCurrent()
		switch {
		case panel == panelCheckOutput && m.selectedCheckOutput() != "":
			m.showCheckOutput()
		case panel == panelHint:
			m.showHint()
		case panel == panelSetup:
			m.showSetup()
		case panel == panelKeys:
			m.showKeys()
		}
		switch {
		case atEnd:
			m.readme.GotoBottom()
		case lines > 0:
			m.readme.SetYOffset(offset * m.readme.TotalLineCount() / lines)
		}
		// In its own tmux session the app opens the terminal below itself
		// once, when the window is tall enough to share.
		if m.ownTmux && !m.terminalTried && m.height >= minSplitHeight {
			m.terminalTried = true
			return m, tea.Batch(m.resetMarquee(), m.openTerminal(false))
		}
		return m, m.resetMarquee()

	case tea.KeyMsg:
		// Runes that arrive in one read come as one message, for example
		// "sss" when a key is held while the view redraws. Handle them as
		// separate presses; a paste stays one message.
		if msg.Type == tea.KeyRunes && len(msg.Runes) > 1 && !msg.Paste {
			return m.updateEachRune(msg)
		}
		key := msg.String()

		// ctrl+c her durumda çıkar, reset onayı beklenirken bile
		if key == "ctrl+c" {
			return m, m.quit()
		}

		// Reset onayı açıksa sadece y/n/esc handle et
		if m.confirmReset || m.confirmResetFiles {
			switch key {
			case "y", "Y":
				files := m.confirmResetFiles
				m.confirmReset, m.confirmResetFiles = false, false
				if files {
					m.resetExerciseFiles()
					return m, nil
				}
				return m, m.resetProgress()

			case "n", "N", "esc":
				m.status = m.ui("status_reset_canceled", "Reset canceled.")
				m.confirmReset, m.confirmResetFiles = false, false
				return m, nil
			}
			// Farklı tuşsa görmezden gel
			return m, nil
		}

		switch key {

		// Global çıkış
		case "q":
			return m, m.quit()

		// SOL PANEL (egzersiz listesi) – W/S ile hareket
		case "w":
			if m.cursor > 0 {
				m.cursor--
				m.loadReadmeForCurrent()
				m.persistSelectedExercise()
				return m, m.resetMarquee()
			}
		case "s":
			if m.cursor < len(m.exercises)-1 {
				m.cursor++
				m.loadReadmeForCurrent()
				m.persistSelectedExercise()
				return m, m.resetMarquee()
			}

		// SAĞ PANEL (instructions) – ok tuşları ve PgUp/PgDn ile scroll
		case "up":
			m.readme.LineUp(1)
		case "down":
			m.readme.LineDown(1)

		case "pgup":
			m.readme.LineUp(5)
		case "pgdown":
			m.readme.LineDown(5)

		// YATAY scroll: sağ/sol ok tuşları
		case "right":
			m.readme.ScrollRight(4)
		case "left":
			m.readme.ScrollLeft(4)

		case "tab":
			if m.panel != panelInstructions {
				m.panel = panelInstructions
				m.readme.SetContent(m.instructionsContent)
				m.readme.GotoTop()
				m.status = m.ui("status_showing_instructions", "Showing exercise instructions.")
			} else if m.selectedCheckOutput() != "" {
				m.showCheckOutput()
				m.status = m.ui("status_showing_check_output", "Showing last check output.")
			} else {
				m.status = m.ui("status_no_check_output", "No check output yet.")
			}

		// CHECK çalıştır
		case "c", "enter":
			if m.loading {
				m.status = m.ui("status_check_already_running", "A check is already running; waiting for its result.")
				return m, nil
			}
			if len(m.exercises) == 0 {
				m.status = m.ui("status_no_exercises", "No exercises found.")
				return m, nil
			}

			idx := m.cursor
			ex := m.exercises[idx]
			m.loading = true
			ctx, cancel := context.WithTimeout(context.Background(), defaultCheckTimeout)
			done := make(chan struct{})
			m.checkCancel = cancel
			m.checkDone = done

			name := ex.DefaultName
			if m.i18n != nil {
				name = m.i18n.ExerciseName(m.currentLangCode(), ex.ID, ex.DefaultName)
			}

			m.status = fmt.Sprintf("\"%s\"...", name)
			return m, runCheck(ctx, idx, ex, done)

		// Hint: sağ panelde gösterilir, çünkü durum satırına sığmaz
		case "h":
			if m.panel == panelHint {
				m.panel = panelInstructions
				m.readme.SetContent(m.instructionsContent)
				m.readme.GotoTop()
				m.status = m.ui("status_showing_instructions", "Showing exercise instructions.")
			} else {
				m.showHint()
				m.status = m.ui("status_showing_hint", "Showing the hint. Press h or Tab to return to the instructions.")
			}

		// Dil değiştir
		case "l", "L":
			if m.i18n != nil && len(m.i18n.Langs()) > 0 {
				m.langIndex = m.i18n.NextLangIndex(m.langIndex)
				m.status = m.ui("status_lang_switched", "Language switched.")
				m.loadReadmeForCurrent()

				// Dil tercihini state'e yaz
				if m.state != nil {
					m.state.LangCode = m.currentLangCode()
					if err := m.state.Save(m.paths.StateFile); err != nil {
						m.status = m.uif("status_state_save_failed", "Progress could not be saved: %v", err)
					}
				}
				return m, tea.Batch(m.resetMarquee(), m.retitleTerminal())
			}

		// Seçili egzersizin başlangıç dosyalarını geri yükle
		case "e", "E":
			switch {
			case m.loading:
				m.status = m.ui("status_reset_files_blocked", "Wait for the running check to finish before restoring files.")
			case m.paths.Workspace == "":
				m.status = m.ui("status_reset_files_unavailable", "These exercises are edited where they ship, so there are no starting files to restore.")
			case len(m.exercises) == 0:
				m.status = m.ui("status_no_exercises", "No exercises found.")
			default:
				m.confirmResetFiles = true
				m.status = m.uif("status_reset_files_confirm", "Restore the starting files of %s? Your changes to this exercise will be lost. (y/n)", m.selectedExerciseName())
			}

		// Tüm tuşların listesi
		case "?":
			if m.panel == panelKeys {
				m.panel = panelInstructions
				m.readme.SetContent(m.instructionsContent)
				m.readme.GotoTop()
				m.status = m.ui("status_showing_instructions", "Showing exercise instructions.")
				break
			}
			m.showKeys()
			m.status = m.ui("status_showing_keys", "Showing all keys. Press ? or Tab to return.")

		// Kurulum kontrolü: aç/kapat; her açılışta yeniden kontrol et
		case "d", "D":
			if m.panel == panelSetup {
				m.panel = panelInstructions
				m.readme.SetContent(m.instructionsContent)
				m.readme.GotoTop()
				m.status = m.ui("status_showing_instructions", "Showing exercise instructions.")
				return m, nil
			}
			m.showSetup()
			m.status = m.ui("status_showing_setup", "Showing the setup check. Press d or Tab to return.")
			return m, m.setupCheckCmd()

		// Kurulum panelindeki ilk düzeltmeyi çalıştır
		case "f", "F":
			if m.panel != panelSetup {
				break
			}
			var fix []string
			if m.setup != nil {
				fix, _ = m.setup.firstFix()
			}
			if fix == nil {
				m.status = m.ui("status_setup_nothing_to_fix", "There is no command to run.")
				break
			}
			m.status = m.uif("status_setup_running_fix", "Running %s", shellCommand(fix))
			return m, runFix(fix)

		// Terminal: alttaki kabuğu seçili egzersizin klasörüne götür
		case "t", "T":
			return m, m.terminalToSelected()

		// Egzersiz klasörünü dosya yöneticisinde aç
		case "o", "O":
			m.status = m.openExerciseFolder()

		// Progress reset
		case "r", "R":
			if m.loading {
				m.status = m.ui("status_reset_blocked", "Wait for the running check to finish before resetting progress.")
				return m, nil
			}
			m.confirmReset = true
			m.status = m.ui("status_reset_confirm", "Reset all progress? (y/n)")
		}

		// KeyMsg case’inin sonunda mutlaka return et
		return m, nil

	case checkResultMsg:
		m.loading = false
		if m.checkCancel != nil {
			m.checkCancel()
		}
		m.checkCancel = nil
		m.checkDone = nil

		// Kaçış dizilerini ve kontrol karakterlerini temizle
		out := sanitizeCheckOutput(msg.output)
		if msg.err != nil && out == "" {
			out = "[FAIL] " + msg.err.Error()
		}

		// Egzersiz state’ini güncelle
		if msg.index >= 0 && msg.index < len(m.exercises) {
			ex := m.exercises[msg.index]

			success := (msg.err == nil)
			m.exercises[msg.index].Completed = success

			if m.state != nil {
				if m.state.Completed == nil {
					m.state.Completed = make(map[string]bool)
				}
				m.state.Completed[ex.ID] = success
				if err := m.state.Save(m.paths.StateFile); err != nil {
					if out != "" {
						out = out + fmt.Sprintf("\n[state save error: %v]", err)
					} else {
						out = fmt.Sprintf("[state save error: %v]", err)
					}
				}
			}
		}

		// Status satırı için kısa özet
		summary := lastLine(out)
		if summary == "" && msg.err != nil {
			summary = "[FAIL] check.sh failed"
		} else if summary == "" {
			summary = "[PASS] check.sh OK"
		}
		m.status = summary

		// Çıktıyı egzersize bağla; panel sadece sonuç hâlâ seçili egzersize
		// aitse değişir, yoksa kullanıcının baktığı içerik yerinde kalır.
		if msg.index >= 0 && msg.index < len(m.exercises) {
			id := m.exercises[msg.index].ID
			if out != "" {
				if m.checkOutputs == nil {
					m.checkOutputs = make(map[string]string)
				}
				m.checkOutputs[id] = out
			}
			if msg.index == m.cursor {
				if out != "" {
					m.showCheckOutput()
				}
			} else {
				m.status = id + ": " + summary
			}
		}

		return m, nil

	case setupCheckMsg:
		report := msg.report
		m.setup = &report
		problems := len(report.problems)
		switch {
		case m.panel == panelSetup:
			m.showSetup()
			if problems == 0 {
				m.status = m.ui("status_setup_ok", "Everything the exercises need is ready.")
			} else {
				m.status = m.uif("status_setup_remaining", "Setup problems left: %d.", problems)
			}
		case problems > 0 && m.status == m.initialStatusText():
			m.status = m.uif("status_setup_problems", "Setup needs attention (%d). Press d to see what to do.", problems)
		}
		return m, nil

	case terminalPaneMsg:
		if msg.err != nil {
			m.status = m.uif("status_terminal_failed", "The terminal could not be opened: %v", msg.err)
			return m, nil
		}
		m.terminalPane = msg.pane
		return m, nil

	case terminalMovedMsg:
		if msg.busy != "" {
			m.status = m.uif("status_terminal_busy", "The terminal is running %s. Type cd yourself when it ends.", msg.busy)
		} else {
			m.status = m.uif("status_terminal_moved", "Terminal moved to %s. F2 comes back here.", displayPath(msg.dir))
		}
		return m, nil

	case shellDoneMsg:
		m.status = m.ui("status_shell_done", "Back from the shell.")
		return m, nil

	case setupFixDoneMsg:
		if msg.err != nil {
			m.status = m.uif("status_setup_fix_failed", "The command did not finish: %v", msg.err)
		} else {
			m.status = m.ui("status_setup_fix_done", "Done. Checking the setup again...")
		}
		return m, m.setupCheckCmd()

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case marqueeTickMsg:
		if msg.generation != m.marqueeGeneration {
			return m, nil
		}
		if !m.selectedNameOverflows() {
			m.marquee = 0
			m.marqueeGeneration++
			return m, nil
		}
		m.marquee++
		return m, marqueeTick(
			m.marqueeGeneration,
			marqueeTickDelay(m.marquee, m.selectedMarqueeCycleLength()),
		)
	}

	return m, nil
}

// updateEachRune applies a multi-rune key message one rune at a time and
// batches the commands the single presses return.
func (m model) updateEachRune(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var current tea.Model = m
	cmds := make([]tea.Cmd, 0, len(msg.Runes))
	for _, r := range msg.Runes {
		var cmd tea.Cmd
		current, cmd = current.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}, Alt: msg.Alt})
		cmds = append(cmds, cmd)
	}
	return current, tea.Batch(cmds...)
}
