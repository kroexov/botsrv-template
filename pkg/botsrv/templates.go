package botsrv

import (
	"bytes"
	"fmt"
	"gold-botsrv/pkg/db"
	"text/template"
	"time"
)

// Функции для шаблона
var templateFuncs = template.FuncMap{
	"formatTime": func(t time.Time) string {
		return t.Format("02.01.2006 15:04")
	},
	"formatTimePtr": func(t *time.Time) string {
		if t == nil {
			return "не назначено"
		}
		return t.Format("02.01.2006 15:04")
	},
}

// Шаблон для одной задачи
const taskTemplate = `ID: {{.ID}} 
Описание: {{.Description}}  
Приоритет: {{.Priority}} 
Займет: {{.Length}} минут 
Дедлайн: {{formatTime .Deadline}} 
Начнем: {{formatTimePtr .StartAt}}
{{if eq .StatusID 2}}❌ Не получилось назначить время{{end}}
`

// Шаблон для списка задач
const tasksListTemplate = `Задачи:

{{range .}}----------------------------------------
{{template "task" .}}
{{end}}`

// Функция для генерации сообщения из списка задач
func FormatTasksMessage(tasks []db.Task) (string, error) {
	tmpl := template.New("tasksList")
	tmpl.Funcs(templateFuncs)

	// Сначала парсим шаблон задачи
	_, err := tmpl.Parse("{{define \"task\"}}" + taskTemplate + "{{end}}")
	if err != nil {
		return "", fmt.Errorf("failed to parse task template: %w", err)
	}

	// Затем парсим основной шаблон списка
	tmpl, err = tmpl.Parse(tasksListTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse tasks list template: %w", err)
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, tasks)
	if err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}
