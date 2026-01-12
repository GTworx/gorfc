package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/sap/gorfc/gorfc"
)

type SAPConfig struct {
	Dest      string
	Client    string
	User      string
	Password  string
	Lang      string
	Ashost    string
	Sysnr     string
	Saprouter string // Optional
}

func main() {
	// Setup log file
	logFile, err := os.OpenFile("sap_rfc_reader.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	defer logFile.Close()

	// Create multi-writer to write to both file and stdout
	multiWriter := io.MultiWriter(os.Stdout, logFile)

	// Log session start
	logMessage := fmt.Sprintf("\n========== New Session: %s ==========\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprint(multiWriter, logMessage)

	// Configure your SAP connection parameters
	config := SAPConfig{
		Dest:     "",
		Client:   "100", // Your SAP client number
		User:     "",    // Your SAP username
		Password: "",    // Your SAP password
		Lang:     "EN",
		Ashost:   "",   // SAP application server host
		Sysnr:    "00", // System number
		// Saprouter: "/H/saprouter.example.com/S/3299/H/", // Uncomment if using SAP Router
	}

	// Connect to SAP
	c, err := connectToSAP(config)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to connect to SAP: %v\n", err)
		fmt.Fprint(multiWriter, errMsg)
		log.Fatal(errMsg)
	}
	defer c.Close()

	fmt.Fprint(multiWriter, "✓ Successfully connected to SAP\n")

	// Read T000 table
	tableName := "T000"
	fields := []string{"MANDT", "MTEXT", "ORT01", "CCCATEGORY"} // Specify fields you want to fetch

	data, err := readTable(c, tableName, fields, "", 0, multiWriter)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to read table: %v\n", err)
		fmt.Fprint(multiWriter, errMsg)
		log.Fatal(errMsg)
	}

	// Display results
	displayResults(tableName, fields, data, multiWriter)

	// Log session end
	endMessage := fmt.Sprintf("\n========== Session End: %s ==========\n\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprint(multiWriter, endMessage)
}

func connectToSAP(config SAPConfig) (*gorfc.Connection, error) {
	// Build connection parameters
	params := map[string]string{
		"dest":   config.Dest,
		"client": config.Client,
		"user":   config.User,
		"passwd": config.Password,
		"lang":   config.Lang,
		"ashost": config.Ashost,
		"sysnr":  config.Sysnr,
	}

	// Add SAP Router if configured
	if config.Saprouter != "" {
		params["saprouter"] = config.Saprouter
	}

	// Create connection
	c, err := gorfc.ConnectionFromParams(params)
	if err != nil {
		return nil, fmt.Errorf("connection error: %w", err)
	}

	return c, nil
}

func readTable(c *gorfc.Connection, tableName string, fields []string, whereClause string, rowCount int, writer io.Writer) ([]map[string]interface{}, error) {
	// Prepare FIELDS parameter
	var fieldsList []map[string]interface{}
	for _, field := range fields {
		fieldsList = append(fieldsList, map[string]interface{}{
			"FIELDNAME": field,
		})
	}

	// Prepare parameters for RFC_READ_TABLE
	params := map[string]interface{}{
		"QUERY_TABLE": tableName,
		"DELIMITER":   "|",
		"FIELDS":      fieldsList,
	}

	// Add WHERE clause if provided
	if whereClause != "" {
		params["OPTIONS"] = []map[string]interface{}{
			{"TEXT": whereClause},
		}
	}

	// Add ROWCOUNT if specified
	if rowCount > 0 {
		params["ROWCOUNT"] = rowCount
	}

	// Call RFC_READ_TABLE
	result, err := c.Call("RFC_READ_TABLE", params)
	if err != nil {
		return nil, fmt.Errorf("RFC call error: %w", err)
	}

	// Parse the results
	data, err := parseRFCReadTableResult(result, fields, writer)
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	return data, nil
}

func parseRFCReadTableResult(result map[string]interface{}, fields []string, writer io.Writer) ([]map[string]interface{}, error) {
	var parsedData []map[string]interface{}

	// Debug: Print raw result
	fmt.Fprintf(writer, "\n[DEBUG] Raw RFC result keys: %v\n", getKeys(result))

	// Get the DATA table from result
	dataTable, ok := result["DATA"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid DATA format in result")
	}
	fmt.Fprintf(writer, "[DEBUG] DATA table has %d rows\n", len(dataTable))

	// Get field information for parsing positions
	fieldsInfo, ok := result["FIELDS"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid FIELDS format in result")
	}
	fmt.Fprintf(writer, "[DEBUG] FIELDS info has %d field definitions\n", len(fieldsInfo))

	// Build field offset map
	fieldOffsets := make(map[string]struct {
		offset int
		length int
	})

	for _, fieldInfo := range fieldsInfo {
		fieldMap, ok := fieldInfo.(map[string]interface{})
		if !ok {
			continue
		}

		fieldName, _ := fieldMap["FIELDNAME"].(string)

		// Try to get offset - it might be int or string
		var offset, length int
		switch v := fieldMap["OFFSET"].(type) {
		case int:
			offset = v
		case string:
			fmt.Sscanf(v, "%d", &offset)
		}

		// Try to get length - it might be int or string
		switch v := fieldMap["LENGTH"].(type) {
		case int:
			length = v
		case string:
			fmt.Sscanf(v, "%d", &length)
		}

		fieldOffsets[fieldName] = struct {
			offset int
			length int
		}{offset, length}
		fmt.Fprintf(writer, "[DEBUG] Field: %s, Offset: %d, Length: %d\n", fieldName, offset, length)
	}

	// Parse each row
	for rowIdx, row := range dataTable {
		rowMap, ok := row.(map[string]interface{})
		if !ok {
			fmt.Fprintf(writer, "[DEBUG] Row %d: Not a valid map\n", rowIdx)
			continue
		}

		wa, ok := rowMap["WA"].(string)
		if !ok {
			fmt.Fprintf(writer, "[DEBUG] Row %d: WA field not found or not a string\n", rowIdx)
			continue
		}

		fmt.Fprintf(writer, "[DEBUG] Row %d WA content (length %d): '%s'\n", rowIdx, len(wa), wa)

		// Parse fields from the row data
		parsedRow := make(map[string]interface{})
		for _, fieldName := range fields {
			if info, exists := fieldOffsets[fieldName]; exists {
				// Extract field value based on offset and length
				start := info.offset
				end := start + info.length
				if end > len(wa) {
					end = len(wa)
				}
				if start < len(wa) {
					value := wa[start:end]
					// Trim spaces
					trimmedValue := trimString(value)
					parsedRow[fieldName] = trimmedValue
					fmt.Fprintf(writer, "[DEBUG]   Field '%s': '%s'\n", fieldName, trimmedValue)
				} else {
					parsedRow[fieldName] = ""
					fmt.Fprintf(writer, "[DEBUG]   Field '%s': (empty - out of bounds)\n", fieldName)
				}
			} else {
				fmt.Fprintf(writer, "[DEBUG]   Field '%s': (not in offset map)\n", fieldName)
			}
		}

		parsedData = append(parsedData, parsedRow)
	}

	fmt.Fprintf(writer, "[DEBUG] Total parsed records: %d\n\n", len(parsedData))
	return parsedData, nil
}

func trimString(s string) string {
	// Simple trim function to remove leading/trailing spaces
	start := 0
	end := len(s)

	for start < end && s[start] == ' ' {
		start++
	}

	for end > start && s[end-1] == ' ' {
		end--
	}

	return s[start:end]
}

func getKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func displayResults(tableName string, fields []string, data []map[string]interface{}, writer io.Writer) {
	fmt.Fprintf(writer, "\n╔══════════════════════════════════════════════════════════════╗\n")
	fmt.Fprintf(writer, "║  Table: %-50s ║\n", tableName)
	fmt.Fprintf(writer, "║  Total Records: %-42d ║\n", len(data))
	fmt.Fprintf(writer, "╚══════════════════════════════════════════════════════════════╝\n\n")

	if len(data) == 0 {
		fmt.Fprintln(writer, "No data found.")
		return
	}

	// Calculate column widths
	colWidths := make(map[string]int)
	for _, field := range fields {
		colWidths[field] = len(field)
	}

	// Find max width for each column
	for _, row := range data {
		for _, field := range fields {
			if val, ok := row[field]; ok {
				valStr := fmt.Sprintf("%v", val)
				if len(valStr) > colWidths[field] {
					colWidths[field] = len(valStr)
				}
			}
		}
	}

	// Limit column width to 40 characters for readability
	for field := range colWidths {
		if colWidths[field] > 40 {
			colWidths[field] = 40
		}
		// Minimum width of 10
		if colWidths[field] < 10 {
			colWidths[field] = 10
		}
	}

	// Print header
	fmt.Fprint(writer, "│ ")
	for i, field := range fields {
		fmt.Fprintf(writer, "%-*s", colWidths[field], field)
		if i < len(fields)-1 {
			fmt.Fprint(writer, " │ ")
		}
	}
	fmt.Fprintln(writer, " │")

	// Print separator
	fmt.Fprint(writer, "├─")
	for i, field := range fields {
		fmt.Fprint(writer, strings.Repeat("─", colWidths[field]))
		if i < len(fields)-1 {
			fmt.Fprint(writer, "─┼─")
		}
	}
	fmt.Fprintln(writer, "─┤")

	// Print data rows
	for rowNum, row := range data {
		fmt.Fprint(writer, "│ ")
		for i, field := range fields {
			var valStr string
			if val, ok := row[field]; ok {
				valStr = fmt.Sprintf("%v", val)
				// Truncate if too long
				if len(valStr) > colWidths[field] {
					valStr = valStr[:colWidths[field]-3] + "..."
				}
			}
			fmt.Fprintf(writer, "%-*s", colWidths[field], valStr)
			if i < len(fields)-1 {
				fmt.Fprint(writer, " │ ")
			}
		}
		fmt.Fprintf(writer, " │ (Row %d)\n", rowNum+1)
	}

	// Print footer
	fmt.Fprint(writer, "└─")
	for i, field := range fields {
		fmt.Fprint(writer, strings.Repeat("─", colWidths[field]))
		if i < len(fields)-1 {
			fmt.Fprint(writer, "─┴─")
		}
	}
	fmt.Fprintln(writer, "─┘")

	fmt.Fprintf(writer, "\n✓ Displayed %d record(s)\n", len(data))
}
