package handler

import (
	"math"

	"github.com/julianlk522/fitm/db"

	"database/sql"
	"slices"
	"strings"

	e "github.com/julianlk522/fitm/error"
	"github.com/julianlk522/fitm/model"
	"github.com/julianlk522/fitm/query"
)

const TMAP_CATS_PAGE_LIMIT int = 12

// GetTreasureMap
func UserExists(login_name string) (bool, error) {
	var u sql.NullString
	err := db.Client.QueryRow("SELECT id FROM Users WHERE login_name = ?;", login_name).Scan(&u)
	if err == sql.ErrNoRows {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return true, nil
}

func BuildTmapFromOpts[T model.TmapLink | model.TmapLinkSignedIn](opts *model.TmapOptions) (interface{}, error) {
	if opts.OwnerLoginName == "" {
		return nil, e.ErrNoTmapOwnerLoginName
	}
	tmap_owner := opts.OwnerLoginName

	nsfw_links_count_sql := query.NewTmapNSFWLinksCount(tmap_owner)

	has_cat_filter := len(opts.CatsFilter) > 0
	var profile *model.Profile
	if has_cat_filter {
		nsfw_links_count_sql = nsfw_links_count_sql.FromCats(opts.CatsFilter)
		// add profile only if unfiltered
	} else {
		var err error
		profile_sql := query.NewTmapProfile(tmap_owner)
		profile, err = ScanTmapProfile(profile_sql)
		if err != nil {
			return nil, err
		}
	}

	var nsfw_links_count int

	// single section
	if opts.Section != "" {
		var links *[]T
		var err error

		switch opts.Section {
		case "submitted":
			links, err = ScanTmapLinks[T](query.NewTmapSubmitted(tmap_owner).FromOptions(opts).Query)
			nsfw_links_count_sql = nsfw_links_count_sql.SubmittedOnly()
		case "copied":
			links, err = ScanTmapLinks[T](query.NewTmapCopied(tmap_owner).FromOptions(opts).Query)
			nsfw_links_count_sql = nsfw_links_count_sql.CopiedOnly()
		case "tagged":
			links, err = ScanTmapLinks[T](query.NewTmapTagged(tmap_owner).FromOptions(opts).Query)
			nsfw_links_count_sql = nsfw_links_count_sql.TaggedOnly()
		default:
			return nil, e.ErrInvalidSectionParams
		}

		if err != nil {
			return nil, err
		}

		if links == nil || len(*links) == 0 {
			return model.PaginatedTmapSection[T]{
				Links:          &[]T{},
				Cats:           &[]model.CatCount{},
				NSFWLinksCount: 0,
				NextPage:       -1,
			}, nil
		}

		var cat_counts *[]model.CatCount
		if has_cat_filter {
			cat_counts = GetCatCountsFromTmapLinks(links, &model.TmapCatCountsOptions{
				RawCatsParams: opts.RawCatsParams,
			})
		} else {
			cat_counts = GetCatCountsFromTmapLinks(links, nil)
		}

		// Paginate section links
		// due to counting cats manually (i.e., not in SQL) the pagination
		// must also be done manually after retrieving the full slice of links
		// and counting cats
		page, next_page := 1, -1
		if opts.Page < 0 {
			return nil, e.ErrInvalidPageParams
		} else if opts.Page > 0 {
			page = opts.Page
		}

		total_pages := int(math.Ceil(float64(len(*links)) / float64(query.LINKS_PAGE_LIMIT)))
		if page > total_pages {
			links = &[]T{}
		} else if page == total_pages {
			*links = (*links)[query.LINKS_PAGE_LIMIT*(page-1):]
		} else {
			*links = (*links)[query.LINKS_PAGE_LIMIT*(page-1) : query.LINKS_PAGE_LIMIT*page]
			next_page = page + 1
		}

		if err := db.Client.QueryRow(nsfw_links_count_sql.Text, nsfw_links_count_sql.Args...).Scan(&nsfw_links_count); err != nil {
			return nil, err
		}

		return model.PaginatedTmapSection[T]{
			Links:          links,
			Cats:           cat_counts,
			NextPage:       next_page,
			NSFWLinksCount: nsfw_links_count,
		}, nil

		// all sections
	} else {
		submitted, err := ScanTmapLinks[T](query.NewTmapSubmitted(tmap_owner).FromOptions(opts).Query)
		if err != nil {
			return nil, err
		}
		copied, err := ScanTmapLinks[T](query.NewTmapCopied(tmap_owner).FromOptions(opts).Query)
		if err != nil {
			return nil, err
		}
		tagged, err := ScanTmapLinks[T](query.NewTmapTagged(tmap_owner).FromOptions(opts).Query)
		if err != nil {
			return nil, err
		}

		all_links := slices.Concat(*submitted, *copied, *tagged)
		if len(all_links) == 0 {
			return model.FilteredTmap[T]{
				TmapSections:   &model.TmapSections[T]{},
				NSFWLinksCount: 0,
			}, nil
		}

		var cat_counts *[]model.CatCount
		if has_cat_filter {
			cat_counts = GetCatCountsFromTmapLinks(&all_links, &model.TmapCatCountsOptions{
				RawCatsParams: opts.RawCatsParams,
			})
		} else {
			cat_counts = GetCatCountsFromTmapLinks(&all_links, nil)
		}

		// limit sections to top 20 links
		// 20+ links: indicate in response so can be paginated
		var sections_with_more []string
		if len(*submitted) > query.LINKS_PAGE_LIMIT {
			sections_with_more = append(sections_with_more, "submitted")
			*submitted = (*submitted)[0:query.LINKS_PAGE_LIMIT]
		}
		if len(*copied) > query.LINKS_PAGE_LIMIT {
			sections_with_more = append(sections_with_more, "copied")
			*copied = (*copied)[0:query.LINKS_PAGE_LIMIT]
		}
		if len(*tagged) > query.LINKS_PAGE_LIMIT {
			sections_with_more = append(sections_with_more, "tagged")
			*tagged = (*tagged)[0:query.LINKS_PAGE_LIMIT]
		}

		sections := &model.TmapSections[T]{
			Submitted:        submitted,
			Copied:           copied,
			Tagged:           tagged,
			SectionsWithMore: sections_with_more,
			Cats:             cat_counts,
		}

		if err := db.Client.QueryRow(nsfw_links_count_sql.Text, nsfw_links_count_sql.Args...).Scan(&nsfw_links_count); err != nil {
			return nil, err
		}

		if has_cat_filter {
			return model.FilteredTmap[T]{
				TmapSections:   sections,
				NSFWLinksCount: nsfw_links_count,
			}, nil

		} else {
			return model.Tmap[T]{
				Profile:        profile,
				TmapSections:   sections,
				NSFWLinksCount: nsfw_links_count,
			}, nil
		}
	}
}

func ScanTmapProfile(sql *query.TmapProfile) (*model.Profile, error) {
	var u model.Profile
	err := db.Client.
		QueryRow(sql.Text, sql.Args...).
		Scan(
			&u.LoginName,
			&u.About,
			&u.PFP,
			&u.Created,
		)
	if err != nil {
		return nil, e.ErrNoUserWithLoginName
	}

	return &u, nil
}

func ScanTmapLinks[T model.TmapLink | model.TmapLinkSignedIn](sql *query.Query) (*[]T, error) {
	rows, err := db.Client.Query(sql.Text, sql.Args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links interface{}

	switch any(new(T)).(type) {
	case *model.TmapLink:
		var signed_out_links = []model.TmapLink{}

		for rows.Next() {
			l := model.TmapLink{}
			err := rows.Scan(
				&l.ID,
				&l.URL,
				&l.SubmittedBy,
				&l.SubmitDate,
				&l.Cats,
				&l.CatsFromUser,
				&l.Summary,
				&l.SummaryCount,
				&l.LikeCount,
				&l.CopyCount,
				&l.ClickCount,
				&l.TagCount,
				&l.ImgURL)
			if err != nil {
				return nil, err
			}
			signed_out_links = append(signed_out_links, l)
		}

		links = &signed_out_links
	case *model.TmapLinkSignedIn:
		var signed_in_links = []model.TmapLinkSignedIn{}

		for rows.Next() {
			l := model.TmapLinkSignedIn{}
			err := rows.Scan(
				&l.ID,
				&l.URL,
				&l.SubmittedBy,
				&l.SubmitDate,
				&l.Cats,
				&l.CatsFromUser,
				&l.Summary,
				&l.SummaryCount,
				&l.LikeCount,
				&l.CopyCount,
				&l.ClickCount,
				&l.TagCount,
				&l.ImgURL,

				// signed-in only properties
				&l.IsLiked,
				&l.IsCopied)
			if err != nil {
				return nil, err
			}
			signed_in_links = append(signed_in_links, l)
		}

		links = &signed_in_links
	}

	return links.(*[]T), nil
}

func GetCatCountsFromTmapLinks[T model.TmapLink | model.TmapLinkSignedIn](links *[]T, opts *model.TmapCatCountsOptions) *[]model.CatCount {
	var omitted_cats []string
	// Use raw cats params here to determine omitted_cats because CatsFilter
	// (from BuildTmapFromOpts) is modified to escape reserved chars and
	// include singular/plural spelling variations. To correctly count cats
	// (omitting ones passed in the request), omitted_cats must _not_ have
	// these modifications applied.

	// Use lowercase so that capitalization variants of cat filters
	// are still not counted
	if opts != nil && opts.RawCatsParams != "" {
		omitted_cats = strings.Split(strings.ToLower(opts.RawCatsParams), ",")
	}
	has_cat_filter := len(omitted_cats) > 0

	counts := []model.CatCount{}
	all_found_cats := []string{}
	var found bool

	for _, link := range *links {
		var cats string
		switch l := any(link).(type) {
		case model.TmapLink:
			cats = l.Cats
		case model.TmapLinkSignedIn:
			cats = l.Cats
		}

		link_found_cats := []string{}

		for _, cat := range strings.Split(cats, ",") {
			lc_cat := strings.ToLower(cat)

			if strings.TrimSpace(cat) == "" || slices.ContainsFunc(link_found_cats, func(c string) bool { return strings.ToLower(c) == lc_cat }) || (has_cat_filter &&
				slices.Contains(omitted_cats, lc_cat)) {
				continue
			}

			link_found_cats = append(link_found_cats, cat)

			found = false
			for _, found_cat := range all_found_cats {
				if found_cat == cat {
					found = true

					for i, count := range counts {
						if count.Category == cat {
							counts[i].Count++
							break
						}
					}
				}
			}

			if !found {
				counts = append(counts, model.CatCount{Category: cat, Count: 1})
				all_found_cats = append(all_found_cats, cat)
			}
		}
	}

	slices.SortFunc(counts, model.SortCats)

	if has_cat_filter {
		MergeCatCountsCapitalizationVariants(&counts, omitted_cats)
	}

	if len(counts) > TMAP_CATS_PAGE_LIMIT {
		counts = (counts)[:TMAP_CATS_PAGE_LIMIT]
	}

	return &counts
}

// merge counts of capitalization variants e.g. "Music" and "music"
func MergeCatCountsCapitalizationVariants(counts *[]model.CatCount, omitted_cats []string) {
	for i, count := range *counts {
		for j := i + 1; j < len(*counts); j++ {
			if strings.EqualFold(count.Category, (*counts)[j].Category) {

				// skip if is some capitalization variant of a cat filter
				if len(omitted_cats) > 0 && slices.Contains(omitted_cats, strings.ToLower((*counts)[j].Category)) {
					continue
				}
				(*counts)[i].Count += (*counts)[j].Count
				*counts = append((*counts)[:j], (*counts)[j+1:]...)
			}
		}
	}
}
