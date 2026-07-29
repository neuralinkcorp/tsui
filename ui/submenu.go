package ui

import (
	"slices"
	"strings"

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
	render(isSelected bool, isSubmenuOpen bool, isFocused bool, width int) string
}

const (
	defaultSubmenuItemWidth = 45
	minSubmenuItemWidth     = 24
)

// Visual variant of a submenu item:
//
//	SubmenuItemVariantDefault, SubmenuItemVariantDanger
type SubmenuItemVariant int

const (
	SubmenuItemVariantDefault SubmenuItemVariant = iota
	SubmenuItemVariantAccent
	SubmenuItemVariantDanger
)

func (v SubmenuItemVariant) color() lipgloss.Color {
	switch v {
	case SubmenuItemVariantAccent:
		return Secondary
	case SubmenuItemVariantDanger:
		return Red
	}

	return lipgloss.Color("")
}

// A menu item with a label.
type LabeledSubmenuItem struct {
	// The text to be displayed for this menu item.
	Label string
	// An extra label shown on the right side. Will be shown in a muted color.
	AdditionalLabel string
	// Optional submenu shown to the right when this item is selected.
	ChildSubmenu *Submenu
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

func (item *LabeledSubmenuItem) childSubmenu() *Submenu {
	return item.ChildSubmenu
}

func (item *LabeledSubmenuItem) render(isSelected bool, isSubmenuOpen bool, isFocused bool, width int) string {
	colorStyle := lipgloss.NewStyle()

	if isSubmenuOpen {
		if isSelected && isFocused {
			if item.Variant == SubmenuItemVariantDanger {
				colorStyle = colorStyle.
					Background(Red).
					Foreground(Black)
			} else {
				colorStyle = colorStyle.
					Background(Secondary).
					Foreground(Black)
			}
		} else if isSelected {
			colorStyle = colorStyle.
				Background(DarkGray).
				Faint(true)
		} else if item.IsDim {
			colorStyle = colorStyle.
				Faint(true)
		}

		if item.Variant != SubmenuItemVariantDefault {
			if isSelected {
				colorStyle = colorStyle.
					Bold(true)
			} else if !item.IsDim {
				colorStyle = colorStyle.
					Foreground(item.Variant.color())
			}
		}
	} else {
		colorStyle = colorStyle.
			Faint(true)
	}

	outerStyle := colorStyle.
		PaddingRight(1).
		PaddingLeft(2).
		Width(width)
	innerWidth := width - outerStyle.GetHorizontalPadding()
	rightLabel := colorStyle.
		Faint(true).
		Render(item.AdditionalLabel)

	content := RenderSplit(
		colorStyle.Render(item.Label),
		rightLabel,
		innerWidth,
		colorStyle,
	)
	if item.AdditionalLabel != "" && lipgloss.Width(item.Label)+lipgloss.Width(item.AdditionalLabel)+1 > innerWidth {
		content = colorStyle.Render(truncateString(item.Label, innerWidth)) + "\n" +
			colorStyle.
				Faint(true).
				Width(innerWidth).
				Render(item.AdditionalLabel)
	}

	return outerStyle.Render(content)
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

func (item *ToggleableSubmenuItem) render(isSelected bool, isSubmenuOpen bool, isFocused bool, width int) string {
	colorStyle := lipgloss.NewStyle()

	if isSubmenuOpen {
		if item.IsActive {
			colorStyle = colorStyle.
				Bold(true)

			if item.Variant == SubmenuItemVariantDefault {
				colorStyle = colorStyle.
					Foreground(Secondary)
			}
		}

		if isSelected && isFocused {
			colorStyle = colorStyle.
				Background(Secondary).
				Foreground(Black)
		} else if isSelected {
			colorStyle = colorStyle.
				Background(DarkGray).
				Faint(true)
		} else if item.IsDim {
			colorStyle = colorStyle.
				Faint(true)
		}

		if item.Variant != SubmenuItemVariantDefault {
			if isSelected {
				colorStyle = colorStyle.
					Bold(true)
			} else if !item.IsDim {
				colorStyle = colorStyle.
					Foreground(item.Variant.color())
			}
		}
	} else {
		colorStyle = colorStyle.
			Faint(true)
	}

	labelPrefix := " "
	if item.IsActive {
		labelPrefix = "*"
	}

	outerStyle := colorStyle.
		Padding(0, 1).
		Width(width)

	return outerStyle.Render(
		RenderSplit(
			labelPrefix+item.Label,
			colorStyle.
				Faint(true).
				Render(item.AdditionalLabel),
			width-outerStyle.GetHorizontalPadding(),
			colorStyle,
		),
	)
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

func (item *SettingSubmenuItem) render(isSelected bool, isSubmenuOpen bool, isFocused bool, width int) string {
	selectedLabel := item.options[item.selected]

	style := lipgloss.NewStyle().
		PaddingRight(1).
		PaddingLeft(2).
		Width(width)
	selectedLabelStyle := lipgloss.NewStyle()

	if isSubmenuOpen {
		if isSelected && isFocused {
			style = style.
				Background(Secondary).
				Foreground(Black)

			selectedLabelStyle = selectedLabelStyle.
				Bold(true)
		} else if isSelected {
			style = style.
				Background(DarkGray).
				Faint(true)
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

	return style.Render(
		RenderSplit(
			item.Label,
			selectedLabelStyle.Render(selectedLabel),
			width-style.GetHorizontalPadding(),
			lipgloss.NewStyle(),
		),
	)
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

func (d *DividerSubmenuItem) render(isSelected bool, isSubmenuOpen bool, isFocused bool, width int) string {
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

func (s *SpacerSubmenuItem) render(isSelected bool, isSubmenuOpen bool, isFocused bool, width int) string {
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

func (i *TitleSubmenuItem) render(isSelected bool, isSubmenuOpen bool, isFocused bool, width int) string {
	return lipgloss.NewStyle().
		Faint(true).
		PaddingLeft(2).
		Width(width).
		Render(truncateString(i.Label, width-2))
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
	childOpen   bool
}

type childSubmenuItem interface {
	childSubmenu() *Submenu
}

// A rendered submenu item and its computed layout info.
type ComputedSubmenuItem struct {
	text   string
	height int
}

// Render the submenu to a fixed-height string with scrolling.
func (submenu *Submenu) Render(isSubmenuOpen bool, height int, width int) string {
	if width <= 0 {
		return ""
	}

	itemWidth := min(defaultSubmenuItemWidth, width)
	childWidth := 0
	if submenu.selectedChildSubmenu() != nil {
		childWidth = min(defaultSubmenuItemWidth, max(minSubmenuItemWidth, width/2))
		itemWidth = max(minSubmenuItemWidth, width-childWidth)
	}

	// Render all of the computedItems to strings so we can work with their computed heights.
	computedItems := make([]ComputedSubmenuItem, len(submenu.items))
	isFocused := isSubmenuOpen && !submenu.childOpen
	for i, item := range submenu.items {
		text := item.render(i == submenu.cursor && item.isSelectable(), isSubmenuOpen, isFocused, itemWidth)

		computedItems[i] = ComputedSubmenuItem{
			text:   text,
			height: lipgloss.Height(text),
		}
	}

	// Submenus can scroll, and each item can be any number of lines, so our layout engine
	// needs to be a little bit smart.

	// 1. Start with the selected item. We will always display this.
	topOverflow := false
	bottomOverflow := false
	rangeStart := submenu.cursor
	rangeEnd := submenu.cursor + 1
	totalHeight := computedItems[submenu.cursor].height

	// 2. Add items backward from the selected item...
	for i := rangeStart - 1; i >= 0; i-- {
		// ... until an item doesn't fit. In this case, show a "..." indicator at the top instead
		// of adding that item.
		if totalHeight+computedItems[i].height >= height && i != 0 {
			topOverflow = true
			totalHeight++
			break
		}

		rangeStart = i
		totalHeight += computedItems[i].height
	}

	// 3. Add items forward from the selected item...
	for i := rangeEnd; i < len(computedItems); i++ {
		// ... until an item doesn't fit. In this case, show a "..." indicator at the bottom
		// instead of adding that item.
		if totalHeight+computedItems[i].height >= height && i != len(computedItems)-1 {
			bottomOverflow = true
			totalHeight++
			break
		}

		rangeEnd = i + 1
		totalHeight += computedItems[i].height
	}

	// 4. Make sure everything fits after adding the overflow indicators by popping
	//    items from the start until we no longer exceed the height.
	for totalHeight > height {
		if !topOverflow {
			topOverflow = true
			totalHeight++
		}

		totalHeight -= computedItems[rangeStart].height
		rangeStart++
	}

	// 5. Clamp the range so we can NEVER crash.
	rangeStart = max(0, rangeStart)
	rangeEnd = max(rangeStart, min(rangeEnd, len(computedItems)))

	// Now we have a range of all menu items that can fit on screen, and we can create the
	// final string.
	overflow := lipgloss.NewStyle().
		Background(DarkGray).
		MarginLeft(2).
		Render("...")

	var s strings.Builder

	// Add the top overflow indicator.
	if topOverflow {
		s.WriteString(overflow + "\n")

		// If we have a top overflow indicator but aren't using the whole height of the screen,
		// one of the items was oddly sized. Add some padding underneath the overflow indicator
		// to keep all of the other items aligned to the bottom.
		for i := 0; i < height-totalHeight; i++ {
			s.WriteByte('\n')
		}
	}

	// Add the rendered items.
	for i, item := range computedItems[rangeStart:rangeEnd] {
		if i != 0 {
			s.WriteByte('\n')
		}
		s.WriteString(item.text)
	}

	// Add the bottom overflow indicator.
	if bottomOverflow {
		s.WriteString("\n" + overflow)
	}

	rendered := s.String()
	if child := submenu.selectedChildSubmenu(); isSubmenuOpen && child != nil {
		return lipgloss.JoinHorizontal(lipgloss.Top,
			rendered,
			child.Render(isSubmenuOpen && submenu.childOpen, height, childWidth))
	}

	return rendered
}

func (submenu *Submenu) selectedChildSubmenu() *Submenu {
	if submenu.cursor < 0 || submenu.cursor >= len(submenu.items) {
		return nil
	}

	item, ok := submenu.items[submenu.cursor].(childSubmenuItem)
	if !ok {
		return nil
	}

	return item.childSubmenu()
}

// Move the cursor to the next selectable item.
func (submenu *Submenu) CursorDown() {
	if submenu.childOpen {
		if child := submenu.selectedChildSubmenu(); child != nil {
			child.CursorDown()
			return
		}
	}

	for i := submenu.cursor + 1; i < len(submenu.items); i++ {
		if submenu.items[i].isSelectable() {
			submenu.cursor = i
			submenu.childOpen = false
			return
		}
	}
}

// Move the cursor to the previous selectable item.
func (submenu *Submenu) CursorUp() {
	if submenu.childOpen {
		if child := submenu.selectedChildSubmenu(); child != nil {
			child.CursorUp()
			return
		}
	}

	for i := submenu.cursor - 1; i >= 0; i-- {
		if submenu.items[i].isSelectable() {
			submenu.cursor = i
			submenu.childOpen = false
			return
		}
	}
}

// Reset the cursor to the first selectable item.
func (submenu *Submenu) ResetCursor() {
	submenu.childOpen = false
	for i, item := range submenu.items {
		if item.isSelectable() {
			submenu.cursor = i
			return
		}
	}
}

// Set the items list and ensure the cursor is within bounds and on a selectable item.
func (submenu *Submenu) SetItems(items []SubmenuItem) {
	childOpen := submenu.childOpen
	childCursor := 0
	if child := submenu.selectedChildSubmenu(); child != nil {
		childCursor = child.cursor
	}

	submenu.items = items
	submenu.fixCursor()
	submenu.childOpen = childOpen && submenu.selectedChildSubmenu() != nil
	if submenu.childOpen {
		child := submenu.selectedChildSubmenu()
		child.cursor = childCursor
		child.fixCursor()
	}
}

func (submenu *Submenu) OpenChildSubmenu() bool {
	child := submenu.selectedChildSubmenu()
	if child == nil {
		return false
	}

	submenu.childOpen = true
	child.ResetCursor()
	return true
}

func (submenu *Submenu) CloseChildSubmenu() bool {
	if !submenu.childOpen {
		return false
	}

	submenu.childOpen = false
	return true
}

func (submenu *Submenu) HasOpenChildSubmenu() bool {
	return submenu.childOpen && submenu.selectedChildSubmenu() != nil
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
	if submenu.childOpen {
		if child := submenu.selectedChildSubmenu(); child != nil {
			return child.Activate()
		}
	}

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
