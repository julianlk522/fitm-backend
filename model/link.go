package model

import (
	"net/http"
	"strings"

	e "github.com/julianlk522/fitm/error"

	util "github.com/julianlk522/fitm/model/util"

	"github.com/google/uuid"
)

type YTVideoMetaData struct {
	Items []YTVideoItems `json:"items"`
}

type YTVideoItems struct {
	Snippet YTVideoSnippet `json:"snippet"`
}

type YTVideoSnippet struct {
	Title      string `json:"title"`
	Thumbnails struct {
		Default struct {
			URL string `json:"url"`
		} `json:"default"`
	}
}

type HasCats interface {
	Link | LinkSignedIn

	GetCats() string
}

type Link struct {
	ID           string
	URL          string
	SubmittedBy  string
	SubmitDate   string
	Cats         string
	Summary      string
	SummaryCount int
	LikeCount    int64
	CopyCount    int
	ClickCount   int64
	TagCount     int
	ImgURL       string
}

func (l Link) GetCats() string {
	return l.Cats
}

type LinkSignedIn struct {
	Link
	IsLiked  bool
	IsCopied bool
}

func (lsi LinkSignedIn) GetCats() string {
	return lsi.Cats
}

type PaginatedLinks[T Link | LinkSignedIn] struct {
	Links      *[]T
	MergedCats []string
	NextPage   int
}

type Contributor struct {
	LoginName      string
	LinksSubmitted int
}

type NewLink struct {
	URL     string `json:"url"`
	Cats    string `json:"cats"`
	Summary string `json:"summary,omitempty"`
}

type NewLinkRequest struct {
	*NewLink
	ID         string
	SubmitDate string
	LikeCount  int64

	// to be assigned by handler
	URL          string // potentially modified after test request(s)
	SubmittedBy  string
	Cats         string // potentially modified after sort
	AutoSummary  string
	SummaryCount int
	ImgURL       string
}

func (nlr *NewLinkRequest) Bind(r *http.Request) error {
	if nlr.NewLink.URL == "" {
		return e.ErrNoURL
	} else if len(nlr.NewLink.URL) > util.URL_CHAR_LIMIT {
		return e.ErrLinkURLCharsExceedLimit(util.URL_CHAR_LIMIT)
	}

	switch {
	case nlr.NewLink.Cats == "":
		return e.ErrNoTagCats
	case util.HasTooLongCats(nlr.NewLink.Cats):
		return e.CatCharsExceedLimit(util.CAT_CHAR_LIMIT)
	case util.HasTooManyCats(nlr.NewLink.Cats):
		return e.NumCatsExceedsLimit(util.NUM_CATS_LIMIT)
	case util.HasDuplicateCats(nlr.NewLink.Cats):
		return e.ErrDuplicateCats
	}

	if len(nlr.NewLink.Summary) > util.SUMMARY_CHAR_LIMIT {
		return e.SummaryLengthExceedsLimit(util.SUMMARY_CHAR_LIMIT)
	}

	if strings.Contains(nlr.NewLink.Summary, "\"") {
		nlr.NewLink.Summary = strings.ReplaceAll(nlr.NewLink.Summary, "\"", "'")
	}

	nlr.ID = uuid.New().String()
	nlr.SubmitDate = util.NEW_LONG_TIMESTAMP()
	nlr.LikeCount = 0

	return nil
}

type DeleteLinkRequest struct {
	LinkID string `json:"link_id"`
}

func (dlr *DeleteLinkRequest) Bind(r *http.Request) error {
	if dlr.LinkID == "" {
		return e.ErrNoLinkID
	}

	return nil
}

type NewClickRequest struct {
	LinkID string `json:"link_id"`
	IPAddr string
	Timestamp string
}

func (ncr *NewClickRequest) Bind(r *http.Request) error {
	if ncr.LinkID == "" {
		return e.ErrNoLinkID
	}

	ncr.Timestamp = util.NEW_LONG_TIMESTAMP()
	
	return nil
}
