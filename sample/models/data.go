package models

import (
	"fmt"
	"fyne.io/fyne/v2/data/binding"
)

type Todo struct {
	Project      string
	Workflow     string
	Status       string
	Number       string
	Time         string
	Branch       string
	Sha          string
	Id           string
	TriggerEvent string
}

func NewTodo(p, w, n, i, t, tt string) Todo {
	return Todo{p, w, "red", n, tt, "main", "2a779aeecc39eabc4a99d92169470742a94fc8c0", i, t}
}

func NewTodoFromDataItem(di binding.DataItem) Todo {
	v, _ := di.(binding.Untyped).Get()
	return v.(Todo)
}

func (t Todo) String() string {
	return fmt.Sprintf("%s  - %s  : %s", t.Project, t.Workflow, t.Id)
}
