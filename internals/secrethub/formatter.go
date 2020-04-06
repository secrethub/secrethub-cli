package secrethub

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

type listFormatter interface {
	Write([]string) error
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
	return strings.ReplaceAll(strings.Title(s), " ", "")
}

type jsonFormatter struct {
	encoder *json.Encoder
	fields  []string
}

// formatRow returns the json representation of the given row
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
	grid := f.rowToGrid(row, columnWidths)

	strRes := strings.Builder{}
	for _, row := range grid {
		strRes.WriteString(strings.Join(row, "  ") + "\n")
	}
	return []byte(strRes.String())
}

// rowToGrid returns a the given row split over a matrix in which all columns have equal length.
// Longer values are split over multiple cells and shorter (or empty) ones are padded with " ".
func (f *tableFormatter) rowToGrid(row []string, columnWidths []int) [][]string {
	maxLinesPerCell := f.lineCount(row, columnWidths)

	grid := make([][]string, maxLinesPerCell)
	for i := 0; i < maxLinesPerCell; i++ {
		grid[i] = make([]string, len(row))
	}

	for i, cell := range row {
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
	res := make([]int, len(f.columns))

	// Distribute the maximum width equally between all columns and repeatedly
	// check if any of them have a smaller maximum width and can be shrunk.
	// Stop when no columns can be further adjusted.
	adjusted := true
	columnsLeft := len(f.columns)
	widthLeft := f.tableWidth - 2*(len(f.columns)-1)
	widthPerColumn := widthLeft / columnsLeft
	for adjusted {
		adjusted = false
		for i, col := range f.columns {
			if res[i] == 0 && col.maxWidth != 0 && col.maxWidth < widthPerColumn {
				res[i] = col.maxWidth
				widthLeft -= col.maxWidth
				columnsLeft--
				adjusted = true
			}
		}
		if columnsLeft == 0 {
			for i := range res {
				res[i] += widthLeft / len(res)
			}
			break
		}
		widthPerColumn = widthLeft / columnsLeft
	}

	// distribute the remaining width equally between columns with no maximum width.
	for i := range res {
		if res[i] == 0 {
			res[i] = widthPerColumn
		}
	}
	f.computedColumnWidths = res
	return res
}
