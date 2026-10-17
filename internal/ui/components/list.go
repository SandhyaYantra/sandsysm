package components

import "strings"

type ListItem struct {
	Title string
	Desc  string
}

type ListModel struct {
	Items    []ListItem
	Selected int
}

func (list ListModel) View() string {
	if len(list.Items) == 0 {
		return "(no items)"
	}
	selected := list.Selected
	if selected < 0 {
		selected = 0
	}
	if selected >= len(list.Items) {
		selected = len(list.Items) - 1
	}

	var view strings.Builder
	for index, item := range list.Items {
		prefix := "  "
		line := item.Title
		if item.Desc != "" {
			line += "  " + item.Desc
		}
		if index == selected {
			prefix = "> "
			line = StyleTitle.Render(line)
		}
		if index > 0 {
			view.WriteByte('\n')
		}
		view.WriteString(prefix)
		view.WriteString(line)
	}
	return view.String()
}

func (list *ListModel) Up() {
	if list != nil && list.Selected > 0 {
		list.Selected--
	}
}

func (list *ListModel) Down() {
	if list != nil && list.Selected < len(list.Items)-1 {
		list.Selected++
	}
}
