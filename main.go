package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"
	"github.com/kakengloh/tsk/driver"
	"github.com/kakengloh/tsk/entity"
	"github.com/kakengloh/tsk/repository"
	"github.com/xeonx/timeago"
)

const (
	columnKeyID       = "id"
	columnKeyTitle    = "title"
	columnKeyStatus   = "status"
	columnKeyPriority = "priority"
	columnKeyCreated  = "created"
	columnKeyDueDate  = "due_date"
	columnKeyNotes    = "notes"
)

var (
	customBorder = table.Border{
		Top:    "─",
		Left:   "│",
		Right:  "│",
		Bottom: "─",

		TopRight:    "╮",
		TopLeft:     "╭",
		BottomRight: "╯",
		BottomLeft:  "╰",

		TopJunction:    "╥",
		LeftJunction:   "├",
		RightJunction:  "┤",
		BottomJunction: "╨",
		InnerJunction:  "╫",

		InnerDivider: "║",
	}
)

func taskDueAsString(task entity.Task) string {
	due := ""

	if !task.Due.IsZero() {
		if time.Now().Before(task.Due) {
			if time.Until(task.Due) < 24*time.Hour {
				due = timeago.English.Format(task.Due)
			} else {
				due = task.Due.Format("2006-01-02 15:04:05")
			}
		} else {
			due = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#cc241d")).
				Render(timeago.English.Format(task.Due))
		}
	}

	return due
}

type Model struct {
	taskRepository repository.TaskRepository
	tableModel     table.Model
}

func NewModel(tr repository.TaskRepository) Model {
	columns := []table.Column{
		table.NewColumn(columnKeyID, "ID", 5).WithStyle(
			lipgloss.NewStyle().
				Faint(true).
				Foreground(lipgloss.Color("#fabd2f")).
				Align(lipgloss.Center)),
		table.NewColumn(columnKeyTitle, "Title", 20),
		table.NewColumn(columnKeyStatus, "Status", 6),
		table.NewColumn(columnKeyPriority, "Priority", 8),
		table.NewColumn(columnKeyCreated, "Created", 19),
		table.NewColumn(columnKeyDueDate, "Due Date", 19),
		table.NewColumn(columnKeyNotes, "Notes", 16), // TODO FlexColumn
	}

	keys := table.DefaultKeyMap()
	keys.RowDown.SetKeys("j", "down", "s")
	keys.RowUp.SetKeys("k", "up", "w")

	model := Model{
		tableModel: table.New(columns).
			WithKeyMap(keys).
			Focused(true).
			Border(customBorder).
			HeaderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("#83a598")).Bold(true)).
			WithBaseStyle(
				lipgloss.NewStyle().
					BorderForeground(lipgloss.Color("#689d6a")).
					Foreground(lipgloss.Color("#b8bb26")).
					Align(lipgloss.Left),
			).
			HighlightStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("#fabd2f")).Background(lipgloss.Color("#3c3836"))).
			SortByAsc(columnKeyID),
		taskRepository: tr,
	}

	model = updateRows(model)

	return model
}

func updateRows(m Model) Model {
	rows := []table.Row{}

	// TODO task filters
	tasks, err := m.taskRepository.ListTasksWithFilters(entity.TaskFilters{
		Status:   0,
		Priority: 0,
	})
	if err != nil {
		log.Fatalf("failed to list tasks: %s", err)
	}

	for _, task := range tasks {
		rows = append(rows, table.NewRow(table.RowData{
			columnKeyID:       fmt.Sprintf("%d", task.ID),
			columnKeyTitle:    task.Title,
			columnKeyStatus:   entity.TaskStatusToString[task.Status],
			columnKeyPriority: entity.TaskPriorityToString[task.Priority],
			columnKeyCreated:  task.CreatedAt.Format("2006-01-02 15:04:05"),
			columnKeyDueDate:  taskDueAsString(task),
			columnKeyNotes:    strings.Join(task.Notes, "\n"),
		}))
	}

	m.tableModel = m.tableModel.WithRows(rows)

	return m
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	m.tableModel, cmd = m.tableModel.Update(msg)
	cmds = append(cmds, cmd)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			cmds = append(cmds, tea.Quit)

		case "h":
			m.tableModel = m.tableModel.WithHeaderVisibility(!m.tableModel.GetHeaderVisibility())
		}
	}

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	return m.tableModel.View() + "\n"
}

func main() {
	// Database
	db, err := driver.NewBolt()
	if err != nil {
		log.Fatalf("failed to connect to BoltDB: %s", err)
	}
	defer driver.CloseBolt()

	// Task repository
	tr, err := repository.NewBoltTaskRepository(db)
	if err != nil {
		log.Fatalf("failed to initialize task repository: %s", err)
	}

	p := tea.NewProgram(NewModel(tr))
	if err := p.Start(); err != nil {
		log.Fatal(err)
	}
}
