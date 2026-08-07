package metatube

import (
	"testing"

	"github.com/metatube-community/metatube-sdk-go/model"
)

func TestEffectivePosterURLs_BigFieldsFallback(t *testing.T) {
	t.Parallel()
	c, th := effectivePosterURLs(&model.MovieInfo{
		BigCoverURL: "https://example.com/big-cover.jpg",
		BigThumbURL: "https://example.com/big-thumb.jpg",
	}, nil)
	if c != "https://example.com/big-cover.jpg" || th != "https://example.com/big-thumb.jpg" {
		t.Fatalf("cover=%q thumb=%q", c, th)
	}
}

func TestEffectivePosterURLs_MutualFallback(t *testing.T) {
	t.Parallel()
	c, th := effectivePosterURLs(&model.MovieInfo{CoverURL: "https://example.com/c.jpg"}, nil)
	if c != "https://example.com/c.jpg" || th != "https://example.com/c.jpg" {
		t.Fatalf("cover=%q thumb=%q want same URL", c, th)
	}
}

func TestEffectivePosterURLs_Nil(t *testing.T) {
	t.Parallel()
	c, th := effectivePosterURLs(nil, nil)
	if c != "" || th != "" {
		t.Fatalf("cover=%q thumb=%q", c, th)
	}
}

func TestEffectivePosterURLs_PrefersDistinctSearchThumbOverDuplicatedDetailCover(t *testing.T) {
	t.Parallel()
	cover := "https://example.com/wide-cover.jpg"
	c, th := effectivePosterURLs(
		&model.MovieInfo{CoverURL: cover, ThumbURL: cover},
		&model.MovieSearchResult{
			CoverURL: cover,
			ThumbURL: "https://example.com/portrait-card.jpg",
		},
	)
	if c != cover || th != "https://example.com/portrait-card.jpg" {
		t.Fatalf("cover=%q thumb=%q", c, th)
	}
}

func TestEffectivePosterURLs_UsesSearchThumbWhenDetailThumbMissing(t *testing.T) {
	t.Parallel()
	c, th := effectivePosterURLs(
		&model.MovieInfo{CoverURL: "https://example.com/wide-cover.jpg"},
		&model.MovieSearchResult{ThumbURL: "https://example.com/portrait-card.jpg"},
	)
	if c != "https://example.com/wide-cover.jpg" || th != "https://example.com/portrait-card.jpg" {
		t.Fatalf("cover=%q thumb=%q", c, th)
	}
}
