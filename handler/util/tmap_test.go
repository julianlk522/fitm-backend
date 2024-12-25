package handler

import (
	"slices"
	"strings"
	"testing"

	"github.com/julianlk522/fitm/model"
	"github.com/julianlk522/fitm/query"
)

func TestUserExists(t *testing.T) {
	var test_login_names = []struct {
		login_name string
		Exists     bool
	}{
		{"johndoe", false},
		{"janedoe", false},
		{test_login_name, true},
	}

	for _, l := range test_login_names {
		return_true, err := UserExists(l.login_name)
		if err != nil {
			t.Fatalf("failed with error: %s", err)
		} else if l.Exists && !return_true {
			t.Fatalf("expected user %s to exist", l.login_name)
		} else if !l.Exists && return_true {
			t.Fatalf("user %s does not exist", l.login_name)
		}
	}
}

func TestBuildTmapFromOpts(t *testing.T) {
	var test_data = []struct {
		LoginName        string
		RequestingUserID string
		CatsParams       string
		SortByNewest     bool
		IncludeNSFW      bool
		SectionParams    string
		PageParams       int
		Valid            bool
	}{
		{test_login_name, test_user_id, "", false, false, "", 1, true},
		{test_login_name, test_req_user_id, "", false, true, "", 1, true},
		{test_login_name, "", "", false, true, "", 1, true},
		{test_login_name, test_user_id, "umvc3", true, true, "", 1, true},
		{test_login_name, test_req_user_id, "", true, false, "", 0, true},
		{test_login_name, "", "", false, false, "", 10, true},
		{test_login_name, test_user_id, "umvc3,flowers", true, true, "", 1, true},
		{test_login_name, "", "umvc3,flowers", true, false, "", 2, true},
		{test_login_name, "", "umvc3,flowers", false, true, "", 1, true},
		{test_login_name, "", "umvc3,flowers", false, true, "submitted", 4, true},
		{test_login_name, "", "umvc3,flowers", false, true, "copied", 0, true},
		// "notasection" is invalid
		{test_login_name, "", "umvc3,flowers", false, true, "notasection", 1, false},
		// negative page is invalid
		{test_login_name, "", "", false, true, "submitted", -1, false},
	}

	for _, td := range test_data {
		var opts = &model.TmapOptions{
			OwnerLoginName: td.LoginName,
			RawCatsParams:  td.CatsParams,
			AsSignedInUser: td.RequestingUserID,
			SortByNewest:   td.SortByNewest,
			IncludeNSFW:    td.IncludeNSFW,
			Section:        td.SectionParams,
			Page:           td.PageParams,
		}

		if td.CatsParams != "" {
			cats := strings.Split(td.CatsParams, ",")
			query.EscapeCatsReservedChars(cats)
			cats = query.GetCatsOptionalPluralOrSingularForms(cats)
			opts.CatsFilter = cats
		}

		var tmap interface{}
		var err error

		if td.RequestingUserID != "" {
			tmap, err = BuildTmapFromOpts[model.TmapLinkSignedIn](opts)
		} else {
			tmap, err = BuildTmapFromOpts[model.TmapLink](opts)
		}

		if err != nil && td.Valid {
			t.Fatalf("error %s for opts %+v", err, opts)
		} else if err == nil && !td.Valid {
			t.Fatalf("expected error for opts %+v", opts)
		}

		if !td.Valid {
			continue
		}

		// verify type and filtered
		var is_filtered bool
		switch tmap.(type) {
		case model.Tmap[model.TmapLink], model.Tmap[model.TmapLinkSignedIn]:
			is_filtered = false
		case model.FilteredTmap[model.TmapLink], model.FilteredTmap[model.TmapLinkSignedIn]:
			is_filtered = true
		case model.PaginatedTmapSection[model.TmapLink], model.PaginatedTmapSection[model.TmapLinkSignedIn]:
			continue
		}

		if is_filtered && td.CatsParams == "" {
			t.Fatalf("expected unfiltered treasure map type, got %T", tmap)
		} else if !is_filtered && td.CatsParams != "" {
			t.Fatalf("expected filtered treasure map type, got %T (request params: %+v)", tmap, td)
		}
	}
}

func TestScanTmapProfile(t *testing.T) {
	profile_sql := query.NewTmapProfile(test_login_name)
	// NewTmapProfile() tested in query/tmap_test.go

	profile, err := ScanTmapProfile(profile_sql)
	if err != nil {
		t.Fatal(err)
	}

	if profile.LoginName != test_login_name {
		t.Fatalf(
			"expected %s, got %s", test_login_name,
			profile.LoginName,
		)
	}

	if profile.Created != "2024-04-10T03:48:09Z" {
		t.Fatalf(
			"expected %s, got %s", "2024-04-10T03:48:09Z",
			profile.Created,
		)
	}
}

func TestScanTmapLinks(t *testing.T) {
	var test_requests = []struct {
		LoginName        string
		RequestingUserID string
	}{
		{test_login_name, test_user_id},
		{test_login_name, test_req_user_id},
		{test_login_name, ""},
	}

	for _, r := range test_requests {
		submitted_sql := query.NewTmapSubmitted(r.LoginName)
		copied_sql := query.NewTmapCopied(r.LoginName)
		tagged_sql := query.NewTmapTagged(r.LoginName)

		if r.RequestingUserID != "" {
			submitted_sql = submitted_sql.AsSignedInUser(r.RequestingUserID)
			copied_sql = copied_sql.AsSignedInUser(r.RequestingUserID)
			tagged_sql = tagged_sql.AsSignedInUser(r.RequestingUserID)

			_, err := ScanTmapLinks[model.TmapLinkSignedIn](submitted_sql.Query)
			if err != nil {
				t.Fatalf(
					"failed scanning tmap submitted links (signed-in) with error: %s",
					err,
				)
			}
			_, err = ScanTmapLinks[model.TmapLinkSignedIn](copied_sql.Query)
			if err != nil {
				t.Fatalf(
					"failed scanning tmap copied links (signed-in) with error: %s",
					err,
				)
			}
			_, err = ScanTmapLinks[model.TmapLinkSignedIn](tagged_sql.Query)
			if err != nil {
				t.Fatalf(
					"failed scanning tmap tagged links (signed-in) with error: %s",
					err,
				)
			}
		} else {
			_, err := ScanTmapLinks[model.TmapLink](submitted_sql.Query)
			if err != nil {
				t.Fatalf(
					"failed scanning tmap submitted links (no auth) with error: %s",
					err,
				)
			}
			_, err = ScanTmapLinks[model.TmapLink](copied_sql.Query)
			if err != nil {
				t.Fatalf(
					"failed scanning tmap copied links (no auth) with error: %s",
					err,
				)
			}
			_, err = ScanTmapLinks[model.TmapLink](tagged_sql.Query)
			if err != nil {
				t.Fatalf(
					"failed scanning tmap tagged links (no auth) with error: %s",
					err,
				)
			}
		}
	}
}

func TestGetCatCountsFromTmapLinks(t *testing.T) {
	tmap, err := BuildTmapFromOpts[model.TmapLink](&model.TmapOptions{
		OwnerLoginName: "xyz",
	})
	if err != nil {
		t.Fatalf("failed with error %s", err)
	}

	var all_links interface{}

	switch tmap.(type) {
	case model.Tmap[model.TmapLink]:
		all_links = slices.Concat(
			*tmap.(model.Tmap[model.TmapLink]).Submitted,
			*tmap.(model.Tmap[model.TmapLink]).Copied,
			*tmap.(model.Tmap[model.TmapLink]).Tagged,
		)
		l, ok := all_links.([]model.TmapLink)
		if !ok {
			t.Fatalf("unexpected type %T", all_links)
		}

		// no omitted cats
		var unfiltered_test_cat_counts = []struct {
			Cat   string
			Count int32
		}{
			{"test", 2},
			// tag has cats "flowers" and "Flowers": tests that tags with
			// capitalization variant duplicates are only counted once still
			{"flowers", 1},
		}

		cat_counts := GetCatCountsFromTmapLinks(&l, nil)
		for _, count := range *cat_counts {
			for _, test_count := range unfiltered_test_cat_counts {
				if count.Category == test_count.Cat && count.Count != test_count.Count {
					t.Fatalf(
						"expected count %d for cat %s, got %d",
						test_count.Count,
						test_count.Cat,
						count.Count,
					)
				}
			}
		}

		// empty omitted cats
		// (should never happen, but should behave as if no omitted cats were passed)
		cat_counts = GetCatCountsFromTmapLinks(
			&l,
			&model.TmapCatCountsOptions{
				RawCatsParams: "",
			},
		)

		for _, count := range *cat_counts {
			for _, test_count := range unfiltered_test_cat_counts {
				if count.Category == test_count.Cat && count.Count != test_count.Count {
					t.Fatalf(
						"expected count %d for cat %s, got %d",
						test_count.Count,
						test_count.Cat,
						count.Count,
					)
				}
			}
		}

		// omitted cats
		var filtered_test_cat_counts = []struct {
			Cat   string
			Count int32
		}{
			{"test", 0},
			{"flowers", 1},
		}

		cat_counts = GetCatCountsFromTmapLinks(
			&l,
			&model.TmapCatCountsOptions{
				RawCatsParams: "test",
			},
		)
		for _, count := range *cat_counts {
			for _, test_count := range filtered_test_cat_counts {
				if count.Category == test_count.Cat && count.Count != test_count.Count {
					t.Fatalf(
						"expected count %d for cat %s, got %d",
						test_count.Count,
						test_count.Cat,
						count.Count,
					)
				}
			}
		}
	default:
		t.Fatalf("unexpected tmap type %T", tmap)
	}
}

func TestMergeCatCountsCapitalizationVariants(t *testing.T) {
	var counts = []model.CatCount{
		{Category: "Music", Count: 1},
		{Category: "music", Count: 1},
		{Category: "musica", Count: 1},
		{Category: "musics", Count: 1},
		{Category: "FITM", Count: 5},
		{Category: "fitm", Count: 5},
	}
	MergeCatCountsCapitalizationVariants(&counts, nil)
	if counts[0].Count != 2 {
		t.Fatalf("expected count 2, got %d", counts[0].Count)
	}

	if counts[3].Category != "FITM" {
		t.Fatalf("expected FITM to have moved up to index 3 because Music and music were combined, got %s", counts[3].Category)
	}
}
