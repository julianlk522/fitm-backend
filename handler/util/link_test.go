package handler

import (
	"strings"
	"testing"

	"github.com/julianlk522/fitm/db"
	"github.com/julianlk522/fitm/model"
	"github.com/julianlk522/fitm/query"
)

func TestPrepareLinksResponse(t *testing.T) {
	var test_requests = []struct {
		LinksSQL   *query.TopLinks
		Page       int
		CatsParams string
		Valid      bool
	}{
		{
			LinksSQL:   query.NewTopLinks(),
			Page:       1,
			CatsParams: "",
			Valid:      true,
		},
		{
			LinksSQL:   query.NewTopLinks().FromCats([]string{"umvc3", "flowers"}).Page(1),
			Page:       1,
			CatsParams: "umvc3,flowers",
			Valid:      true,
		},
		{
			LinksSQL:   query.NewTopLinks().DuringPeriod("batman"),
			Page:       1,
			CatsParams: "",
			Valid:      false,
		},
		{
			LinksSQL: &query.TopLinks{
				Query: query.Query{
					Text: "spiderman",
				},
			},
			Page:       1,
			CatsParams: "",
			Valid:      false,
		},
	}

	for _, tr := range test_requests {
		_, err := PrepareLinksResponse[model.Link](tr.LinksSQL, tr.Page, tr.CatsParams)
		if tr.Valid && err != nil {
			t.Fatal(err)
		} else if !tr.Valid && err == nil {
			t.Fatalf("expected error for request %+v\n", tr)
		}
	}
}

func TestScanLinks(t *testing.T) {
	links_sql := query.NewTopLinks()
	// NewTopLinks().Error tested in query/link_test.go

	// signed out
	links_signed_out, err := ScanLinks[model.Link](links_sql)
	if err != nil {
		t.Fatal(err)
	} else if len(*links_signed_out) == 0 {
		t.Fatal("no links")
	}

	// signed in
	links_sql = links_sql.AsSignedInUser(test_req_user_id)
	links_signed_in, err := ScanLinks[model.LinkSignedIn](links_sql)
	if err != nil {
		t.Fatal(err)
	} else if len(*links_signed_in) == 0 {
		t.Fatal("no links")
	}
}

func TestPaginateLinks(t *testing.T) {

	// no links
	links_sql := query.NewTopLinks().FromCats([]string{"umvc3"}).DuringPeriod("day").Page(1)

	links, err := ScanLinks[model.Link](links_sql)
	if err != nil {
		t.Fatal(err)
	} else if len(*links) != 0 {
		t.Fatal("expected no links")
	}

	pl := PaginateLinks(links, 0)
	if pl.Links != nil {
		t.Fatal("expected no links")
	} else if pl.NextPage != -1 {
		t.Fatal("expected no more pages")
	}

	// single page
	links_sql = query.NewTopLinks().FromCats([]string{"umvc3", "flowers"}).Page(1)
	links, err = ScanLinks[model.Link](links_sql)
	if err != nil {
		t.Fatal(err)
	}

	pl = PaginateLinks(links, 1)
	if len(*pl.Links) == 0 {
		t.Fatal("expected links")
	} else if pl.NextPage != -1 {
		t.Fatal("expected no more pages")
	}

	// multiple pages
	links_sql = query.NewTopLinks().Page(1)

	links, err = ScanLinks[model.Link](links_sql)
	if err != nil {
		t.Fatal(err)
	} else if len(*links) == 0 {
		t.Fatal("expected links")
	}

	pl = PaginateLinks(links, 1)
	if len(*pl.Links) == 0 {
		t.Fatal("expected links")
	} else if pl.NextPage != 2 {
		t.Fatalf("expected next page to be 2, got %d (%d links)", pl.NextPage, len(*pl.Links))
	}
}

func TestCountMergedCatSpellingVariants(t *testing.T) {
	// no links; no merged cats
	test_cat := "nonexistentcat"
	links_sql := query.NewTopLinks().FromCats([]string{test_cat}).DuringPeriod("day").Page(1)
	links, err := ScanLinks[model.Link](links_sql)
	if err != nil {
		t.Fatal(err)
	}

	pl := PaginateLinks(links, 1)
	CountMergedCatSpellingVariants(pl, test_cat)
	if len(pl.MergedCats) != 0 {
		t.Fatal("expected no merged cats")
	}

	// 1 merged cat
	test_cat = "flower" // should merge "flowers"
	links_sql = query.NewTopLinks().FromCats([]string{test_cat})
	links, err = ScanLinks[model.Link](links_sql)
	if err != nil {
		t.Fatal(err)
	}

	pl = PaginateLinks(links, 1)
	CountMergedCatSpellingVariants(pl, test_cat)
	if len(pl.MergedCats) != 1 {
		t.Fatalf("expected 1 merged cat, got %d (%v)", len(pl.MergedCats), pl.MergedCats)
	}

	// multiple merged cats
	test_cats := []string{"flower", "tests"} // should merge "flowers" and "test"
	links_sql = query.NewTopLinks().FromCats(test_cats)
	links, err = ScanLinks[model.Link](links_sql)
	if err != nil {
		t.Fatal(err)
	}

	pl = PaginateLinks(links, 1)
	CountMergedCatSpellingVariants(pl, strings.Join(test_cats, ","))
	if len(pl.MergedCats) != 2 {
		t.Fatalf("expected 2 merged cats, got %d (%v)", len(pl.MergedCats), pl.MergedCats)
	}

	// inconsistent capitalization: should still merge
	test_cat = "FlOwEr" // should merge "flowers"
	links_sql = query.NewTopLinks().FromCats([]string{test_cat})
	links, err = ScanLinks[model.Link](links_sql)
	if err != nil {
		t.Fatal(err)
	}

	pl = PaginateLinks(links, 1)
	CountMergedCatSpellingVariants(pl, test_cat)
	if len(pl.MergedCats) != 1 {
		t.Fatalf("expected 1 merged cat, got %d (%v)", len(pl.MergedCats), pl.MergedCats)
	}

}

// Add link
func TestObtainURLMetaData(t *testing.T) {
	var test_requests = []struct {
		request *model.NewLinkRequest
		Valid   bool
	}{
		{&model.NewLinkRequest{NewLink: &model.NewLink{URL: "abc.com"}}, true},
		{&model.NewLinkRequest{NewLink: &model.NewLink{URL: "www.abc.com"}}, true},
		{&model.NewLinkRequest{NewLink: &model.NewLink{URL: "https://www.abc.com"}}, true},
		{&model.NewLinkRequest{NewLink: &model.NewLink{URL: "about.google.com"}}, true},
		{&model.NewLinkRequest{NewLink: &model.NewLink{URL: "julianlk.com/notreal"}}, false},
		{&model.NewLinkRequest{NewLink: &model.NewLink{URL: "gobblety gook"}}, false},
	}

	for _, tr := range test_requests {
		err := ObtainURLMetaData(tr.request)
		if tr.Valid && err != nil {
			t.Fatal(err)
		} else if !tr.Valid && err == nil {
			t.Fatalf("expected error for url %s", tr.request.NewLink.URL)
		}
	}
}

func TestGetResolvedURLResponse(t *testing.T) {
	var test_urls = []struct {
		URL   string
		Valid bool
	}{
		{"abc.com", true},
		{"www.abc.com", true},
		{"https://www.abc.com", true},
		{"about.google.com", true},
		{"julianlk.com/notreal", false},
		{"gobblety gook", false},
		// TODO: get the user agent headers to correctly apply and
		// add test case e.g., https://neal.fun/deep-sea
		// (responds with 403 if no user agent set)
	}

	for _, u := range test_urls {
		_, err := GetResolvedURLResponse(u.URL)
		if u.Valid && err != nil {
			t.Fatal(err)
		} else if !u.Valid && err == nil {
			t.Fatalf("expected error for url %s", u.URL)
		}
	}
}

func TestAssignMetadata(t *testing.T) {
	mock_metas := []HTMLMeta{
		// Auto Summary should be og:description,
		// og:image should be set
		{
			Title:         "title",
			Description:   "description",
			OGTitle:       "og:title",
			OGDescription: "og:description",
			OGImage:       "https://i.ytimg.com/vi/L4gaqVH0QHU/maxresdefault.jpg",
			OGAuthor:      "",
			OGPublisher:   "",
			OGSiteName:    "og:site_name",
		},
		// Auto Summary should be description
		{
			Title:         "",
			Description:   "description",
			OGTitle:       "",
			OGDescription: "",
			OGImage:       "",
			OGAuthor:      "",
			OGSiteName:    "",
			OGPublisher:   "",
		},
		// Auto Summary should be og:title
		{
			Title:         "title",
			Description:   "",
			OGTitle:       "og:title",
			OGDescription: "",
			OGImage:       "",
			OGAuthor:      "",
			OGSiteName:    "",
			OGPublisher:   "",
		},
		// Auto Summary should be title
		{
			Title:         "title",
			Description:   "",
			OGTitle:       "",
			OGDescription: "",
			OGImage:       "",
			OGAuthor:      "",
			OGSiteName:    "test",
			OGPublisher:   "",
		},
		// Auto Summary should be test
		// og:image should be set
		{
			Title:         "",
			Description:   "",
			OGTitle:       "",
			OGDescription: "",
			OGImage:       "https://i.ytimg.com/vi/XdfoXdzGmr0/maxresdefault.jpg",
			OGAuthor:      "",
			OGSiteName:    "test",
			OGPublisher:   "",
		},
	}

	for i, meta := range mock_metas {
		mock_request := &model.NewLinkRequest{
			NewLink: &model.NewLink{
				URL:     "",
				Cats:    "",
				Summary: "",
			},
		}

		AssignMetadata(meta, mock_request)

		switch i {
		case 0:
			if mock_request.AutoSummary != "og:description" {
				t.Fatalf("og:description provided but auto summary set to: %s", mock_request.AutoSummary)
			} else if mock_request.ImgURL != "https://i.ytimg.com/vi/L4gaqVH0QHU/maxresdefault.jpg" {
				t.Fatal("expected og:image to be set")
			}
		case 1:
			if mock_request.AutoSummary != "description" {
				t.Fatalf("description provided but auto summary set to: %s", mock_request.AutoSummary)
			}
		case 2:
			if mock_request.AutoSummary != "og:title" {
				t.Fatalf("og:title provided but auto summary set to: %s", mock_request.AutoSummary)
			}
		case 3:
			if mock_request.AutoSummary != "title" {
				t.Fatalf("title provided but auto summary set to: %s", mock_request.AutoSummary)
			}
		case 4:
			if mock_request.AutoSummary != "test" {
				t.Fatalf("og:sitename provided but auto summary set to: %s", mock_request.AutoSummary)
			} else if mock_request.ImgURL != "https://i.ytimg.com/vi/XdfoXdzGmr0/maxresdefault.jpg" {
				t.Fatal("expected og:image to be set")
			}
		default:
			t.Fatal("unhandled case, you f'ed up dawg")
		}
	}
}

func TestLinkAlreadyAdded(t *testing.T) {
	var test_urls = []struct {
		URL   string
		Added bool
	}{
		{"https://stackoverflow.co/", true},
		{"https://www.ronjarzombek.com", true},
		{"https://somethingnotonfitm", false},
		{"jimminy jillickers", false},
	}

	for _, u := range test_urls {
		added, _ := LinkAlreadyAdded(u.URL)
		if u.Added && !added {
			t.Fatalf("expected url %s to be added", u.URL)
		} else if !u.Added && added {
			t.Fatalf("%s NOT added, expected error", u.URL)
		}
	}
}

func TestIncrementSpellfixRanksForCats(t *testing.T) {
	var test_cats = []struct {
		Cats         []string
		CurrentRanks []int
	}{
		{
			[]string{"umvc3"},
			[]int{4},
		},
		{
			[]string{"flowers", "nerd"},
			[]int{6, 1},
		},
		// cat doesn't exist: should be added to global_cats_spellfix
		{
			[]string{"jksfdkhsdf"},
			[]int{0},
		},
	}

	for _, tc := range test_cats {
		err := IncrementSpellfixRanksForCats(nil, tc.Cats)
		if err != nil {
			t.Fatal(err)
		}

		for i, cat := range tc.Cats {
			var rank int
			err := db.Client.QueryRow(
				"SELECT rank FROM global_cats_spellfix WHERE word = ?", cat,
			).Scan(&rank)

			if err != nil {
				t.Fatal(err)
			} else if rank != tc.CurrentRanks[i]+1 {
				t.Fatal(
					"expected rank for", cat, "to be", tc.CurrentRanks[i]+1, "got", rank,
				)
			}
		}
	}
}

// Delete link
func TestDecrementSpellfixRanksForCats(t *testing.T) {
	var test_cats = []struct {
		Cats         []string
		CurrentRanks []int
	}{
		{
			[]string{"test"},
			[]int{11},
		},
		{
			[]string{"coding", "hacking"},
			[]int{7, 2},
		},
	}

	for _, tc := range test_cats {
		err := DecrementSpellfixRanksForCats(nil, tc.Cats)
		if err != nil {
			t.Fatal(err)
		}

		for i, cat := range tc.Cats {
			var rank int
			err := db.Client.QueryRow(
				"SELECT rank FROM global_cats_spellfix WHERE word = ?", cat,
			).Scan(&rank)

			if err != nil {
				t.Fatal(err)
			} else if rank != tc.CurrentRanks[i]-1 {
				t.Fatal(
					"expected rank for", cat, "to be", tc.CurrentRanks[i]-1, "got", rank,
				)
			}
		}
	}
}

// Like / unlike link
func TestUserSubmittedLink(t *testing.T) {
	var test_links = []struct {
		ID                  string
		SubmittedByTestUser bool
	}{
		// user jlk submitted links with ID 7, 13, 23
		// (not 0, 1, or 86)
		{"7", true},
		{"13", true},
		{"23", true},
		{"0", false},
		{"1", false},
		{"86", false},
	}

	for _, l := range test_links {
		return_true := UserSubmittedLink(test_login_name, l.ID)
		if l.SubmittedByTestUser && !return_true {
			t.Fatalf("expected link %s to be submitted by user", l.ID)
		} else if !l.SubmittedByTestUser && return_true {
			t.Fatalf("%s NOT submitted by user, expected error", l.ID)
		}
	}
}

func TestUserHasLikedLink(t *testing.T) {
	var test_links = []struct {
		ID              string
		LikedByTestUser bool
	}{
		// user jlk liked links with ID 24, 32, 103
		// (not 9, 11, or 15)
		{"24", true},
		{"32", true},
		{"103", true},
		{"9", false},
		{"11", false},
		{"15", false},
	}

	for _, l := range test_links {
		return_true := UserHasLikedLink(test_user_id, l.ID)
		if l.LikedByTestUser && !return_true {
			t.Fatalf("expected link %s to be liked by user", l.ID)
		} else if !l.LikedByTestUser && return_true {
			t.Fatalf("%s NOT liked by user, expected error", l.ID)
		}
	}
}

// Copy link
func TestUserHasCopiedLink(t *testing.T) {
	var test_links = []struct {
		ID               string
		CopiedByTestUser bool
	}{
		// test user jlk copied links with ID 19, 31, 32
		// (not 0, 1, or 104)
		{"19", true},
		{"31", true},
		{"32", true},
		{"0", false},
		{"1", false},
		{"104", false},
	}

	for _, l := range test_links {
		return_true := UserHasCopiedLink(test_user_id, l.ID)
		if l.CopiedByTestUser && !return_true {
			t.Fatalf("expected link %s to be copied by user", l.ID)
		} else if !l.CopiedByTestUser && return_true {
			t.Fatalf("%s NOT copied by user, expected error", l.ID)
		}
	}
}
