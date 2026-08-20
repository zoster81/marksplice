package splice

import "fmt"

// Section is a derived source-bound view governed by one supported document heading.
type Section struct {
	HeadingID       NodeID
	Level           int
	Range           Range
	BodyRange       Range
	ParentHeadingID NodeID
	HasParent       bool
}

func buildSections(source []byte, nodes []Node) ([]Section, map[NodeID]int, error) {
	headings := make([]Node, 0)
	for _, node := range nodes {
		if node.Kind != KindHeading || !node.Editable || !node.TopLevel {
			continue
		}
		if !node.Range.Valid(len(source)) || node.Range.Start == node.Range.End || node.Level < 1 || node.Level > 6 {
			return nil, nil, fmt.Errorf("invalid section heading %q: level %d range [%d,%d)", node.ID, node.Level, node.Range.Start, node.Range.End)
		}
		if len(headings) != 0 && node.Range.Start <= headings[len(headings)-1].Range.Start {
			previous := headings[len(headings)-1]
			return nil, nil, fmt.Errorf("section headings out of source order: %q at %d after %q at %d", node.ID, node.Range.Start, previous.ID, previous.Range.Start)
		}
		headings = append(headings, node)
	}

	sections := make([]Section, len(headings))
	index := make(map[NodeID]int, len(headings))
	stack := make([]int, 0, len(headings))
	for i, heading := range headings {
		bodyStart, err := sectionBodyStart(source, heading.Range.End)
		if err != nil {
			return nil, nil, fmt.Errorf("section heading %q: %w", heading.ID, err)
		}
		bodyEnd := len(source)
		if i+1 < len(headings) {
			bodyEnd = headings[i+1].Range.Start
		}
		if bodyStart > bodyEnd {
			return nil, nil, fmt.Errorf("section heading %q body starts at %d after next heading at %d", heading.ID, bodyStart, bodyEnd)
		}

		for len(stack) != 0 && sections[stack[len(stack)-1]].Level >= heading.Level {
			open := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			sections[open].Range.End = heading.Range.Start
		}

		section := Section{
			HeadingID: heading.ID,
			Level:     heading.Level,
			Range:     Range{Start: heading.Range.Start, End: len(source)},
			BodyRange: Range{Start: bodyStart, End: bodyEnd},
		}
		if len(stack) != 0 {
			section.ParentHeadingID = sections[stack[len(stack)-1]].HeadingID
			section.HasParent = true
		}
		if _, exists := index[heading.ID]; exists {
			return nil, nil, fmt.Errorf("duplicate section heading ID %q", heading.ID)
		}
		sections[i] = section
		index[heading.ID] = i
		stack = append(stack, i)
	}
	return sections, index, nil
}

func sectionBodyStart(source []byte, headingEnd int) (int, error) {
	if headingEnd < 0 || headingEnd > len(source) {
		return 0, fmt.Errorf("heading end %d is outside source length %d", headingEnd, len(source))
	}
	if headingEnd == len(source) {
		return headingEnd, nil
	}
	switch source[headingEnd] {
	case '\n':
		return headingEnd + 1, nil
	case '\r':
		if headingEnd+1 < len(source) && source[headingEnd+1] == '\n' {
			return headingEnd + 2, nil
		}
		return headingEnd + 1, nil
	default:
		return 0, fmt.Errorf("heading range ends before non-line-ending byte 0x%02x at %d", source[headingEnd], headingEnd)
	}
}

// SectionCount returns the number of derived document sections.
func (d *Document) SectionCount() int {
	if d == nil {
		return 0
	}
	return len(d.sections)
}

// SectionAt returns one derived section by source order.
func (d *Document) SectionAt(index int) (Section, bool) {
	if d == nil || index < 0 || index >= len(d.sections) {
		return Section{}, false
	}
	return d.sections[index], true
}

// SectionByHeadingID returns the section governed by one heading node ID.
func (d *Document) SectionByHeadingID(id NodeID) (Section, bool) {
	if d == nil {
		return Section{}, false
	}
	index, ok := d.sectionIndex[id]
	if !ok || index < 0 || index >= len(d.sections) {
		return Section{}, false
	}
	return d.sections[index], true
}
