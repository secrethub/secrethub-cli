package secrethub

import (
	"encoding/json"
	"fmt"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"io"
	"strings"
)

type listFormatter interface {
	Write([]string) error
}

func newLineFormatter(writer io.Writer) lineFormatter {
	return lineFormatter{writer: writer}
}

// lineFormatter returns a formatter that formats the given table into lines of text with unaligned columns.
type lineFormatter struct {
	writer io.Writer
}

// Write writes a the given row entries separated by '\t' characters.
func (l lineFormatter) Write(line []string) error {
	_, err := l.writer.Write([]byte(strings.Join(line, "\t") + "\n"))
	return err
}

// newJSONFormatter returns a table formatter that formats the given table rows as json.
func newJSONFormatter(writer io.Writer, fieldNames []string) *jsonFormatter {
	for i := range fieldNames {
		fieldNames[i] = toPascalCase(fieldNames[i])
	}
	return &jsonFormatter{
		encoder: json.NewEncoder(writer),
		fields:  fieldNames,
	}
}

func toPascalCase(s string) string {
	caser := cases.Title(language.English)
	return strings.ReplaceAll(caser.String(s), " ", "")
}

type jsonFormatter struct {
	encoder *json.Encoder
	fields  []string
}

// Write writes the json representation of the given row
// with the configured field names as keys and the provided values
func (f *jsonFormatter) Write(values []string) error {
	if len(f.fields) != len(values) {
		return fmt.Errorf("unexpected number of json fields")
	}

	jsonMap := make(map[string]string)
	for i, element := range values {
		jsonMap[f.fields[i]] = element
	}

	return f.encoder.Encode(jsonMap)
}

// newTableFormatter returns a list formatter that formats entries in a table.
func newTableFormatter(writer io.Writer, tableWidth int, columns []tableColumn) *tableFormatter {
	return &tableFormatter{
		writer:     writer,
		tableWidth: tableWidth,
		columns:    columns,
	}
}

type tableFormatter struct {
	tableWidth           int
	writer               io.Writer
	computedColumnWidths []int
	columns              []tableColumn
	headerPrinted        bool
}

// Write writes the given values formatted in a table with the configured column widths and names.
// The header of the table is printed on the first call, before any other value.
func (f *tableFormatter) Write(values []string) error {
	if !f.headerPrinted {
		header := make([]string, len(f.columns))
		for i, col := range f.columns {
			header[i] = strings.ToUpper(col.name)
		}
		formattedHeader := f.formatRow(header)
		_, err := f.writer.Write(formattedHeader)
		if err != nil {
			return err
		}
		f.headerPrinted = true
	}

	formattedRow := f.formatRow(values)
	_, err := f.writer.Write(formattedRow)
	return err
}

// formatRow formats the given table row to fit the configured width by
// giving each cell an equal width and wrapping the text in cells that exceed it.
func (f *tableFormatter) formatRow(row []string) []byte {
	columnWidths := f.columnWidths()
	grid := f.fitToColumns(row, columnWidths)

	strRes := strings.Builder{}
	for _, row := range grid {
		strRes.WriteString(strings.Join(row, "  ") + "\n")
	}
	return []byte(strRes.String())
}

// fitToColumns returns a the given row split over a matrix in which all columns have equal length.
// Longer values are split over multiple cells and shorter (or empty) ones are padded with " ".
func (f *tableFormatter) fitToColumns(cells []string, columnWidths []int) [][]string {
	maxLinesPerCell := f.lineCount(cells, columnWidths)

	grid := make([][]string, maxLinesPerCell)
	for i := 0; i < maxLinesPerCell; i++ {
		grid[i] = make([]string, len(cells))
	}

	for i, cell := range cells {
		columnWidth := columnWidths[i]
		lineCount := len(cell) / columnWidth
		for j := 0; j < lineCount; j++ {
			begin := j * columnWidth
			end := (j + 1) * columnWidth
			grid[j][i] = cell[begin:end]
		}

		charactersLeft := len(cell) % columnWidth
		if charactersLeft != 0 {
			grid[lineCount][i] = cell[len(cell)-charactersLeft:] + strings.Repeat(" ", columnWidth-charactersLeft)
		} else if lineCount < maxLinesPerCell {
			grid[lineCount][i] = strings.Repeat(" ", columnWidth)
		}

		for j := lineCount + 1; j < maxLinesPerCell; j++ {
			grid[j][i] = strings.Repeat(" ", columnWidth)
		}
	}

	return grid
}

// lineCount returns the number of lines the given table row will occupy after splitting the
// cell values that exceed their column width.
func (f *tableFormatter) lineCount(row []string, widths []int) int {
	maxLinesPerCell := 1
	for i, value := range row {
		lines := len(value) / widths[i]
		if len(value)%widths[i] != 0 {
			lines++
		}
		if lines > maxLinesPerCell {
			maxLinesPerCell = lines
		}
	}
	return maxLinesPerCell
}

// columnWidths returns the width of each column based on their maximum widths
// and the table width.
func (f *tableFormatter) columnWidths() []int {
	if f.computedColumnWidths != nil {
		return f.computedColumnWidths
	}
	adjustedWidths := make([]int, len(f.columns))

	// Distribute the table width equally between all columns and leave a margin of 2 characters between them.
	columnsLeft := len(f.columns)
	widthLeft := f.tableWidth - 2*(len(f.columns)-1)
	widthPerColumn := widthLeft / columnsLeft
	adjusted := true
	for adjusted {
		adjusted = false
		for i, col := range f.columns {
			// fix columns that have a smaller maximum width than the current width/column and have not been fixed yet.
			if adjustedWidths[i] == 0 && col.maxWidth != 0 && col.maxWidth < widthPerColumn {
				adjustedWidths[i] = col.maxWidth
				widthLeft -= col.maxWidth
				columnsLeft--
				adjusted = true
			}
		}
		// If all columns are fixed to their max width, distribute the remaining width equally between all of them.
		if columnsLeft == 0 {
			for i := range adjustedWidths {
				adjustedWidths[i] += widthLeft / len(adjustedWidths)
			}
			break
		}
		// Recalculate the width/column for the remaining unadjusted columns.
		widthPerColumn = widthLeft / columnsLeft
	}

	// distribute the remaining width equally between columns with no maximum width.
	for i := range adjustedWidths {
		if adjustedWidths[i] == 0 {
			adjustedWidths[i] = widthPerColumn
		}
	}
	f.computedColumnWidths = adjustedWidths
	return adjustedWidths
}
