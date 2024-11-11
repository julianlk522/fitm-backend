package query

import (
	"fmt"
	"strings"

	e "github.com/julianlk522/fitm/error"
)

const (
	TAG_RANKINGS_PAGE_LIMIT = 20
	GLOBAL_CATS_PAGE_LIMIT  = 20
	MORE_GLOBAL_CATS_PAGE_LIMIT = 100

	SPELLFIX_DISTANCE_LIMIT = 100
	SPELLFIX_MATCHES_LIMIT  = 3
)

// Tags Page link
type TagPageLink struct {
	*Query
}

func NewTagPageLink(link_id string) *TagPageLink {
	return (&TagPageLink{Query: 
		&Query{
			Text: 
				TAG_PAGE_LINK_BASE_FIELDS + 
				TAG_PAGE_LINK_BASE_FROM + 
				TAG_PAGE_LINK_BASE_JOINS,
			Args: []interface{}{link_id},
		},
	})
}

const TAG_PAGE_LINK_BASE_FIELDS = `SELECT 
	links_id as link_id, 
	url, 
	sb, 
	sd, 
	cats, 
	summary,
	COALESCE(summary_count,0) as summary_count, 
	COUNT("Link Likes".id) as like_count, 
	img_url`

const TAG_PAGE_LINK_BASE_FROM = `
FROM 
	(
	SELECT 
		id as links_id, 
		url, 
		submitted_by as sb, 
		submit_date as sd, 
		COALESCE(global_cats,"") as cats, 
		global_summary as summary, 
		COALESCE(img_url,"") as img_url 
	FROM Links
	WHERE id = ?
	)`

const TAG_PAGE_LINK_BASE_JOINS = `
LEFT JOIN "Link Likes"
ON "Link Likes".link_id = links_id
LEFT JOIN
	(
	SELECT count(*) as summary_count, link_id as slink_id
	FROM Summaries
	GROUP BY slink_id
	)
ON slink_id = links_id;`

func (l *TagPageLink) AsSignedInUser(user_id string) *TagPageLink {
	l.Text = strings.Replace(
		l.Text,
		TAG_PAGE_LINK_BASE_FIELDS,
		TAG_PAGE_LINK_BASE_FIELDS+TAG_PAGE_LINK_AUTH_FIELDS,
		1,
	)

	l.Text = strings.Replace(
		l.Text,
		";",
		TAG_PAGE_LINK_AUTH_JOINS,
		1,
	)
	l.Args = append(l.Args, user_id, user_id)

	return l
}

const TAG_PAGE_LINK_AUTH_FIELDS = `,
	COALESCE(is_liked,0) as is_liked, 
	COALESCE(is_copied,0) as is_copied`

const TAG_PAGE_LINK_AUTH_JOINS = ` 
	LEFT JOIN
		(
		SELECT id as like_id, count(*) as is_liked, user_id as luser_id, link_id as like_link_id2
		FROM "Link Likes"
		WHERE luser_id = ?
		GROUP BY like_id
		)
	ON like_link_id2 = link_id
	LEFT JOIN
		(
		SELECT id as copy_id, count(*) as is_copied, user_id as cuser_id, link_id as copy_link_id
		FROM "Link Copies"
		WHERE cuser_id = ?
		GROUP BY copy_id
		)
	ON copy_link_id = link_id;`

// Tag Rankings (cat overlap scores)
// ranked from highest lifespan overlap to lowest
type TagRankings struct {
	*Query
}

func NewTagRankings(link_id string) *TagRankings {
	return (&TagRankings{Query: 
		&Query{
			Text: 
				TAG_RANKINGS_BASE,
			Args: []interface{}{link_id, TAG_RANKINGS_PAGE_LIMIT},
		},
	})
}

const TAG_RANKINGS_BASE_FIELDS = `SELECT
	(julianday('now') - julianday(last_updated)) / (julianday('now') - julianday(submit_date)) * 100 AS lifespan_overlap, 
	cats`

var TAG_RANKINGS_BASE = TAG_RANKINGS_BASE_FIELDS + ` 
FROM Tags 
INNER JOIN Links 
ON Links.id = Tags.link_id
WHERE link_id = ?
ORDER BY lifespan_overlap DESC
LIMIT ?`

func (o *TagRankings) Public() *TagRankings {
	o.Text = strings.Replace(
		o.Text,
		TAG_RANKINGS_BASE_FIELDS,
		TAG_RANKINGS_BASE_FIELDS + TAG_RANKINGS_PUBLIC_FIELDS,
		1,
	)

	return o
}

const TAG_RANKINGS_PUBLIC_FIELDS = `, 
	Tags.submitted_by, 
	last_updated`

// Global Cat Counts
type GlobalCatCounts struct {
	*Query
}

func NewTopGlobalCatCounts() *GlobalCatCounts {
	return (&GlobalCatCounts{
		Query: &Query{
			Text: 
				GLOBAL_CATS_BASE,
			Args: []interface{}{GLOBAL_CATS_PAGE_LIMIT},
		},
	})
}

const GLOBAL_CATS_BASE = `WITH RECURSIVE Split(id, global_cats, str) AS (
    SELECT id, '', global_cats||','
    FROM Links
    UNION ALL SELECT
    id,
    substr(str, 0, instr(str, ',')),
    substr(str, instr(str, ',') + 1)
    FROM Split
    WHERE str != ''
),
OrderedCats AS (
    SELECT 
        FIRST_VALUE(global_cats) OVER (
            PARTITION BY LOWER(global_cats)
            ORDER BY global_cats = LOWER(global_cats) DESC
        ) as preferred_cats,
        id
    FROM Split
    WHERE global_cats != ''
)
SELECT 
    preferred_cats as global_cats,
    COUNT(preferred_cats) as count
FROM OrderedCats
GROUP BY preferred_cats
ORDER BY count DESC, preferred_cats ASC
LIMIT ?;`
// preferred_cats is so that when the same cat appears twice, once capitalized and once not, the lowercase version is listed
// unless the uppercase version has a higher count
// ORDER BY global_cats = LOWER(global_cats) DESC ensures that the lowercase version is listed first

func (t *GlobalCatCounts) SubcatsOfCats(cats_params string) *GlobalCatCounts {
	// take lowercase to ensure all case variations are returned
	cats := strings.Split(strings.ToLower(cats_params), ",")

	// build NOT IN clause
	not_in_clause := `
	AND LOWER(global_cats) NOT IN (?`

	t.Args = append(t.Args, cats[0])
	for i := 1; i < len(cats); i++ {
		not_in_clause += ", ?"
		t.Args = append(t.Args, cats[i])
	}
	not_in_clause += ")"

	// build match clause
	match_clause := `
	AND id IN (
		SELECT link_id
		FROM global_cats_fts
		WHERE global_cats MATCH ?
		)`

	// with MATCH, reserved chars must be surrounded with ""
	match_arg := 
		// and singular/plural variants are added after escaping
		// reserved chars so that "(" and ")" are preserved
		WithOptionalPluralOrSingularForm(
			WithDoubleQuotesAroundReservedChars(
				cats[0],
			),
		)
	for i := 1; i < len(cats); i++ {
		match_arg += " AND " + 
			WithOptionalPluralOrSingularForm(
				WithDoubleQuotesAroundReservedChars(cats[i]),
		)
	}
	// add optional singular/plural variants
	// (skip for NOT IN clause otherwise subcats include filters)
	t.Args = append(t.Args, match_arg)

	t.Text = strings.Replace(
		t.Text,
		"WHERE global_cats != ''",
		"WHERE global_cats != ''" + 
		not_in_clause + 
		match_clause,
		1)

	// move LIMIT arg to end
	t.Args = append(t.Args[1:], GLOBAL_CATS_PAGE_LIMIT)

	return t
}

func (t *GlobalCatCounts) DuringPeriod(period string) *GlobalCatCounts {
	clause, err := GetPeriodClause(period)
	if err != nil {
		t.Error = err
		return t
	}

	t.Text = strings.Replace(
		t.Text,
		"FROM Links",
		fmt.Sprintf(
			`FROM Links
			WHERE %s`,
			clause,
		),
		1)

	return t
}

func (t *GlobalCatCounts) More() *GlobalCatCounts {
	t.Args[len(t.Args)-1] = MORE_GLOBAL_CATS_PAGE_LIMIT

	return t
}

// Global Cats Spellfix Matches For Snippet
type SpellfixMatches struct {
	*Query
}

func NewSpellfixMatchesForSnippet(snippet string) *SpellfixMatches {

	// oddly, "WHERE word MATCH "%s OR %s*" doesn't work very well here
	// hence the UNION
	return (&SpellfixMatches{
		Query: &Query{
			Text: `WITH CombinedResults AS (
		SELECT word, rank, distance
		FROM global_cats_spellfix
		WHERE word MATCH ?
		UNION ALL
		SELECT word, rank, distance
		FROM global_cats_spellfix
		WHERE word MATCH ? || '*'
			),
		RankedResults AS (
			SELECT 
				word, 
				rank,
				distance,
				ROW_NUMBER() OVER (PARTITION BY word ORDER BY distance) AS row_num
			FROM CombinedResults
		),
		TopResults AS (
			SELECT word, rank, distance
			FROM RankedResults
			WHERE row_num = 1
			AND distance <= ?
			ORDER BY distance, rank DESC
		)
		SELECT word, SUM(rank) as rank
		FROM TopResults
		GROUP BY LOWER(word)
		ORDER BY distance, rank DESC
		LIMIT ?;`,
			Args: []interface{}{
				snippet,
				snippet,
				SPELLFIX_DISTANCE_LIMIT,
				SPELLFIX_MATCHES_LIMIT,
			},
		},
	})
}

func (s *SpellfixMatches) OmitCats(cats []string) error {
	if len(cats) == 0 || cats[0] == "" {
		return e.ErrNoOmittedCats
	}

	// pop SPELLFIX_MATCHES_LIMIT arg
	s.Args = s.Args[0 : len(s.Args)-1]

	not_in_clause := `
	AND LOWER(word) NOT IN (?`
	s.Args = append(s.Args, cats[0])

	for i := 1; i < len(cats); i++ {
		not_in_clause += ", ?"
		s.Args = append(s.Args, cats[i])
	}
	not_in_clause += ")"

	distance_clause := `AND distance <= ?`
	s.Text = strings.Replace(
		s.Text,
		distance_clause,
		distance_clause+not_in_clause,
		1,
	)

	// push SPELLFIX_MATCHES_LIMIT arg back to end
	s.Args = append(s.Args, SPELLFIX_MATCHES_LIMIT)

	return nil
}
