package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/hiroki1117/scrapbox-cli/internal/scrapbox"
)

const writeHelp = `Writes are previewed and applied in one step. Use --dry-run to only show the result.
Line numbers are the ones shown by 'sb page get -n' (line 1 is the title).`

const textInputHelp = `Text is taken from --text, --file (- for stdin), or stdin. Each line becomes a page line.`

// textInput is the --text / --file input of write commands.
type textInput struct {
	text string
	file string
}

func (in *textInput) addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&in.text, "text", "t", "", "text to write (may contain newlines)")
	cmd.Flags().StringVarP(&in.file, "file", "f", "", "read text from a file (- for stdin)")
}

// lines returns the input lines. Without --text or --file it reads stdin, unless stdin is a terminal.
// A single trailing newline is ignored.
func (in *textInput) lines(app *App, required bool) ([]string, error) {
	var s string
	switch {
	case in.text != "":
		s = in.text
	case in.file != "" && in.file != "-":
		b, err := os.ReadFile(in.file)
		if err != nil {
			return nil, err
		}
		s = string(b)
	default:
		if f, ok := app.IO.In.(*os.File); ok && in.file == "" && term.IsTerminal(int(f.Fd())) {
			break
		}
		b, err := io.ReadAll(app.IO.In)
		if err != nil {
			return nil, fmt.Errorf("read stdin: %w", err)
		}
		s = string(b)
	}
	s = strings.TrimSuffix(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
	if s == "" {
		if required {
			return nil, errors.New("no text: use --text, --file, or pipe text to stdin")
		}
		return nil, nil
	}
	return strings.Split(s, "\n"), nil
}

func loadEditTarget(cmd *cobra.Command, app *App, arg string) (*pageTarget, error) {
	t, err := fetchPage(cmd, app, arg)
	if errors.Is(err, scrapbox.ErrNotFound) {
		return nil, fmt.Errorf("%w (use 'sb page create')", err)
	}
	return t, err
}

// line returns the 0-based index of line number n (1-based).
func (t *pageTarget) line(n int) (int, error) {
	if n < 1 || n > len(t.page.Lines) {
		return 0, fmt.Errorf("line %d is out of range (the page has %d lines)", n, len(t.page.Lines))
	}
	return n - 1, nil
}

// anchor returns the insert anchor for the position before line index i.
func (t *pageTarget) anchor(i int) string {
	if i >= len(t.page.Lines) {
		return scrapbox.End
	}
	return t.page.Lines[i].ID
}

func (t *pageTarget) apply(cmd *cobra.Command, app *App, dryRun bool, changes []scrapbox.Change) error {
	return applyEdit(cmd, app, t.client, t.project, t.page.ID, t.page.Lines, dryRun, changes)
}

func insertLines(anchor string, lines []string) []scrapbox.Change {
	changes := make([]scrapbox.Change, len(lines))
	for i, l := range lines {
		changes[i] = scrapbox.InsertBefore(anchor, l)
	}
	return changes
}

// applyEdit previews changes and, unless dryRun, submits them. pageID is empty for a new page.
func applyEdit(cmd *cobra.Command, app *App, client *scrapbox.Client, project, pageID string, old []scrapbox.Line, dryRun bool, changes []scrapbox.Change) error {
	preview, err := client.Pages.PreviewEdit(cmd.Context(), project, pageID, changes)
	if err != nil {
		return err
	}
	if dryRun {
		return app.Printer().Print(preview, func(w io.Writer) error {
			printDiff(w, old, preview, changes)
			return nil
		})
	}
	res, err := client.Pages.SubmitEdit(cmd.Context(), project, preview.PreviewID)
	if err != nil {
		return err
	}
	out := struct {
		*scrapbox.EditResult
		URL string `json:"url,omitempty"`
	}{EditResult: res}
	if res.Page != nil {
		out.URL = client.PageURL(project, res.Page.Title)
	}
	return app.Printer().Print(out, func(w io.Writer) error {
		switch {
		case res.PageDeleted != nil:
			fmt.Fprintf(w, "Deleted %s\n", res.PageDeleted.Title)
		case res.TitleChanged != nil:
			fmt.Fprintf(w, "Renamed %q to %q: %s\n", res.TitleChanged.From, res.TitleChanged.To, out.URL)
		case pageID == "":
			fmt.Fprintf(w, "Created %s\n", out.URL)
		default:
			fmt.Fprintf(w, "Updated %s\n", out.URL)
		}
		return nil
	})
}

// printDiff prints the page after the edit, marking added (+), replaced (~) and deleted (-) lines.
func printDiff(w io.Writer, old []scrapbox.Line, preview *scrapbox.EditPreview, changes []scrapbox.Change) {
	if preview.PageDelete || preview.Page == nil {
		fmt.Fprintln(w, "The page will be deleted:")
		for _, l := range old {
			fmt.Fprintf(w, "- %s\n", l.Text)
		}
		return
	}
	updated := map[string]bool{}
	for _, c := range changes {
		if c.Update != "" {
			updated[c.Update] = true
		}
	}
	existing := map[string]bool{}
	for _, l := range old {
		existing[l.ID] = true
	}
	// Kept lines stay in their original order, so the old lines skipped before a kept line were deleted.
	i := 0
	for _, l := range preview.Page.Lines {
		mark := "+"
		if existing[l.ID] {
			for ; i < len(old) && old[i].ID != l.ID; i++ {
				fmt.Fprintf(w, "- %s\n", old[i].Text)
			}
			i++
			mark = " "
			if updated[l.ID] {
				mark = "~"
			}
		}
		fmt.Fprintf(w, "%s %s\n", mark, l.Text)
	}
	for ; i < len(old); i++ {
		fmt.Fprintf(w, "- %s\n", old[i].Text)
	}
}

func newPageCreateCmd(app *App) *cobra.Command {
	var (
		in     textInput
		dryRun bool
	)
	cmd := &cobra.Command{
		Use:   "create <title>",
		Short: "Create a page",
		Long: `Create a page with the given title. The body is optional.
Fails if the page already exists; use 'sb page append' to add to it.


` + textInputHelp + `

` + writeHelp + `

Text output: "Created <url>" (with --dry-run, the page to be created).
JSON output: {commitId, page: {title}, url} (with --dry-run, {previewId, pagePreview, ...})`,
		Example: `  sb page create "Meeting 2026-10-11" --text "Agenda"
  printf 'line 1\nline 2\n' | sb page create "New page"
  sb page create "New page" --file body.txt --dry-run`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			project, client, err := app.projectClient()
			if err != nil {
				return err
			}
			title := args[0]
			body, err := in.lines(app, false)
			if err != nil {
				return err
			}
			// The server adds a suffix to a duplicate title instead of failing, so check first.
			page, err := client.Pages.Get(cmd.Context(), project, title)
			if err == nil && page.Persistent {
				return fmt.Errorf("page %q already exists in %s (use 'sb page append')", title, project)
			}
			if err != nil && !errors.Is(err, scrapbox.ErrNotFound) {
				return err
			}
			changes := insertLines(scrapbox.End, append([]string{title}, body...))
			return applyEdit(cmd, app, client, project, "", nil, dryRun, changes)
		},
	}
	in.addFlags(cmd)
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show the result without writing")
	return cmd
}

func newPageAppendCmd(app *App) *cobra.Command {
	var (
		in     textInput
		dryRun bool
	)
	cmd := &cobra.Command{
		Use:   "append <title|url>",
		Short: "Append lines to the end of a page",
		Long: `Append lines to the end of an existing page.

` + pageArgHelp + `

` + textInputHelp + `

` + writeHelp + `

Text output: "Updated <url>" (with --dry-run, the page with added lines marked "+").
JSON output: {commitId, page: {title}, url} (with --dry-run, {previewId, pagePreview, ...})`,
		Example: `  sb page append "Daily log" --text "- finished the review"
  date | sb page append "Daily log"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			lines, err := in.lines(app, true)
			if err != nil {
				return err
			}
			t, err := loadEditTarget(cmd, app, args[0])
			if err != nil {
				return err
			}
			return t.apply(cmd, app, dryRun, insertLines(scrapbox.End, lines))
		},
	}
	in.addFlags(cmd)
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show the result without writing")
	return cmd
}

func newPageInsertCmd(app *App) *cobra.Command {
	var (
		in            textInput
		after, before int
		dryRun        bool
	)
	cmd := &cobra.Command{
		Use:   "insert <title|url> (--after N | --before N)",
		Short: "Insert lines at a position in a page",
		Long: `Insert lines after or before a line of an existing page.

` + pageArgHelp + `

` + textInputHelp + `

` + writeHelp + `

Text output: "Updated <url>" (with --dry-run, the page with added lines marked "+").
JSON output: {commitId, page: {title}, url} (with --dry-run, {previewId, pagePreview, ...})`,
		Example: `  sb page insert "Page" --after 1 --text "Inserted below the title"
  sb page insert "Page" --before 5 --file snippet.txt --dry-run`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if (after == 0) == (before == 0) {
				return errors.New("specify exactly one of --after or --before")
			}
			lines, err := in.lines(app, true)
			if err != nil {
				return err
			}
			t, err := loadEditTarget(cmd, app, args[0])
			if err != nil {
				return err
			}
			n := before
			if after != 0 {
				n = after
			}
			i, err := t.line(n)
			if err != nil {
				return err
			}
			if after != 0 {
				i++
			} else if i == 0 {
				return errors.New("cannot insert before the title")
			}
			return t.apply(cmd, app, dryRun, insertLines(t.anchor(i), lines))
		},
	}
	in.addFlags(cmd)
	cmd.Flags().IntVar(&after, "after", 0, "insert after line N")
	cmd.Flags().IntVar(&before, "before", 0, "insert before line N")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show the result without writing")
	return cmd
}

func newPageReplaceCmd(app *App) *cobra.Command {
	var all, dryRun bool
	cmd := &cobra.Command{
		Use:   "replace <title|url> <old> <new>",
		Short: "Replace text in a page (e.g. fix a typo)",
		Long: `Replace text in the body of an existing page. The title is not changed; use 'sb page rename'.

By default <old> must occur exactly once, so an ambiguous match never changes the wrong line.
Use --all to replace every occurrence.

` + pageArgHelp + `

` + writeHelp + `

Text output: "Updated <url>" (with --dry-run, the page with replaced lines marked "~").
JSON output: {commitId, page: {title}, url} (with --dry-run, {previewId, pagePreview, ...})`,
		Example: `  sb page replace "Page" Scrapobx Scrapbox
  sb page replace "Page" "old name" "new name" --all --dry-run`,
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			oldText, newText := args[1], args[2]
			if oldText == "" || strings.Contains(newText, "\n") {
				return errors.New("<old> must not be empty and <new> must be a single line")
			}
			t, err := loadEditTarget(cmd, app, args[0])
			if err != nil {
				return err
			}
			var (
				changes []scrapbox.Change
				matched []string
				count   int
			)
			for i, l := range t.page.Lines[1:] {
				if n := strings.Count(l.Text, oldText); n > 0 {
					count += n
					matched = append(matched, strconv.Itoa(i+2))
					changes = append(changes, scrapbox.Replace(l.ID, strings.ReplaceAll(l.Text, oldText, newText)))
				}
			}
			switch {
			case count == 0:
				return fmt.Errorf("%q not found in the page body", oldText)
			case count > 1 && !all:
				return fmt.Errorf("%q occurs %d times (lines %s); use --all or a longer <old>", oldText, count, strings.Join(matched, ", "))
			}
			return t.apply(cmd, app, dryRun, changes)
		},
	}
	cmd.Flags().BoolVar(&all, "all", false, "replace every occurrence")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show the result without writing")
	return cmd
}

func newPageDeleteLinesCmd(app *App) *cobra.Command {
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "delete-lines <title|url> <N|N-M>",
		Short: "Delete lines from a page",
		Long: `Delete line N, or lines N to M, from an existing page. The title (line 1) cannot be deleted.

` + pageArgHelp + `

` + writeHelp + `

Text output: "Updated <url>" (with --dry-run, the page with deleted lines marked "-").
JSON output: {commitId, page: {title}, url} (with --dry-run, {previewId, pagePreview, ...})`,
		Example: `  sb page delete-lines "Page" 5
  sb page delete-lines "Page" 5-8 --dry-run`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			from, to, err := parseLineRange(args[1])
			if err != nil {
				return err
			}
			if from == 1 {
				return errors.New("cannot delete the title; use 'sb page delete' to delete the page")
			}
			t, err := loadEditTarget(cmd, app, args[0])
			if err != nil {
				return err
			}
			if _, err := t.line(to); err != nil {
				return err
			}
			var changes []scrapbox.Change
			for _, l := range t.page.Lines[from-1 : to] {
				changes = append(changes, scrapbox.DeleteLine(l.ID))
			}
			return t.apply(cmd, app, dryRun, changes)
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show the result without writing")
	return cmd
}

// parseLineRange parses "N" or "N-M".
func parseLineRange(s string) (from, to int, err error) {
	a, b, isRange := strings.Cut(s, "-")
	from, err = strconv.Atoi(a)
	to = from
	if err == nil && isRange {
		to, err = strconv.Atoi(b)
	}
	if err != nil || from < 1 || to < from {
		return 0, 0, fmt.Errorf("invalid line range %q: use N or N-M (N <= M)", s)
	}
	return from, to, nil
}

func newPageRenameCmd(app *App) *cobra.Command {
	var updateLinks, dryRun bool
	cmd := &cobra.Command{
		Use:   "rename <title|url> <new-title>",
		Short: "Rename a page",
		Long: `Rename an existing page. Fails if a page with the new title exists.
Links to the old title in other pages are kept unless --update-links is given.

` + pageArgHelp + `

` + writeHelp + `

Text output: "Renamed <old> to <new>: <url>" (with --dry-run, the page with the title marked "~").
JSON output: {commitId, page: {title}, titleChanged: {from, to}, url}`,
		Example: `  sb page rename "Old title" "New title"
  sb page rename "Old title" "New title" --update-links`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			newTitle := args[1]
			if newTitle == "" || strings.Contains(newTitle, "\n") {
				return errors.New("<new-title> must be a non-empty single line")
			}
			t, err := loadEditTarget(cmd, app, args[0])
			if err != nil {
				return err
			}
			page, err := t.client.Pages.Get(cmd.Context(), t.project, newTitle)
			if err == nil && page.Persistent && page.ID != t.page.ID {
				return fmt.Errorf("page %q already exists in %s", newTitle, t.project)
			}
			if err != nil && !errors.Is(err, scrapbox.ErrNotFound) {
				return err
			}
			if err := t.apply(cmd, app, dryRun, []scrapbox.Change{scrapbox.Replace(t.page.Lines[0].ID, newTitle)}); err != nil {
				return err
			}
			if !updateLinks || dryRun {
				return nil
			}
			msg, err := t.client.Pages.ReplaceLinks(cmd.Context(), t.project, t.page.Title, newTitle)
			if err != nil {
				return fmt.Errorf("renamed, but updating links failed: %w", err)
			}
			fmt.Fprintln(app.IO.ErrOut, msg)
			return nil
		},
	}
	cmd.Flags().BoolVar(&updateLinks, "update-links", false, "also rewrite links to the old title across the project")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show the result without writing")
	return cmd
}

func newPageDeleteCmd(app *App) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "delete <title|url>",
		Short: "Delete a page",
		Long: `Delete an existing page. Without --yes, only shows the page to be deleted.

` + pageArgHelp + `

Text output: "Deleted <title>" (without --yes, the page to be deleted).
JSON output: {commitId, pageDeleted: {title}} (without --yes, {previewId, pagePreview, pageDelete})`,
		Example: `  sb page delete "Old page"
  sb page delete "Old page" --yes`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			t, err := loadEditTarget(cmd, app, args[0])
			if err != nil {
				return err
			}
			if err := t.apply(cmd, app, !yes, []scrapbox.Change{scrapbox.DeletePage()}); err != nil {
				return err
			}
			if !yes {
				fmt.Fprintln(app.IO.ErrOut, "Run again with --yes to delete.")
			}
			return nil
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "delete without the preview step")
	return cmd
}

func newPageReplaceLinksCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "replace-links <old-title> <new-title>",
		Short: "Rewrite links to a title across the project",
		Long: `Rewrite links ([title]), hashtags (#title) and icons ([title.icon]) pointing to <old-title>
so they point to <new-title>, in every page of the project. Page titles are not changed.
Plain text, code blocks and links to other projects are not changed.

Text output: the server message, e.g. "3 pages have been successfully updated!".
JSON output: {message}`,
		Example: `  sb page replace-links "Old name" "New name"`,
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			project, client, err := app.projectClient()
			if err != nil {
				return err
			}
			msg, err := client.Pages.ReplaceLinks(cmd.Context(), project, args[0], args[1])
			if err != nil {
				return err
			}
			return app.Printer().Print(struct {
				Message string `json:"message"`
			}{msg}, func(w io.Writer) error {
				_, err := fmt.Fprintln(w, msg)
				return err
			})
		},
	}
}
