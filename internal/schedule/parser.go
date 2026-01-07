// Package schedule provides parsing and execution of schedule tokens.
package schedule

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Token represents a parsed schedule token.
type Token struct {
	Raw       string        // Original token text (e.g., "@daily:9am")
	Type      TokenType     // Type of schedule
	Time      *TimeSpec     // Specific time (if applicable)
	Recurring *RecurSpec    // Recurrence pattern (if applicable)
	Offset    *OffsetSpec   // Relative offset (if applicable)
	Date      *time.Time    // Specific date (if applicable)
	Line      int           // Source line number
	Column    int           // Source column number
}

// TokenType indicates the kind of schedule token.
type TokenType int

const (
	TokenRelative   TokenType = iota // @today, @tomorrow, @yesterday
	TokenWeekday                     // @monday, @tuesday, etc.
	TokenISODate                     // @2024-03-15
	TokenTime                        // @9am, @9:30pm
	TokenOffset                      // @in:2hours, @in:30min
	TokenDaily                       // @daily:9am
	TokenWeekly                      // @weekly:mon,wed
	TokenMonthly                     // @monthly:1st, @monthly:15
	TokenYearly                      // @yearly:mar-15
)

// TimeSpec represents a time of day.
type TimeSpec struct {
	Hour   int
	Minute int
}

// RecurSpec represents a recurrence pattern.
type RecurSpec struct {
	Days      []time.Weekday // For weekly: specific days
	DayOfMonth int           // For monthly: day number (1-31)
	Month     time.Month     // For yearly: month
	Day       int            // For yearly: day of month
	Time      *TimeSpec      // Time to trigger
}

// OffsetSpec represents a relative time offset.
type OffsetSpec struct {
	Duration time.Duration
}

// ParseWarning represents a non-fatal parsing issue.
type ParseWarning struct {
	Message string
	Line    int
	Column  int
	Token   string
}

// Parser parses schedule tokens from text.
type Parser struct {
	Location *time.Location
	Warnings []ParseWarning
}

// NewParser creates a new schedule parser with the specified timezone.
func NewParser(location *time.Location) *Parser {
	if location == nil {
		location = time.Local
	}
	return &Parser{
		Location: location,
		Warnings: make([]ParseWarning, 0),
	}
}

// ParseText extracts schedule tokens from markdown text, skipping code blocks.
func (p *Parser) ParseText(text string) ([]*Token, error) {
	p.Warnings = make([]ParseWarning, 0)
	var tokens []*Token

	lines := strings.Split(text, "\n")
	inCodeBlock := false
	inInlineCode := false

	for lineNum, line := range lines {
		// Check for code block boundaries
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inCodeBlock = !inCodeBlock
			continue
		}
		if inCodeBlock {
			continue
		}

		// Parse tokens from this line, skipping inline code
		lineTokens := p.parseLine(line, lineNum+1, &inInlineCode)
		tokens = append(tokens, lineTokens...)
	}

	return tokens, nil
}

// parseLine extracts schedule tokens from a single line.
func (p *Parser) parseLine(line string, lineNum int, inInlineCode *bool) []*Token {
	var tokens []*Token

	// Track positions and inline code state
	col := 0
	for col < len(line) {
		// Handle inline code boundaries
		if line[col] == '`' {
			*inInlineCode = !*inInlineCode
			col++
			continue
		}

		// Skip if in inline code
		if *inInlineCode {
			col++
			continue
		}

		// Check for escaped @ symbol
		if col > 0 && line[col-1] == '\\' && col < len(line) && line[col] == '@' {
			col++
			continue
		}

		// Look for @ tokens
		if line[col] == '@' {
			token, end := p.parseToken(line, col, lineNum)
			if token != nil {
				tokens = append(tokens, token)
				col = end
				continue
			}
		}

		col++
	}

	return tokens
}

// parseToken attempts to parse a schedule token starting at position.
func (p *Parser) parseToken(line string, start int, lineNum int) (*Token, int) {
	// Extract the token text (up to whitespace or end)
	end := start + 1
	for end < len(line) && !isTokenTerminator(line[end]) {
		end++
	}

	tokenText := line[start:end]
	if len(tokenText) <= 1 {
		return nil, start + 1
	}

	token, err := p.ParseToken(tokenText)
	if err != nil {
		p.Warnings = append(p.Warnings, ParseWarning{
			Message: err.Error(),
			Line:    lineNum,
			Column:  start + 1,
			Token:   tokenText,
		})
		return nil, end
	}

	if token != nil {
		token.Line = lineNum
		token.Column = start + 1
	}

	return token, end
}

// isTokenTerminator returns true if the character ends a token.
func isTokenTerminator(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r' ||
		c == ',' || c == '.' || c == ';' || c == ')' || c == ']' || c == '}'
}

// ParseToken parses a single schedule token string.
func (p *Parser) ParseToken(s string) (*Token, error) {
	if !strings.HasPrefix(s, "@") {
		return nil, fmt.Errorf("schedule token must start with @")
	}

	s = strings.TrimPrefix(s, "@")
	if s == "" {
		return nil, fmt.Errorf("empty schedule token")
	}

	token := &Token{Raw: "@" + s}

	// Try parsing in order of specificity
	if t, ok := p.parseRelative(s); ok {
		token.Type = TokenRelative
		token.Date = t
		return token, nil
	}

	if t, ok := p.parseWeekday(s); ok {
		token.Type = TokenWeekday
		token.Date = t
		return token, nil
	}

	if t, ok := p.parseISODate(s); ok {
		token.Type = TokenISODate
		token.Date = t
		return token, nil
	}

	if t, ok := p.parseTimeOnly(s); ok {
		token.Type = TokenTime
		token.Time = t
		return token, nil
	}

	if t, ok := p.parseOffset(s); ok {
		token.Type = TokenOffset
		token.Offset = t
		return token, nil
	}

	if t, ts, ok := p.parseDaily(s); ok {
		token.Type = TokenDaily
		token.Recurring = &RecurSpec{Time: ts}
		_ = t
		return token, nil
	}

	if days, ts, ok := p.parseWeekly(s); ok {
		token.Type = TokenWeekly
		token.Recurring = &RecurSpec{Days: days, Time: ts}
		return token, nil
	}

	if day, ts, ok := p.parseMonthly(s); ok {
		token.Type = TokenMonthly
		token.Recurring = &RecurSpec{DayOfMonth: day, Time: ts}
		return token, nil
	}

	if month, day, ts, ok := p.parseYearly(s); ok {
		token.Type = TokenYearly
		token.Recurring = &RecurSpec{Month: month, Day: day, Time: ts}
		return token, nil
	}

	return nil, fmt.Errorf("unrecognized schedule token: @%s", s)
}

// parseRelative handles @today, @tomorrow, @yesterday.
func (p *Parser) parseRelative(s string) (*time.Time, bool) {
	now := time.Now().In(p.Location)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, p.Location)

	switch strings.ToLower(s) {
	case "today":
		return &today, true
	case "tomorrow":
		t := today.AddDate(0, 0, 1)
		return &t, true
	case "yesterday":
		t := today.AddDate(0, 0, -1)
		return &t, true
	}
	return nil, false
}

// parseWeekday handles @monday, @tuesday, etc.
func (p *Parser) parseWeekday(s string) (*time.Time, bool) {
	weekdays := map[string]time.Weekday{
		"sunday":    time.Sunday,
		"monday":    time.Monday,
		"tuesday":   time.Tuesday,
		"wednesday": time.Wednesday,
		"thursday":  time.Thursday,
		"friday":    time.Friday,
		"saturday":  time.Saturday,
		"sun":       time.Sunday,
		"mon":       time.Monday,
		"tue":       time.Tuesday,
		"wed":       time.Wednesday,
		"thu":       time.Thursday,
		"fri":       time.Friday,
		"sat":       time.Saturday,
	}

	targetDay, ok := weekdays[strings.ToLower(s)]
	if !ok {
		return nil, false
	}

	now := time.Now().In(p.Location)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, p.Location)
	currentDay := today.Weekday()

	daysUntil := int(targetDay) - int(currentDay)
	if daysUntil <= 0 {
		daysUntil += 7
	}

	t := today.AddDate(0, 0, daysUntil)
	return &t, true
}

// parseISODate handles @2024-03-15.
func (p *Parser) parseISODate(s string) (*time.Time, bool) {
	// Match YYYY-MM-DD format
	re := regexp.MustCompile(`^(\d{4})-(\d{2})-(\d{2})$`)
	matches := re.FindStringSubmatch(s)
	if matches == nil {
		return nil, false
	}

	year, _ := strconv.Atoi(matches[1])
	month, _ := strconv.Atoi(matches[2])
	day, _ := strconv.Atoi(matches[3])

	t := time.Date(year, time.Month(month), day, 0, 0, 0, 0, p.Location)
	return &t, true
}

// parseTimeOnly handles @9am, @9:30pm, @14:00.
func (p *Parser) parseTimeOnly(s string) (*TimeSpec, bool) {
	// Try 12-hour format: 9am, 9:30pm
	re12 := regexp.MustCompile(`^(\d{1,2})(?::(\d{2}))?(am|pm)$`)
	if matches := re12.FindStringSubmatch(strings.ToLower(s)); matches != nil {
		hour, _ := strconv.Atoi(matches[1])
		minute := 0
		if matches[2] != "" {
			minute, _ = strconv.Atoi(matches[2])
		}

		if matches[3] == "pm" && hour != 12 {
			hour += 12
		} else if matches[3] == "am" && hour == 12 {
			hour = 0
		}

		if hour >= 0 && hour < 24 && minute >= 0 && minute < 60 {
			return &TimeSpec{Hour: hour, Minute: minute}, true
		}
	}

	// Try 24-hour format: 14:00, 9:30
	re24 := regexp.MustCompile(`^(\d{1,2}):(\d{2})$`)
	if matches := re24.FindStringSubmatch(s); matches != nil {
		hour, _ := strconv.Atoi(matches[1])
		minute, _ := strconv.Atoi(matches[2])

		if hour >= 0 && hour < 24 && minute >= 0 && minute < 60 {
			return &TimeSpec{Hour: hour, Minute: minute}, true
		}
	}

	return nil, false
}

// parseOffset handles @in:2hours, @in:30min, @in:1day.
func (p *Parser) parseOffset(s string) (*OffsetSpec, bool) {
	if !strings.HasPrefix(strings.ToLower(s), "in:") {
		return nil, false
	}

	offsetStr := s[3:]

	// Parse duration with units: 2hours, 30min, 1day, etc.
	re := regexp.MustCompile(`^(\d+)(hours?|mins?|minutes?|days?|weeks?|h|m|d|w)$`)
	matches := re.FindStringSubmatch(strings.ToLower(offsetStr))
	if matches == nil {
		return nil, false
	}

	value, _ := strconv.Atoi(matches[1])
	unit := matches[2]

	var duration time.Duration
	switch {
	case strings.HasPrefix(unit, "hour") || unit == "h":
		duration = time.Duration(value) * time.Hour
	case strings.HasPrefix(unit, "min") || unit == "m":
		duration = time.Duration(value) * time.Minute
	case strings.HasPrefix(unit, "day") || unit == "d":
		duration = time.Duration(value) * 24 * time.Hour
	case strings.HasPrefix(unit, "week") || unit == "w":
		duration = time.Duration(value) * 7 * 24 * time.Hour
	default:
		return nil, false
	}

	return &OffsetSpec{Duration: duration}, true
}

// parseDaily handles @daily:9am.
func (p *Parser) parseDaily(s string) (bool, *TimeSpec, bool) {
	if !strings.HasPrefix(strings.ToLower(s), "daily:") {
		return false, nil, false
	}

	timeStr := s[6:]
	ts, ok := p.parseTimeOnly(timeStr)
	if !ok {
		return false, nil, false
	}

	return true, ts, true
}

// parseWeekly handles @weekly:mon,wed or @weekly:mon,wed:9am.
func (p *Parser) parseWeekly(s string) ([]time.Weekday, *TimeSpec, bool) {
	if !strings.HasPrefix(strings.ToLower(s), "weekly:") {
		return nil, nil, false
	}

	rest := s[7:]

	// Check if there's a time component
	var daysPart, timePart string
	if idx := strings.LastIndex(rest, ":"); idx != -1 {
		// Could be time:
		possibleTime := rest[idx+1:]
		if _, ok := p.parseTimeOnly(possibleTime); ok {
			daysPart = rest[:idx]
			timePart = possibleTime
		} else {
			daysPart = rest
		}
	} else {
		daysPart = rest
	}

	// Parse days
	dayNames := strings.Split(daysPart, ",")
	var days []time.Weekday

	weekdayMap := map[string]time.Weekday{
		"sun": time.Sunday, "sunday": time.Sunday,
		"mon": time.Monday, "monday": time.Monday,
		"tue": time.Tuesday, "tuesday": time.Tuesday,
		"wed": time.Wednesday, "wednesday": time.Wednesday,
		"thu": time.Thursday, "thursday": time.Thursday,
		"fri": time.Friday, "friday": time.Friday,
		"sat": time.Saturday, "saturday": time.Saturday,
	}

	for _, dayName := range dayNames {
		day, ok := weekdayMap[strings.ToLower(strings.TrimSpace(dayName))]
		if !ok {
			return nil, nil, false
		}
		days = append(days, day)
	}

	if len(days) == 0 {
		return nil, nil, false
	}

	var ts *TimeSpec
	if timePart != "" {
		ts, _ = p.parseTimeOnly(timePart)
	}

	return days, ts, true
}

// parseMonthly handles @monthly:1st, @monthly:15, @monthly:15:9am.
func (p *Parser) parseMonthly(s string) (int, *TimeSpec, bool) {
	if !strings.HasPrefix(strings.ToLower(s), "monthly:") {
		return 0, nil, false
	}

	rest := s[8:]

	// Check for time component
	var dayPart, timePart string
	if idx := strings.LastIndex(rest, ":"); idx != -1 {
		possibleTime := rest[idx+1:]
		if _, ok := p.parseTimeOnly(possibleTime); ok {
			dayPart = rest[:idx]
			timePart = possibleTime
		} else {
			dayPart = rest
		}
	} else {
		dayPart = rest
	}

	// Parse day of month
	dayPart = strings.ToLower(dayPart)
	dayPart = strings.TrimSuffix(dayPart, "st")
	dayPart = strings.TrimSuffix(dayPart, "nd")
	dayPart = strings.TrimSuffix(dayPart, "rd")
	dayPart = strings.TrimSuffix(dayPart, "th")

	day, err := strconv.Atoi(dayPart)
	if err != nil || day < 1 || day > 31 {
		return 0, nil, false
	}

	var ts *TimeSpec
	if timePart != "" {
		ts, _ = p.parseTimeOnly(timePart)
	}

	return day, ts, true
}

// parseYearly handles @yearly:mar-15, @yearly:mar-15:9am.
func (p *Parser) parseYearly(s string) (time.Month, int, *TimeSpec, bool) {
	if !strings.HasPrefix(strings.ToLower(s), "yearly:") {
		return 0, 0, nil, false
	}

	rest := s[7:]

	// Check for time component
	var datePart, timePart string
	if idx := strings.LastIndex(rest, ":"); idx != -1 {
		possibleTime := rest[idx+1:]
		if _, ok := p.parseTimeOnly(possibleTime); ok {
			datePart = rest[:idx]
			timePart = possibleTime
		} else {
			datePart = rest
		}
	} else {
		datePart = rest
	}

	// Parse month-day: mar-15, march-15, 3-15
	parts := strings.Split(datePart, "-")
	if len(parts) != 2 {
		return 0, 0, nil, false
	}

	monthMap := map[string]time.Month{
		"jan": time.January, "january": time.January, "1": time.January,
		"feb": time.February, "february": time.February, "2": time.February,
		"mar": time.March, "march": time.March, "3": time.March,
		"apr": time.April, "april": time.April, "4": time.April,
		"may": time.May, "5": time.May,
		"jun": time.June, "june": time.June, "6": time.June,
		"jul": time.July, "july": time.July, "7": time.July,
		"aug": time.August, "august": time.August, "8": time.August,
		"sep": time.September, "september": time.September, "9": time.September,
		"oct": time.October, "october": time.October, "10": time.October,
		"nov": time.November, "november": time.November, "11": time.November,
		"dec": time.December, "december": time.December, "12": time.December,
	}

	month, ok := monthMap[strings.ToLower(parts[0])]
	if !ok {
		return 0, 0, nil, false
	}

	day, err := strconv.Atoi(parts[1])
	if err != nil || day < 1 || day > 31 {
		return 0, 0, nil, false
	}

	var ts *TimeSpec
	if timePart != "" {
		ts, _ = p.parseTimeOnly(timePart)
	}

	return month, day, ts, true
}

// NextOccurrence calculates the next occurrence of this schedule token.
func (t *Token) NextOccurrence(now time.Time, loc *time.Location) time.Time {
	if loc == nil {
		loc = time.Local
	}
	now = now.In(loc)

	switch t.Type {
	case TokenRelative:
		if t.Date != nil {
			return *t.Date
		}
	case TokenWeekday:
		if t.Date != nil {
			return *t.Date
		}
	case TokenISODate:
		if t.Date != nil {
			return *t.Date
		}
	case TokenTime:
		if t.Time != nil {
			next := time.Date(now.Year(), now.Month(), now.Day(),
				t.Time.Hour, t.Time.Minute, 0, 0, loc)
			if next.Before(now) || next.Equal(now) {
				next = next.AddDate(0, 0, 1)
			}
			return next
		}
	case TokenOffset:
		if t.Offset != nil {
			return now.Add(t.Offset.Duration)
		}
	case TokenDaily:
		if t.Recurring != nil && t.Recurring.Time != nil {
			next := time.Date(now.Year(), now.Month(), now.Day(),
				t.Recurring.Time.Hour, t.Recurring.Time.Minute, 0, 0, loc)
			if next.Before(now) || next.Equal(now) {
				next = next.AddDate(0, 0, 1)
			}
			return adjustForDST(next, loc)
		}
	case TokenWeekly:
		if t.Recurring != nil && len(t.Recurring.Days) > 0 {
			return nextWeeklyOccurrence(now, t.Recurring.Days, t.Recurring.Time, loc)
		}
	case TokenMonthly:
		if t.Recurring != nil && t.Recurring.DayOfMonth > 0 {
			return nextMonthlyOccurrence(now, t.Recurring.DayOfMonth, t.Recurring.Time, loc)
		}
	case TokenYearly:
		if t.Recurring != nil && t.Recurring.Month > 0 && t.Recurring.Day > 0 {
			return nextYearlyOccurrence(now, t.Recurring.Month, t.Recurring.Day, t.Recurring.Time, loc)
		}
	}

	return now
}

// nextWeeklyOccurrence finds the next occurrence for weekly schedule.
func nextWeeklyOccurrence(now time.Time, days []time.Weekday, ts *TimeSpec, loc *time.Location) time.Time {
	hour, minute := 0, 0
	if ts != nil {
		hour, minute = ts.Hour, ts.Minute
	}

	currentWeekday := now.Weekday()
	currentTime := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, loc)

	var nearest time.Time
	minDays := 8 // More than a week

	for _, targetDay := range days {
		daysUntil := int(targetDay) - int(currentWeekday)
		if daysUntil < 0 {
			daysUntil += 7
		}

		candidate := currentTime.AddDate(0, 0, daysUntil)

		// If it's today but time has passed, move to next week
		if daysUntil == 0 && (candidate.Before(now) || candidate.Equal(now)) {
			candidate = candidate.AddDate(0, 0, 7)
			daysUntil = 7
		}

		if daysUntil < minDays {
			minDays = daysUntil
			nearest = candidate
		}
	}

	return adjustForDST(nearest, loc)
}

// nextMonthlyOccurrence finds the next occurrence for monthly schedule.
func nextMonthlyOccurrence(now time.Time, dayOfMonth int, ts *TimeSpec, loc *time.Location) time.Time {
	hour, minute := 0, 0
	if ts != nil {
		hour, minute = ts.Hour, ts.Minute
	}

	// Try this month
	candidate := time.Date(now.Year(), now.Month(), dayOfMonth, hour, minute, 0, 0, loc)

	// Handle months with fewer days
	for candidate.Day() != dayOfMonth {
		// Day doesn't exist in this month, use last day
		candidate = time.Date(now.Year(), now.Month()+1, 0, hour, minute, 0, 0, loc)
		break
	}

	if candidate.Before(now) || candidate.Equal(now) {
		// Move to next month
		candidate = time.Date(now.Year(), now.Month()+1, dayOfMonth, hour, minute, 0, 0, loc)
		// Handle months with fewer days
		for candidate.Day() != dayOfMonth {
			candidate = time.Date(now.Year(), now.Month()+2, 0, hour, minute, 0, 0, loc)
			break
		}
	}

	return adjustForDST(candidate, loc)
}

// nextYearlyOccurrence finds the next occurrence for yearly schedule.
func nextYearlyOccurrence(now time.Time, month time.Month, day int, ts *TimeSpec, loc *time.Location) time.Time {
	hour, minute := 0, 0
	if ts != nil {
		hour, minute = ts.Hour, ts.Minute
	}

	// Try this year
	candidate := time.Date(now.Year(), month, day, hour, minute, 0, 0, loc)

	// Handle Feb 29 on non-leap years
	if candidate.Month() != month {
		candidate = time.Date(now.Year(), month+1, 0, hour, minute, 0, 0, loc)
	}

	if candidate.Before(now) || candidate.Equal(now) {
		// Move to next year
		candidate = time.Date(now.Year()+1, month, day, hour, minute, 0, 0, loc)
		if candidate.Month() != month {
			candidate = time.Date(now.Year()+1, month+1, 0, hour, minute, 0, 0, loc)
		}
	}

	return adjustForDST(candidate, loc)
}

// adjustForDST handles DST transitions by moving to the next valid time.
func adjustForDST(t time.Time, loc *time.Location) time.Time {
	// Check if the time exists (DST spring forward creates gaps)
	_, offset1 := t.Zone()
	oneHourLater := t.Add(time.Hour)
	_, offset2 := oneHourLater.Zone()

	// If offsets differ and time doesn't round-trip correctly, adjust
	rebuilt := time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), 0, 0, loc)
	if rebuilt.Hour() != t.Hour() {
		// Time was adjusted due to DST gap, move forward
		return rebuilt
	}

	// Also check for the case where the time was in a DST gap
	if offset1 != offset2 {
		// We're near a DST transition
		// Create the time again to ensure it's valid
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), 0, 0, loc)
	}

	return t
}

// String returns a human-readable representation of the token.
func (t TokenType) String() string {
	switch t {
	case TokenRelative:
		return "relative"
	case TokenWeekday:
		return "weekday"
	case TokenISODate:
		return "iso-date"
	case TokenTime:
		return "time"
	case TokenOffset:
		return "offset"
	case TokenDaily:
		return "daily"
	case TokenWeekly:
		return "weekly"
	case TokenMonthly:
		return "monthly"
	case TokenYearly:
		return "yearly"
	default:
		return "unknown"
	}
}

// String returns a human-readable representation of TimeSpec.
func (ts *TimeSpec) String() string {
	if ts == nil {
		return ""
	}
	return fmt.Sprintf("%02d:%02d", ts.Hour, ts.Minute)
}
