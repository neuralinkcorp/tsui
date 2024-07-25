package ui

import (
	"slices"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// A generic submenu item.
type SubmenuItem interface {
	// Returns true if this item can be navigated to by the user.
	isSelectable() bool
	// Triggers the callback for item activation. Should eagerly update the activation status.
	// Should do nothing if the item is already active. Returns a bubbletea command that can
	// be run asynchronously.
	onActivate() tea.Cmd
	// If applicable, "un-toggles" the item.
	clearActiveFlag()
	// Renders the item. isSelected will always be false if isSelectable() returns false.
	render(isSelected bool, isSubmenuOpen bool) string
}

const submenuItemWidth = 45

// Visual variant of a submenu item:
//
//	SubmenuItemVariantDefault, SubmenuItemVariantDanger
type SubmenuItemVariant int

const (
	SubmenuItemVariantDefault SubmenuItemVariant = iota
	SubmenuItemVariantDanger
)

// A menu item with a label.
type LabeledSubmenuItem struct {
	// The text to be displayed for this menu item.
	Label string
	// Visual variant.
	Variant SubmenuItemVariant
	// Callback when the item is activated.
	OnActivate tea.Cmd
	// Whether this item is visibly de-emphasized.
	IsDim bool
}

func (item *LabeledSubmenuItem) isSelectable() bool {
	return true
}

func (item *LabeledSubmenuItem) onActivate() tea.Cmd {
	return item.OnActivate
}

func (item *LabeledSubmenuItem) clearActiveFlag() {
	// No-op because this item is not toggleable.
}

func (item *LabeledSubmenuItem) render(isSelected bool, isSubmenuOpen bool) string {
	style := lipgloss.NewStyle().
		PaddingRight(1).
		PaddingLeft(2).
		Width(submenuItemWidth)

	if isSubmenuOpen {
		if isSelected {
			style = style.
				Background(Secondary).
				Foreground(Black)
		} else if item.IsDim {
			style = style.
				Faint(true)
		} else if item.Variant == SubmenuItemVariantDanger {
			style = style.
				Foreground(Red)
		}
	} else {
		style = style.
			Faint(true)
	}

	return style.Render(item.Label)
}

// A menu item with a label that can be toggled active or inactive.
type ToggleableSubmenuItem struct {
	LabeledSubmenuItem
	// Whether this item is currently active.
	IsActive bool
}

func (item *ToggleableSubmenuItem) onActivate() tea.Cmd {
	if item.IsActive {
		return nil
	}
	item.IsActive = true
	return item.LabeledSubmenuItem.onActivate()
}

func (item *ToggleableSubmenuItem) clearActiveFlag() {
	item.IsActive = false
}

func (item *ToggleableSubmenuItem) render(isSelected bool, isSubmenuOpen bool) string {
	style := lipgloss.NewStyle().
		Padding(0, 1).
		Width(submenuItemWidth)

	if isSubmenuOpen {
		if item.IsActive {
			style = style.
				Bold(true).
				Foreground(Secondary)
		}

		if isSelected {
			style = style.
				Background(Secondary).
				Foreground(Black)
		} else if item.IsDim {
			style = style.
				Faint(true)
		} else if item.Variant == SubmenuItemVariantDanger && !item.IsActive {
			style = style.
				Foreground(Red)
		}
	} else {
		style = style.
			Faint(true)
	}

	labelPrefix := " "
	if item.IsActive {
		labelPrefix = "*"
	}

	return style.Render(labelPrefix + item.Label)
}

// A submenu item for a "settings control" that can have multiple values and activated to switch between them.
type SettingSubmenuItem struct {
	// Name of this setting.
	Label string
	// Callback when a new value is selected.
	OnChange func(newLabel string) tea.Msg
	// The value options.
	options []string
	// The currently selected value.
	selected int
}

// Create a new SettingsSubmenuItem. initialOption must be one of the options.
func NewSettingsSubmenuItem(label string, options []string, initialOption string, onChange func(newLabel string) tea.Msg) *SettingSubmenuItem {
	return &SettingSubmenuItem{
		Label:    label,
		options:  options,
		selected: slices.Index(options, initialOption),
		OnChange: onChange,
	}
}

// Create a new SettingsSubmenuItem that is just a yes/no toggle.
func NewYesNoSettingsSubmenuItem(label string, initialValue bool, onChange func(newValue bool) tea.Msg) *SettingSubmenuItem {
	var initialValueString string
	if initialValue {
		initialValueString = "Yes"
	} else {
		initialValueString = "No"
	}

	onStringChange := func(newLabel string) tea.Msg {
		if newLabel == "Yes" {
			return onChange(true)
		} else {
			return onChange(false)
		}
	}

	return NewSettingsSubmenuItem(label, []string{"Yes", "No"}, initialValueString, onStringChange)
}

func (item *SettingSubmenuItem) isSelectable() bool {
	return true
}

func (item *SettingSubmenuItem) onActivate() tea.Cmd {
	item.selected++
	if item.selected >= len(item.options) {
		item.selected = 0
	}
	newLabel := item.options[item.selected]
	return func() tea.Msg {
		return item.OnChange(newLabel)
	}
}

func (item *SettingSubmenuItem) clearActiveFlag() {}

func (item *SettingSubmenuItem) render(isSelected bool, isSubmenuOpen bool) string {
	selectedLabel := item.options[item.selected]

	// Label with right-side padding to align the value label to the right.
	expandedLabel := lipgloss.NewStyle().
		Width(submenuItemWidth - lipgloss.Width(selectedLabel) - 3). // 3 for the padding
		Render(item.Label)

	style := lipgloss.NewStyle().
		PaddingRight(1).
		PaddingLeft(2).
		Width(submenuItemWidth)
	selectedLabelStyle := lipgloss.NewStyle()

	if isSubmenuOpen {
		if isSelected {
			style = style.
				Background(Secondary).
				Foreground(Black)

			selectedLabelStyle = selectedLabelStyle.
				Bold(true)
		} else {
			var color lipgloss.Color

			// This is kinda janky but hey, why not style the value by its contents?
			switch selectedLabel {
			case "Yes", "On":
				color = Green
			case "No", "Off":
				color = Red
			default:
				color = Blue
			}

			selectedLabelStyle = selectedLabelStyle.
				Foreground(color)
		}
	} else {
		style = style.
			Faint(true)
	}

	return style.Render(expandedLabel + selectedLabelStyle.Render(selectedLabel))
}

// A divider in a menu.
type DividerSubmenuItem struct{}

func (d *DividerSubmenuItem) isSelectable() bool {
	return false
}

func (d *DividerSubmenuItem) onActivate() tea.Cmd {
	return nil
}

func (d *DividerSubmenuItem) clearActiveFlag() {}

func (d *DividerSubmenuItem) render(isSelected bool, isSubmenuOpen bool) string {
	return lipgloss.NewStyle().
		Faint(true).
		Render("  --")
}

// An empty spacer line.
type SpacerSubmenuItem struct{}

func (s *SpacerSubmenuItem) isSelectable() bool {
	return false
}

func (s *SpacerSubmenuItem) onActivate() tea.Cmd {
	return nil
}

func (s *SpacerSubmenuItem) clearActiveFlag() {}

func (s *SpacerSubmenuItem) render(isSelected bool, isSubmenuOpen bool) string {
	return ""
}

// A title for a section of a menu.
type TitleSubmenuItem struct {
	// The text to be displayed for this menu item.
	Label string
}

func (i *TitleSubmenuItem) isSelectable() bool {
	return false
}

func (i *TitleSubmenuItem) onActivate() tea.Cmd {
	return nil
}

func (i *TitleSubmenuItem) clearActiveFlag() {}

func (i *TitleSubmenuItem) render(isSelected bool, isSubmenuOpen bool) string {
	return lipgloss.NewStyle().
		Faint(true).
		PaddingLeft(2).
		Render(i.Label)
}

type SubmenuExclusivity int

const (
	// No exclusivity, all updates just toggle the new item.
	SubmenuExclusivityNone SubmenuExclusivity = iota
	// All updates first clear the active state of all other items.
	SubmenuExclusivityOne
)

// State container for a submenu with its cursor and items.
type Submenu struct {
	// Controls exclusivity behavior of eager updates:
	//   SubmenuExclusivityNone, SubmenuExclusivityOne
	Exclusivity SubmenuExclusivity
	items       []SubmenuItem
	cursor      int
}

// Render the submenu to a string.
func (submenu *Submenu) Render(isSubmenuOpen bool) string {
	s := ""
	for i, item := range submenu.items {
		if i > 0 {
			s += "\n"
		}
		s += item.render(i == submenu.cursor && item.isSelectable(), isSubmenuOpen)
	}
	return s
}

// Move the cursor to the next selectable item.
func (submenu *Submenu) CursorDown() {
	for i := submenu.cursor + 1; i < len(submenu.items); i++ {
		if submenu.items[i].isSelectable() {
			submenu.cursor = i
			return
		}
	}
}

// Move the cursor to the previous selectable item.
func (submenu *Submenu) CursorUp() {
	for i := submenu.cursor - 1; i >= 0; i-- {
		if submenu.items[i].isSelectable() {
			submenu.cursor = i
			return
		}
	}
}

// Reset the cursor to the first selectable item.
func (submenu *Submenu) ResetCursor() {
	for i, item := range submenu.items {
		if item.isSelectable() {
			submenu.cursor = i
			return
		}
	}
}

// Set the items list and ensure the cursor is within bounds and on a selectable item.
func (submenu *Submenu) SetItems(items []SubmenuItem) {
	submenu.items = items
	submenu.fixCursor()
}

// Ensure the cursor is within bounds and on a selectable item. Call after major updates to the items.
func (submenu *Submenu) fixCursor() {
	if len(submenu.items) == 0 {
		submenu.cursor = 0
		return
	}
	if submenu.cursor > len(submenu.items)-1 {
		submenu.cursor = len(submenu.items) - 1
	}

	if !submenu.items[submenu.cursor].isSelectable() {
		submenu.ResetCursor()
	}
}

// Call the currently selected item's activate callback.
// Returns a bubbletea command that can be run asynchronously.
func (submenu *Submenu) Activate() tea.Cmd {
	if submenu.cursor < 0 || submenu.cursor >= len(submenu.items) {
		return nil
	}

	if submenu.Exclusivity == SubmenuExclusivityOne {
		for _, item := range submenu.items {
			item.clearActiveFlag()
		}
	}

	item := submenu.items[submenu.cursor]
	return item.onActivate()
}
