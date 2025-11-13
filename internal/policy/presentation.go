package policy

// Presentation UI presentation
type Presentation struct {
	Name     string
	Elements []PresentationElement
}

// PresentationElement presentation element
type PresentationElement interface {
	GetID() string
	GetElementType() string
}

// BasePresentationElement base presentation element
type BasePresentationElement struct {
	ID          string
	ElementType string
}

func (b *BasePresentationElement) GetID() string          { return b.ID }
func (b *BasePresentationElement) GetElementType() string { return b.ElementType }

// LabelPresentationElement label element
type LabelPresentationElement struct {
	BasePresentationElement
	Text string
}

// NumericBoxPresentationElement numeric box
type NumericBoxPresentationElement struct {
	BasePresentationElement
	DefaultValue     uint32
	HasSpinner       bool
	SpinnerIncrement uint32
	Label            string
}

// TextBoxPresentationElement text box
type TextBoxPresentationElement struct {
	BasePresentationElement
	Label        string
	DefaultValue string
}

// CheckBoxPresentationElement checkbox
type CheckBoxPresentationElement struct {
	BasePresentationElement
	DefaultState bool
	Text         string
}

// ComboBoxPresentationElement combo box
type ComboBoxPresentationElement struct {
	BasePresentationElement
	NoSort      bool
	Label       string
	DefaultText string
	Suggestions []string
}

// DropDownPresentationElement dropdown list
type DropDownPresentationElement struct {
	BasePresentationElement
	NoSort        bool
	DefaultItemID *int
	Label         string
}

// ListPresentationElement list
type ListPresentationElement struct {
	BasePresentationElement
	Label string
}

// MultiTextPresentationElement multi text
type MultiTextPresentationElement struct {
	BasePresentationElement
	Label string
}
