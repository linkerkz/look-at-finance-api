package ofd

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"
)

var (
	ErrInvalidURL      = errors.New("invalid OFD URL format")
	ErrReceiptNotFound = errors.New("receipt not found on OFD portal")
	ErrParsingFailed   = errors.New("failed to parse receipt data")
	ErrFetchFailed     = errors.New("failed to fetch receipt from OFD")
)

type ParsedReceipt struct {
	FiscalID     string
	StoreName    string
	StoreAddress string
	PurchaseDate time.Time
	TotalAmount  float64
	Items        []ParsedItem
	RawURL       string
}

type ParsedItem struct {
	Name       string
	Quantity   float64
	Price      float64
	TotalPrice float64
	Unit       string
}

// OOFD API response structs
type oofdResponse struct {
	Ticket             oofdTicket        `json:"ticket"`
	OrgTitle           string            `json:"orgTitle"`
	RetailPlaceAddress string            `json:"retailPlaceAddress"`
	MeasureUnits       map[string]string `json:"measureUnits"`
}

type oofdTicket struct {
	TotalSum        float64    `json:"totalSum"`
	TransactionDate string     `json:"transactionDate"`
	FiscalId        string     `json:"fiscalId"`
	Items           []oofdItem `json:"items"`
}

type oofdItem struct {
	ItemType  int            `json:"itemType"`
	Commodity *oofdCommodity `json:"commodity,omitempty"`
}

type oofdCommodity struct {
	Name            string  `json:"name"`
	Quantity        float64 `json:"quantity"`
	Price           float64 `json:"price"`
	Sum             float64 `json:"sum"`
	MeasureUnitCode string  `json:"measureUnitCode"`
}

type Parser struct {
	client *http.Client
}

func NewParser() *Parser {
	// Some KZ OFD portals use certificates that may not be in the system trust store
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, //nolint:gosec // Required for KZ OFD portals
		},
	}

	return &Parser{
		client: &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		},
	}
}

// ValidateURL checks if the URL is a valid KZ OFD URL
func (p *Parser) ValidateURL(url string) (string, error) {
	// Common KZ OFD portals with path-based IDs
	patterns := []string{
		`https?://ofd\.kz/check/([a-zA-Z0-9]+)`,
		`https?://consumer\.oofd\.kz/ticket/([a-zA-Z0-9-]+)`,
		`https?://web\.fiscalcheck\.kz/([a-zA-Z0-9]+)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(url); len(matches) > 1 {
			return matches[1], nil
		}
	}

	// Check for consumer.oofd.kz with query parameters format
	// Example: https://consumer.oofd.kz/?i=2132898740&f=010100707927&s=9852.00&t=20251213T162905
	oofdQueryPattern := regexp.MustCompile(`https?://consumer\.oofd\.kz/\?.*[if]=(\d+)`)
	if matches := oofdQueryPattern.FindStringSubmatch(url); len(matches) > 1 {
		// Extract fiscal ID from query params
		fiscalID := p.extractQueryParam(url, "f")
		if fiscalID == "" {
			fiscalID = p.extractQueryParam(url, "i")
		}
		if fiscalID != "" {
			return fiscalID, nil
		}
	}

	return "", ErrInvalidURL
}

// extractQueryParam extracts a query parameter value from a URL
func (p *Parser) extractQueryParam(rawURL string, param string) string {
	pattern := regexp.MustCompile(param + `=([^&]+)`)
	if matches := pattern.FindStringSubmatch(rawURL); len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// Parse fetches and parses a receipt from the OFD URL
func (p *Parser) Parse(url string) (*ParsedReceipt, error) {
	fiscalID, err := p.ValidateURL(url)
	if err != nil {
		return nil, err
	}

	// Fetch the receipt page
	resp, err := p.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFetchFailed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrReceiptNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status code %d", ErrFetchFailed, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFetchFailed, err)
	}

	// Try to parse based on the URL pattern
	// Note: check oofd.kz before ofd.kz since "oofd.kz" contains "ofd.kz"
	if strings.Contains(url, "oofd.kz") {
		return p.parseOofdKz(string(body), fiscalID, url)
	} else if strings.Contains(url, "ofd.kz") {
		return p.parseOfdKz(string(body), fiscalID, url)
	} else if strings.Contains(url, "fiscalcheck.kz") {
		return p.parseFiscalCheck(string(body), fiscalID, url)
	}

	// Generic fallback parser
	return p.parseGeneric(string(body), fiscalID, url)
}

// parseOfdKz parses receipts from ofd.kz
func (p *Parser) parseOfdKz(content string, fiscalID string, rawURL string) (*ParsedReceipt, error) {
	doc, err := html.Parse(strings.NewReader(content))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrParsingFailed, err)
	}

	receipt := &ParsedReceipt{
		FiscalID: fiscalID,
		RawURL:   rawURL,
		Items:    []ParsedItem{},
	}

	// Find store info
	receipt.StoreName = p.extractTextByClass(doc, "org-name")
	if receipt.StoreName == "" {
		receipt.StoreName = p.extractTextByClass(doc, "company-name")
	}
	receipt.StoreAddress = p.extractTextByClass(doc, "org-address")
	if receipt.StoreAddress == "" {
		receipt.StoreAddress = p.extractTextByClass(doc, "address")
	}

	// Find date
	dateStr := p.extractTextByClass(doc, "receipt-date")
	if dateStr == "" {
		dateStr = p.extractTextByClass(doc, "date")
	}
	if dateStr != "" {
		receipt.PurchaseDate = p.parseDate(dateStr)
	}
	if receipt.PurchaseDate.IsZero() {
		receipt.PurchaseDate = time.Now()
	}

	// Find total
	totalStr := p.extractTextByClass(doc, "total-sum")
	if totalStr == "" {
		totalStr = p.extractTextByClass(doc, "total")
	}
	receipt.TotalAmount = p.parseAmount(totalStr)

	// Find items
	receipt.Items = p.extractItems(doc)

	// If we couldn't find structured data, try to calculate total from items
	if receipt.TotalAmount == 0 && len(receipt.Items) > 0 {
		for _, item := range receipt.Items {
			receipt.TotalAmount += item.TotalPrice
		}
	}

	// Set default store name if not found
	if receipt.StoreName == "" {
		receipt.StoreName = "Unknown Store"
	}

	return receipt, nil
}

// parseOofdKz parses receipts from consumer.oofd.kz using their JSON API
func (p *Parser) parseOofdKz(content string, fiscalID string, rawURL string) (*ParsedReceipt, error) {
	// Build API URL from query parameters
	// User URL: https://consumer.oofd.kz/?i=2132898740&f=010100707927&s=9852.00&t=20251213T162905
	// API URL:  https://consumer.oofd.kz/api/tickets/get-by-url?t=...&i=...&f=...&s=...
	apiURL := p.buildOofdApiURL(rawURL)
	fmt.Printf("[OOFD DEBUG] API URL: %s\n", apiURL)

	resp, err := p.client.Get(apiURL)
	if err != nil {
		fmt.Printf("[OOFD DEBUG] Fetch error: %v\n", err)
		return nil, fmt.Errorf("%w: %v", ErrFetchFailed, err)
	}
	defer resp.Body.Close()

	fmt.Printf("[OOFD DEBUG] Response status: %d\n", resp.StatusCode)

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrReceiptNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status code %d", ErrFetchFailed, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFetchFailed, err)
	}

	fmt.Printf("[OOFD DEBUG] Response body (first 500 chars): %.500s\n", string(body))

	var oofdResp oofdResponse
	if err := json.Unmarshal(body, &oofdResp); err != nil {
		fmt.Printf("[OOFD DEBUG] JSON unmarshal error: %v\n", err)
		return nil, fmt.Errorf("%w: %v", ErrParsingFailed, err)
	}

	fmt.Printf("[OOFD DEBUG] Parsed - OrgTitle: %s, TotalSum: %.2f, Items: %d\n",
		oofdResp.OrgTitle, oofdResp.Ticket.TotalSum, len(oofdResp.Ticket.Items))

	// Parse transaction date
	purchaseDate, err := time.Parse("2006-01-02T15:04:05.000", oofdResp.Ticket.TransactionDate)
	if err != nil {
		// Try without milliseconds
		purchaseDate, err = time.Parse("2006-01-02T15:04:05", oofdResp.Ticket.TransactionDate)
		if err != nil {
			purchaseDate = time.Now()
		}
	}

	receipt := &ParsedReceipt{
		FiscalID:     fiscalID,
		RawURL:       rawURL,
		StoreName:    oofdResp.OrgTitle,
		StoreAddress: oofdResp.RetailPlaceAddress,
		TotalAmount:  oofdResp.Ticket.TotalSum,
		PurchaseDate: purchaseDate,
		Items:        make([]ParsedItem, 0),
	}

	// Parse items - only itemType == 1 (commodities), skip discounts (itemType == 5)
	for _, item := range oofdResp.Ticket.Items {
		if item.ItemType != 1 || item.Commodity == nil {
			continue
		}

		// Map measure unit code to unit name
		unit := "pcs"
		if oofdResp.MeasureUnits != nil {
			if unitName, ok := oofdResp.MeasureUnits[item.Commodity.MeasureUnitCode]; ok {
				unit = unitName
			}
		}
		// Fallback for common codes
		if unit == "pcs" {
			switch item.Commodity.MeasureUnitCode {
			case "166":
				unit = "kg"
			case "796":
				unit = "pcs"
			}
		}

		receipt.Items = append(receipt.Items, ParsedItem{
			Name:       item.Commodity.Name,
			Quantity:   item.Commodity.Quantity,
			Price:      item.Commodity.Price,
			TotalPrice: item.Commodity.Sum,
			Unit:       unit,
		})
	}

	// Default store name if not found
	if receipt.StoreName == "" {
		receipt.StoreName = "Unknown Store"
	}

	return receipt, nil
}

// buildOofdApiURL converts user-facing URL to API URL
func (p *Parser) buildOofdApiURL(rawURL string) string {
	// Extract query params from user URL
	t := p.extractQueryParam(rawURL, "t")
	i := p.extractQueryParam(rawURL, "i")
	f := p.extractQueryParam(rawURL, "f")
	s := p.extractQueryParam(rawURL, "s")

	// Build API URL
	return fmt.Sprintf("https://consumer.oofd.kz/api/tickets/get-by-url?t=%s&i=%s&f=%s&s=%s", t, i, f, s)
}

// parseOofdTimestamp parses the OOFD timestamp format: 20251213T162905
func (p *Parser) parseOofdTimestamp(s string) time.Time {
	s = strings.TrimSpace(s)

	// Format: 20251213T162905 -> YYYYMMDDTHHMMSS
	formats := []string{
		"20060102T150405",
		"20060102T1504",
		"20060102",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t
		}
	}

	return time.Time{}
}

// extractOofdStoreName tries to extract store name from OOFD-specific HTML patterns
func (p *Parser) extractOofdStoreName(doc *html.Node) string {
	// Look for common patterns in OOFD receipts
	var findStoreName func(*html.Node) string
	findStoreName = func(n *html.Node) string {
		if n.Type == html.ElementNode {
			// Check for header elements that often contain store name
			if n.Data == "h1" || n.Data == "h2" || n.Data == "h3" {
				text := p.extractText(n)
				// Filter out generic titles
				text = strings.TrimSpace(text)
				if text != "" && !strings.Contains(strings.ToLower(text), "чек") &&
					!strings.Contains(strings.ToLower(text), "ticket") {
					return text
				}
			}

			// Look for specific OOFD data attributes or patterns
			for _, attr := range n.Attr {
				if attr.Key == "data-store" || attr.Key == "data-merchant" ||
					attr.Key == "data-seller" || attr.Key == "data-org" {
					return attr.Val
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if result := findStoreName(c); result != "" {
				return result
			}
		}
		return ""
	}

	return findStoreName(doc)
}

// extractOofdItems extracts items from OOFD-specific HTML structure
func (p *Parser) extractOofdItems(doc *html.Node) []ParsedItem {
	var items []ParsedItem

	// Look for table-based item listings common in OOFD
	var findItems func(*html.Node)
	findItems = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "tr" {
			// Check if this looks like a product row
			item := p.parseTableRow(n)
			if item.Name != "" && (item.Price > 0 || item.TotalPrice > 0) {
				items = append(items, item)
			}
		}

		// Also look for div-based item containers
		if n.Type == html.ElementNode && n.Data == "div" {
			for _, attr := range n.Attr {
				if attr.Key == "class" {
					classes := strings.ToLower(attr.Val)
					if strings.Contains(classes, "item") ||
						strings.Contains(classes, "product") ||
						strings.Contains(classes, "good") ||
						strings.Contains(classes, "position") {
						item := p.parseItemNode(n)
						if item.Name != "" && (item.Price > 0 || item.TotalPrice > 0) {
							items = append(items, item)
						}
					}
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findItems(c)
		}
	}

	findItems(doc)
	return items
}

// parseTableRow parses a table row for item data
func (p *Parser) parseTableRow(tr *html.Node) ParsedItem {
	item := ParsedItem{
		Quantity: 1,
		Unit:     "pcs",
	}

	var cells []string
	var findCells func(*html.Node)
	findCells = func(n *html.Node) {
		if n.Type == html.ElementNode && (n.Data == "td" || n.Data == "th") {
			cells = append(cells, strings.TrimSpace(p.extractText(n)))
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findCells(c)
		}
	}
	findCells(tr)

	// Common patterns: [Name, Qty, Price, Total] or [Name, Price] or [Name, Qty, Total]
	if len(cells) >= 2 {
		// First cell is usually the name
		item.Name = cells[0]

		// Try to identify numeric cells
		for i := 1; i < len(cells); i++ {
			val := p.parseAmount(cells[i])
			if val > 0 {
				if i == len(cells)-1 {
					// Last numeric is usually total
					item.TotalPrice = val
				} else if item.Price == 0 {
					// First numeric after name might be qty or price
					// If it's a small number, treat as qty
					if val < 100 && val == float64(int(val)) {
						item.Quantity = val
					} else {
						item.Price = val
					}
				} else {
					item.TotalPrice = val
				}
			}
		}
	}

	// Calculate missing values
	if item.TotalPrice == 0 && item.Price > 0 {
		item.TotalPrice = item.Price * item.Quantity
	}
	if item.Price == 0 && item.TotalPrice > 0 && item.Quantity > 0 {
		item.Price = item.TotalPrice / item.Quantity
	}

	return item
}

// parseFiscalCheck parses receipts from web.fiscalcheck.kz
func (p *Parser) parseFiscalCheck(content string, fiscalID string, rawURL string) (*ParsedReceipt, error) {
	return p.parseGeneric(content, fiscalID, rawURL)
}

// parseGeneric is a fallback parser that tries common patterns
func (p *Parser) parseGeneric(content string, fiscalID string, rawURL string) (*ParsedReceipt, error) {
	doc, err := html.Parse(strings.NewReader(content))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrParsingFailed, err)
	}

	receipt := &ParsedReceipt{
		FiscalID:     fiscalID,
		RawURL:       rawURL,
		PurchaseDate: time.Now(),
		Items:        []ParsedItem{},
	}

	// Try various common class names for store
	storeClasses := []string{"org-name", "company-name", "store-name", "merchant-name", "organization"}
	for _, class := range storeClasses {
		if name := p.extractTextByClass(doc, class); name != "" {
			receipt.StoreName = name
			break
		}
	}

	// Try to find address
	addressClasses := []string{"org-address", "address", "store-address", "merchant-address"}
	for _, class := range addressClasses {
		if addr := p.extractTextByClass(doc, class); addr != "" {
			receipt.StoreAddress = addr
			break
		}
	}

	// Try to find total
	totalClasses := []string{"total-sum", "total", "итого", "sum", "amount"}
	for _, class := range totalClasses {
		if total := p.extractTextByClass(doc, class); total != "" {
			receipt.TotalAmount = p.parseAmount(total)
			if receipt.TotalAmount > 0 {
				break
			}
		}
	}

	// Extract items
	receipt.Items = p.extractItems(doc)

	if receipt.StoreName == "" {
		receipt.StoreName = "Unknown Store"
	}

	return receipt, nil
}

func (p *Parser) extractTextByClass(n *html.Node, className string) string {
	if n.Type == html.ElementNode {
		for _, attr := range n.Attr {
			if attr.Key == "class" && strings.Contains(attr.Val, className) {
				return p.extractText(n)
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if result := p.extractTextByClass(c, className); result != "" {
			return result
		}
	}
	return ""
}

func (p *Parser) extractText(n *html.Node) string {
	var sb strings.Builder
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.TextNode {
			sb.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)
	return strings.TrimSpace(sb.String())
}

func (p *Parser) extractItems(doc *html.Node) []ParsedItem {
	var items []ParsedItem

	// Look for table rows or list items that might contain products
	var findItems func(*html.Node)
	findItems = func(n *html.Node) {
		if n.Type == html.ElementNode {
			// Check for common item container patterns
			for _, attr := range n.Attr {
				if attr.Key == "class" {
					classes := strings.ToLower(attr.Val)
					if strings.Contains(classes, "item") ||
						strings.Contains(classes, "product") ||
						strings.Contains(classes, "goods") ||
						strings.Contains(classes, "товар") {

						item := p.parseItemNode(n)
						if item.Name != "" {
							items = append(items, item)
						}
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findItems(c)
		}
	}

	findItems(doc)
	return items
}

func (p *Parser) parseItemNode(n *html.Node) ParsedItem {
	item := ParsedItem{
		Quantity: 1,
		Unit:     "pcs",
	}

	// Extract name
	nameClasses := []string{"name", "product-name", "item-name", "title", "наименование"}
	for _, class := range nameClasses {
		if name := p.extractTextByClass(n, class); name != "" {
			item.Name = name
			break
		}
	}

	// Extract quantity
	qtyClasses := []string{"quantity", "qty", "count", "количество"}
	for _, class := range qtyClasses {
		if qty := p.extractTextByClass(n, class); qty != "" {
			item.Quantity = p.parseAmount(qty)
			if item.Quantity == 0 {
				item.Quantity = 1
			}
			break
		}
	}

	// Extract price
	priceClasses := []string{"price", "unit-price", "цена"}
	for _, class := range priceClasses {
		if price := p.extractTextByClass(n, class); price != "" {
			item.Price = p.parseAmount(price)
			break
		}
	}

	// Extract total
	totalClasses := []string{"total", "sum", "amount", "сумма", "стоимость"}
	for _, class := range totalClasses {
		if total := p.extractTextByClass(n, class); total != "" {
			item.TotalPrice = p.parseAmount(total)
			break
		}
	}

	// Calculate missing values
	if item.TotalPrice == 0 && item.Price > 0 {
		item.TotalPrice = item.Price * item.Quantity
	}
	if item.Price == 0 && item.TotalPrice > 0 && item.Quantity > 0 {
		item.Price = item.TotalPrice / item.Quantity
	}

	// If no name found, use the full text as name
	if item.Name == "" {
		item.Name = p.extractText(n)
		// Truncate if too long
		if len(item.Name) > 100 {
			item.Name = item.Name[:100]
		}
	}

	return item
}

func (p *Parser) parseAmount(s string) float64 {
	// Remove common currency symbols and spaces
	s = strings.ReplaceAll(s, "₸", "")
	s = strings.ReplaceAll(s, "тг", "")
	s = strings.ReplaceAll(s, "тенге", "")
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, ",", ".")
	s = strings.TrimSpace(s)

	// Extract numbers
	re := regexp.MustCompile(`[\d.]+`)
	matches := re.FindString(s)
	if matches == "" {
		return 0
	}

	val, _ := strconv.ParseFloat(matches, 64)
	return val
}

func (p *Parser) parseDate(s string) time.Time {
	s = strings.TrimSpace(s)

	formats := []string{
		"02.01.2006 15:04:05",
		"02.01.2006 15:04",
		"02.01.2006",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
		"02/01/2006 15:04:05",
		"02/01/2006",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t
		}
	}

	return time.Time{}
}
