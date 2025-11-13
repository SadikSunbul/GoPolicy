package policy

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strconv"
)

// AdmlFile ADML localization file
type AdmlFile struct {
	SourceFile        string
	Revision          float64
	DisplayName       string
	Description       string
	StringTable       map[string]string
	PresentationTable map[string]*Presentation
}

// ADML XML structures
type admlPolicyDefinitionResources struct {
	XMLName           xml.Name               `xml:"policyDefinitionResources"`
	Revision          string                 `xml:"revision,attr"`
	DisplayName       string                 `xml:"displayName"`
	Description       string                 `xml:"description"`
	StringTable       *admlStringTable       `xml:"resources>stringTable"`
	PresentationTable *admlPresentationTable `xml:"resources>presentationTable"`
}

type admlStringTable struct {
	Strings []admlString `xml:"string"`
}

type admlString struct {
	ID    string `xml:"id,attr"`
	Value string `xml:",chardata"`
}

type admlPresentationTable struct {
	Presentations []admlPresentation `xml:"presentation"`
}

type admlPresentation struct {
	ID               string               `xml:"id,attr"`
	Texts            []string             `xml:"text"`
	DecimalTextBoxes []admlDecimalTextBox `xml:"decimalTextBox"`
	TextBoxes        []admlTextBox        `xml:"textBox"`
	CheckBoxes       []admlCheckBox       `xml:"checkBox"`
	ComboBoxes       []admlComboBox       `xml:"comboBox"`
	DropdownLists    []admlDropdownList   `xml:"dropdownList"`
	ListBoxes        []admlListBox        `xml:"listBox"`
	MultiTextBoxes   []admlMultiTextBox   `xml:"multiTextBox"`
}

type admlDecimalTextBox struct {
	RefID        string `xml:"refId,attr"`
	DefaultValue string `xml:"defaultValue,attr"`
	Spin         string `xml:"spin,attr"`
	SpinStep     string `xml:"spinStep,attr"`
	Text         string `xml:",chardata"`
}

type admlTextBox struct {
	RefID        string `xml:"refId,attr"`
	Label        string `xml:"label"`
	DefaultValue string `xml:"defaultValue"`
}

type admlCheckBox struct {
	RefID          string `xml:"refId,attr"`
	DefaultChecked string `xml:"defaultChecked,attr"`
	Text           string `xml:",chardata"`
}

type admlComboBox struct {
	RefID       string   `xml:"refId,attr"`
	NoSort      string   `xml:"noSort,attr"`
	Label       string   `xml:"label"`
	Default     string   `xml:"default"`
	Suggestions []string `xml:"suggestion"`
}

type admlDropdownList struct {
	RefID       string `xml:"refId,attr"`
	NoSort      string `xml:"noSort,attr"`
	DefaultItem string `xml:"defaultItem,attr"`
	Text        string `xml:",chardata"`
}

type admlListBox struct {
	RefID string `xml:"refId,attr"`
	Text  string `xml:",chardata"`
}

type admlMultiTextBox struct {
	RefID string `xml:"refId,attr"`
	Text  string `xml:",chardata"`
}

// LoadAdmlFile loads ADML file
func LoadAdmlFile(path string) (*AdmlFile, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var policyResources admlPolicyDefinitionResources
	if err := xml.Unmarshal(data, &policyResources); err != nil {
		return nil, fmt.Errorf("XML parse error: %w", err)
	}

	adml := &AdmlFile{
		SourceFile:        path,
		DisplayName:       policyResources.DisplayName,
		Description:       policyResources.Description,
		StringTable:       make(map[string]string),
		PresentationTable: make(map[string]*Presentation),
	}

	if policyResources.Revision != "" {
		adml.Revision, _ = strconv.ParseFloat(policyResources.Revision, 64)
	}

	// String table
	if policyResources.StringTable != nil {
		for _, str := range policyResources.StringTable.Strings {
			adml.StringTable[str.ID] = str.Value
		}
	}

	// Presentation table
	if policyResources.PresentationTable != nil {
		for _, pres := range policyResources.PresentationTable.Presentations {
			presentation := &Presentation{
				Name:     pres.ID,
				Elements: []PresentationElement{},
			}

			// Text elements
			for _, txt := range pres.Texts {
				elem := &LabelPresentationElement{
					BasePresentationElement: BasePresentationElement{
						ElementType: "text",
					},
					Text: txt,
				}
				presentation.Elements = append(presentation.Elements, elem)
			}

			// DecimalTextBox elements
			for _, dtb := range pres.DecimalTextBoxes {
				elem := &NumericBoxPresentationElement{
					BasePresentationElement: BasePresentationElement{
						ID:          dtb.RefID,
						ElementType: "decimalTextBox",
					},
					HasSpinner:       dtb.Spin != "false",
					SpinnerIncrement: 1,
					Label:            dtb.Text,
				}
				if dtb.DefaultValue != "" {
					val, _ := strconv.ParseUint(dtb.DefaultValue, 10, 32)
					elem.DefaultValue = uint32(val)
				}
				if dtb.SpinStep != "" {
					step, _ := strconv.ParseUint(dtb.SpinStep, 10, 32)
					elem.SpinnerIncrement = uint32(step)
				}
				presentation.Elements = append(presentation.Elements, elem)
			}

			// TextBox elements
			for _, tb := range pres.TextBoxes {
				elem := &TextBoxPresentationElement{
					BasePresentationElement: BasePresentationElement{
						ID:          tb.RefID,
						ElementType: "textBox",
					},
					Label:        tb.Label,
					DefaultValue: tb.DefaultValue,
				}
				presentation.Elements = append(presentation.Elements, elem)
			}

			// CheckBox elements
			for _, cb := range pres.CheckBoxes {
				elem := &CheckBoxPresentationElement{
					BasePresentationElement: BasePresentationElement{
						ID:          cb.RefID,
						ElementType: "checkBox",
					},
					DefaultState: cb.DefaultChecked == "true",
					Text:         cb.Text,
				}
				presentation.Elements = append(presentation.Elements, elem)
			}

			// ComboBox elements
			for _, cmb := range pres.ComboBoxes {
				elem := &ComboBoxPresentationElement{
					BasePresentationElement: BasePresentationElement{
						ID:          cmb.RefID,
						ElementType: "comboBox",
					},
					NoSort:      cmb.NoSort == "true",
					Label:       cmb.Label,
					DefaultText: cmb.Default,
					Suggestions: cmb.Suggestions,
				}
				presentation.Elements = append(presentation.Elements, elem)
			}

			// DropdownList elements
			for _, ddl := range pres.DropdownLists {
				elem := &DropDownPresentationElement{
					BasePresentationElement: BasePresentationElement{
						ID:          ddl.RefID,
						ElementType: "dropdownList",
					},
					NoSort: ddl.NoSort == "true",
					Label:  ddl.Text,
				}
				if ddl.DefaultItem != "" {
					item, _ := strconv.Atoi(ddl.DefaultItem)
					elem.DefaultItemID = &item
				}
				presentation.Elements = append(presentation.Elements, elem)
			}

			// ListBox elements
			for _, lb := range pres.ListBoxes {
				elem := &ListPresentationElement{
					BasePresentationElement: BasePresentationElement{
						ID:          lb.RefID,
						ElementType: "listBox",
					},
					Label: lb.Text,
				}
				presentation.Elements = append(presentation.Elements, elem)
			}

			// MultiTextBox elements
			for _, mtb := range pres.MultiTextBoxes {
				elem := &MultiTextPresentationElement{
					BasePresentationElement: BasePresentationElement{
						ID:          mtb.RefID,
						ElementType: "multiTextBox",
					},
					Label: mtb.Text,
				}
				presentation.Elements = append(presentation.Elements, elem)
			}

			adml.PresentationTable[presentation.Name] = presentation
		}
	}

	return adml, nil
}
