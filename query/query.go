package query

import (
	"fmt"
	"strings"

	e "github.com/julianlk522/fitm/error"
)

type Query struct {
	Text  string
	Args  []interface{}
	Error error
}

func GetPeriodClause(period string) (clause string, err error) {
	var days int
	switch period {
	case "day":
		days = 1
	case "week":
		days = 7
	case "month":
		days = 30
	case "year":
		days = 365
	default:
		return "", e.ErrInvalidPeriod
	}

	return fmt.Sprintf("submit_date >= date('now', '-%d days')", days), nil
}

func GetCatsWithEscapedReservedChars(cats []string) []string {
	modified_cats := make([]string, len(cats))
	for i := 0; i < len(cats); i++ {
		modified_cats[i] = WithDoubleQuotesAroundReservedChars(cats[i])
	}

	return modified_cats
}

func WithDoubleQuotesAroundReservedChars(cat string) string {
	return reserved_chars_double_quotes_surrounder.Replace(cat)
}

var reserved_chars_double_quotes_surrounder = strings.NewReplacer(
	// ! seems to work already with no modifications
	".", `"."`,
	"-", `"-"`,
	// + seems to work
	"'", `"'"`,
	// double quotes seems to work
	"#", `"#"`,
	"$", `"$"`,
	"%", `"%"`,
	"&", `"&"`,
	"\\", `"\"`,
	"/", `"/"`,
	"(", `"("`,
	")", `")"`,
	"[", `"["`,
	"]", `"]"`,
	"{", `"{"`,
	"}", `"}"`,
	"|", `"|"`,
	":", `":"`,
	";", `";"`,
	"=", `"="`,
	"?", `"?"`,
	"@", `"@"`,
)

func GetCatsOptionalPluralOrSingularForms(cats []string) []string {
	modified_cats := make([]string, len(cats))
	for i := 0; i < len(cats); i++ {
		modified_cats[i] = WithOptionalPluralOrSingularForm(cats[i])
	}

	return modified_cats
}

func WithOptionalPluralOrSingularForm(cat string) string {
	if strings.HasSuffix(cat, "ss") {
		return fmt.Sprintf("(%s OR %s)", cat, cat+"es")
	} else if strings.HasSuffix(cat, "sses") {
		return fmt.Sprintf("(%s OR %s)", cat, strings.TrimSuffix(cat, "es"))
	} else if strings.HasSuffix(cat, "s") {
		return fmt.Sprintf("(%s OR %s OR %s)", cat, cat+"es", strings.TrimSuffix(cat, "s"))
	} else {
		return fmt.Sprintf("(%s OR %s)", cat, cat+"s")
	}
}
