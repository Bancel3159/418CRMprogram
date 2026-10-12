package service

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"

	"crm-backend/internal/model"
)

// ParsedRow 解析后的一行数据
type ParsedRow struct {
	OrderNo    string
	PlayerID   string
	Amount     float64
	Currency   string
	Status     model.OrderStatus
	OrderTime  time.Time
	PaidAt     *time.Time
	Remark     string
	RawExtras  map[string]interface{} // 未映射列原值(写入 raw_payload.extras)
	RowNumber  int                   // Excel/CSV 行号(用于错误报告)
}

// ParseResult 解析结果
type ParseResult struct {
	Rows   []ParsedRow
	Errors []ParseError
}

// ParseError 单行错误
type ParseError struct {
	RowNumber int
	Field     string
	Message   string
}

// ExcelParser Excel/CSV 解析器(配合 UploadTemplate 使用)
type ExcelParser struct{}

// NewExcelParser 创建解析器
func NewExcelParser() *ExcelParser {
	return &ExcelParser{}
}

// Parse 解析上传文件,返回结构化行
//
// 文件类型自动识别(.csv / .xlsx / .xls)
// 第一行必须为表头
func (p *ExcelParser) Parse(fileBytes []byte, filename string, tpl *model.UploadTemplate) (*ParseResult, error) {
	var rows [][]string
	var errRead error

	if strings.HasSuffix(strings.ToLower(filename), ".csv") {
		rows, errRead = p.parseCSV(fileBytes)
	} else {
		rows, errRead = p.parseExcel(fileBytes)
	}
	if errRead != nil {
		return nil, fmt.Errorf("文件读取失败: %w", errRead)
	}

	if len(rows) < 2 {
		return nil, errors.New("文件至少需要表头+1行数据")
	}

	// 解析表头 → 列索引映射
	header := rows[0]
	colIdx, err := p.buildColumnIndex(header, tpl)
	if err != nil {
		return nil, err
	}

	result := &ParseResult{
		Rows:   make([]ParsedRow, 0, len(rows)-1),
		Errors: make([]ParseError, 0),
	}

	timeFormat := tpl.TimeFormat
	if timeFormat == "" {
		timeFormat = "2006-01-02 15:04:05"
	}
	defaultCurrency := tpl.Currency
	if defaultCurrency == "" {
		defaultCurrency = "CNY"
	}
	statusMapping := map[string]string{}
	for k, v := range tpl.StatusMapping {
		if s, ok := v.(string); ok {
			statusMapping[k] = s
		}
	}

	for rowIdx := 1; rowIdx < len(rows); rowIdx++ {
		row := rows[rowIdx]
		if p.isEmptyRow(row) {
			continue
		}

		parsed, perr := p.parseRow(row, colIdx, tpl.ColumnMappings, timeFormat, defaultCurrency, statusMapping, rowIdx+1)
		if perr != nil {
			result.Errors = append(result.Errors, *perr)
			continue
		}
		if parsed != nil {
			result.Rows = append(result.Rows, *parsed)
		}
	}

	return result, nil
}

// parseCSV 解析 CSV
func (p *ExcelParser) parseCSV(data []byte) ([][]string, error) {
	// 去除 UTF-8 BOM
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		data = data[3:]
	}
	r := csv.NewReader(bytes.NewReader(data))
	r.FieldsPerRecord = -1
	r.TrimLeadingSpace = true
	var rows [][]string
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		rows = append(rows, rec)
	}
	return rows, nil
}

// parseExcel 解析 Excel
func (p *ExcelParser) parseExcel(data []byte) ([][]string, error) {
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// 取第一个 sheet
	sheetName := f.GetSheetName(0)
	if sheetName == "" {
		return nil, errors.New("Excel 中没有可用的工作表")
	}
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// buildColumnIndex 根据模板映射,建立 "系统字段 → 列索引" 表
func (p *ExcelParser) buildColumnIndex(header []string, tpl *model.UploadTemplate) (map[string]int, error) {
	// header → idx
	headerMap := map[string]int{}
	for i, h := range header {
		headerMap[strings.TrimSpace(h)] = i
	}

	// 系统字段 → 表头列名
	colIdx := map[string]int{}
	for systemField, columnName := range tpl.ColumnMappings {
		cn, ok := columnName.(string)
		if !ok {
			continue
		}
		idx, found := headerMap[cn]
		if !found {
			return nil, fmt.Errorf("模板映射的列 [%s] 在文件表头中未找到", cn)
		}
		colIdx[systemField] = idx
	}

	// 必填字段校验
	for _, required := range []string{"order_no", "player_id", "amount"} {
		if _, ok := colIdx[required]; !ok {
			return nil, fmt.Errorf("缺少必填字段映射: %s", required)
		}
	}
	return colIdx, nil
}

// parseRow 解析单行
func (p *ExcelParser) parseRow(
	row []string,
	colIdx map[string]int,
	mappings model.JSONB,
	timeFormat string,
	defaultCurrency string,
	statusMapping map[string]string,
	rowNum int,
) (*ParsedRow, *ParseError) {
	get := func(field string) string {
		idx, ok := colIdx[field]
		if !ok || idx >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[idx])
	}

	orderNo := get("order_no")
	if orderNo == "" {
		return nil, &ParseError{RowNumber: rowNum, Field: "order_no", Message: "订单号为空"}
	}
	playerID := get("player_id")
	if playerID == "" {
		return nil, &ParseError{RowNumber: rowNum, Field: "player_id", Message: "玩家ID为空"}
	}

	amountStr := get("amount")
	if amountStr == "" {
		return nil, &ParseError{RowNumber: rowNum, Field: "amount", Message: "金额为空"}
	}
	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		return nil, &ParseError{RowNumber: rowNum, Field: "amount", Message: "金额格式错误: " + amountStr}
	}

	orderTimeStr := get("order_time")
	if orderTimeStr == "" {
		return nil, &ParseError{RowNumber: rowNum, Field: "order_time", Message: "下单时间为空"}
	}
	orderTime, err := parseTime(orderTimeStr, timeFormat)
	if err != nil {
		return nil, &ParseError{RowNumber: rowNum, Field: "order_time", Message: "下单时间解析失败: " + err.Error()}
	}

	var paidAt *time.Time
	if paidStr := get("paid_at"); paidStr != "" {
		t, err := parseTime(paidStr, timeFormat)
		if err == nil {
			paidAt = &t
		}
	}

	statusStr := get("status")
	status, err := resolveStatus(statusStr, statusMapping)
	if err != nil {
		return nil, &ParseError{RowNumber: rowNum, Field: "status", Message: err.Error()}
	}

	currency := get("currency")
	if currency == "" {
		currency = defaultCurrency
	}

	// 收集未映射列
	extras := map[string]interface{}{}
	for systemField, columnNameVal := range mappings {
		cn, ok := columnNameVal.(string)
		if !ok {
			continue
		}
		// 已映射的核心字段不算 extras
		if _, isCore := colIdx[systemField]; isCore {
			continue
		}
		for i, h := range row {
			if strings.TrimSpace(h) == cn && i < len(row) {
				extras[cn] = row[i]
				break
			}
		}
	}

	return &ParsedRow{
		OrderNo:   orderNo,
		PlayerID:  playerID,
		Amount:    amount,
		Currency:  currency,
		Status:    status,
		OrderTime: orderTime,
		PaidAt:    paidAt,
		Remark:    get("remark"),
		RawExtras: extras,
		RowNumber: rowNum,
	}, nil
}

// resolveStatus 解析订单状态(支持中文映射)
func resolveStatus(raw string, mapping map[string]string) (model.OrderStatus, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return model.OrderStatusPaid, nil // 默认 paid
	}
	if mapped, ok := mapping[raw]; ok {
		return model.OrderStatus(mapped), validateStatus(mapped)
	}
	// 直接是英文枚举
	s := model.OrderStatus(raw)
	if s.IsValid() {
		return s, nil
	}
	return "", fmt.Errorf("无法识别的状态: %s", raw)
}

func validateStatus(s string) error {
	if !model.OrderStatus(s).IsValid() {
		return fmt.Errorf("状态值非法: %s", s)
	}
	return nil
}

// parseTime 解析时间(支持多种常见格式)
func parseTime(s string, primaryFormat string) (time.Time, error) {
	s = strings.TrimSpace(s)
	formats := []string{
		primaryFormat,
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02 15:04",
		"2006-01-02",
		"2006/01/02 15:04:05",
		"2006/01/02",
		"01/02/2006 15:04:05",
		"01/02/2006",
	}
	for _, f := range formats {
		if f == "" {
			continue
		}
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("无法解析时间: %s", s)
}

func (p *ExcelParser) isEmptyRow(row []string) bool {
	for _, c := range row {
		if strings.TrimSpace(c) != "" {
			return false
		}
	}
	return true
}