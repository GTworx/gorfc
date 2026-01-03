package main

import (
	"fmt"
	"log"

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
	// Configure your SAP connection parameters
	config := SAPConfig{
		Dest:     "KoleksiyonIDES",
		Client:   "800",       // Your SAP client number
		User:     "SOOEZY",    // Your SAP username
		Password: "SAPSooezy", // Your SAP password
		Lang:     "EN",
		Ashost:   "s185.126.178.110", // SAP application server host
		Sysnr:    "00",               // System number
		// Saprouter: "/H/saprouter.example.com/S/3299/H/", // Uncomment if using SAP Router
	}

	// Connect to SAP
	c, err := connectToSAP(config)
	if err != nil {
		log.Fatalf("Failed to connect to SAP: %v", err)
	}
	defer c.Close()

	// Read T000 table
	tableName := "T000"
	fields := []string{"MANDT", "MTEXT", "ORT01", "CCCATEGORY"} // Specify fields you want to fetch

	data, err := readTable(c, tableName, fields, "", 0)
	if err != nil {
		log.Fatalf("Failed to read table: %v", err)
	}

	// Display results
	fmt.Printf("\nData from table %s:\n", tableName)
	fmt.Println("========================================")
	for i, row := range data {
		fmt.Printf("Row %d: %v\n", i+1, row)
	}
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

func readTable(c *gorfc.Connection, tableName string, fields []string, whereClause string, rowCount int) ([]map[string]interface{}, error) {
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
	data, err := parseRFCReadTableResult(result, fields)
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	return data, nil
}

func parseRFCReadTableResult(result map[string]interface{}, fields []string) ([]map[string]interface{}, error) {
	var parsedData []map[string]interface{}

	// Get the DATA table from result
	dataTable, ok := result["DATA"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid DATA format in result")
	}

	// Get field information for parsing positions
	fieldsInfo, ok := result["FIELDS"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid FIELDS format in result")
	}

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
		offset, _ := fieldMap["OFFSET"].(int)
		length, _ := fieldMap["LENGTH"].(int)

		fieldOffsets[fieldName] = struct {
			offset int
			length int
		}{offset, length}
	}

	// Parse each row
	for _, row := range dataTable {
		rowMap, ok := row.(map[string]interface{})
		if !ok {
			continue
		}

		wa, ok := rowMap["WA"].(string)
		if !ok {
			continue
		}

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
					parsedRow[fieldName] = trimString(value)
				} else {
					parsedRow[fieldName] = ""
				}
			}
		}

		parsedData = append(parsedData, parsedRow)
	}

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
