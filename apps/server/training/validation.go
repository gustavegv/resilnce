package training

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/gustavegv/resilnce/apps/server/predefined"
)

func (c *Configuration) UnmarshalJSON(raw []byte) error {
	type value Configuration
	var decoded value
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(&decoded); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return err
	}
	for field, value := range fields {
		if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return invalid("configuration."+field, "Supply a value or omit an optional field")
		}
	}
	*c = Configuration(decoded)
	return nil
}

type Error struct {
	Status        int
	Code, Message string
	Details       any
}

func (e *Error) Error() string { return e.Message }
func invalid(field, message string) error {
	return &Error{422, "invalid-field", message, map[string]string{"field": field}}
}
func conflict(code, message string) error { return &Error{409, code, message, nil} }

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func validID(id string) bool { return uuidPattern.MatchString(id) }
func validateName(name, field string) error {
	if strings.TrimSpace(name) == "" || utf8.RuneCountInString(strings.TrimSpace(name)) > 100 {
		return invalid(field, "Use a nonblank name of at most 100 characters")
	}
	return nil
}

var decimalPattern = regexp.MustCompile(`^[0-9]+(?:\.[0-9]{1,2})?$`)

// parseCents validates decimal input before PostgreSQL can round it. Integer
// cents are the authoritative arithmetic unit for weights and increments.
func parseCents(value string, min, max int64) (int64, error) {
	if !decimalPattern.MatchString(value) {
		return 0, fmt.Errorf("use a decimal string with at most two decimal places")
	}
	parts := strings.SplitN(value, ".", 2)
	whole, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || whole > max/100 {
		return 0, fmt.Errorf("decimal is out of range")
	}
	fraction := int64(0)
	if len(parts) == 2 {
		digits := parts[1]
		if len(digits) == 1 {
			digits += "0"
		}
		fraction, _ = strconv.ParseInt(digits, 10, 64)
	}
	cents := whole*100 + fraction
	if cents < min || cents > max {
		return 0, fmt.Errorf("decimal is out of range")
	}
	return cents, nil
}
func formatCents(cents int64) string { return fmt.Sprintf("%d.%02d", cents/100, cents%100) }

// Codes distinguishes an omitted catalog array (default empty) from explicit
// null. Null elements decode to empty strings and fail membership validation.
type Codes []string

func (c *Codes) UnmarshalJSON(raw []byte) error {
	if string(raw) == "null" {
		return invalid("muscleGroupCodes", "Use an array of catalog codes")
	}
	var values []string
	if err := json.Unmarshal(raw, &values); err != nil {
		return err
	}
	*c = values
	return nil
}
func normalizeCodes(codes Codes) (Codes, error) {
	selected := map[string]bool{}
	for _, code := range codes {
		if selected[code] {
			return nil, invalid("muscleGroupCodes", "Do not repeat muscle-group codes")
		}
		selected[code] = true
	}
	ordered := Codes{}
	for _, group := range predefined.MuscleGroups() {
		if selected[group.Code] {
			ordered = append(ordered, group.Code)
			delete(selected, group.Code)
		}
	}
	if len(selected) > 0 {
		return nil, invalid("muscleGroupCodes", "Use known muscle-group codes")
	}
	return ordered, nil
}
func validateCreate(c CreateSession) error {
	if !validID(c.OperationID) {
		return invalid("operationId", "Use a UUID operation ID")
	}
	if err := validateName(c.Name, "name"); err != nil {
		return err
	}
	if len(c.Items) < 1 || len(c.Items) > 100 {
		return invalid("items", "Add between 1 and 100 placements")
	}
	for i, item := range c.Items {
		field := fmt.Sprintf("items[%d]", i)
		if item.Kind != "createExercise" {
			return invalid(field+".kind", "Explicitly create a new exercise")
		}
		if err := validateName(item.Name, field+".name"); err != nil {
			return err
		}
		if _, err := normalizeCodes(item.MuscleGroupCodes); err != nil {
			return err
		}
		cfg := item.Configuration
		if cfg.PrescribedSets < 1 || cfg.PrescribedSets > 20 {
			return invalid(field+".configuration.prescribedSets", "Use 1–20 prescribed sets")
		}
		if _, err := parseCents(cfg.InitialWeight, 0, 999900); err != nil {
			return invalid(field+".configuration.initialWeight", err.Error())
		}
		if cfg.RepThreshold != nil && (*cfg.RepThreshold < 1 || *cfg.RepThreshold > 99) {
			return invalid(field+".configuration.repThreshold", "Use a threshold of 1–99")
		}
		if cfg.AutoIncrement != nil {
			if _, err := parseCents(*cfg.AutoIncrement, 25, 9900); err != nil {
				return invalid(field+".configuration.autoIncrement", err.Error())
			}
		}
		if cfg.RestSeconds != nil && (*cfg.RestSeconds < 0 || *cfg.RestSeconds > 2147483647) {
			return invalid(field+".configuration.restSeconds", "Use nonnegative whole rest seconds within storage range")
		}
	}
	return nil
}
