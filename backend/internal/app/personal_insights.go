package app

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"
	_ "time/tzdata"

	"curated-backend/internal/contracts"
)

const (
	personalInsightsCompletionThreshold = 0.90
	personalInsightsDefaultLimit        = 10
	personalInsightsMaxLimit            = 25
	personalInsightsAttribution         = "full-per-entity"
)

var personalInsightsNow = time.Now

type personalInsightsWindow struct {
	rangeValue  contracts.PersonalInsightsRange
	from        string
	to          string
	timezone    string
	generatedAt string
	dataSince   *string
}

func (a *App) GetPersonalInsightsOverview(
	ctx context.Context,
	rawRange string,
	rawTimezone string,
) (contracts.PersonalInsightsOverviewDTO, error) {
	window, err := a.resolvePersonalInsightsWindow(ctx, rawRange, rawTimezone, personalInsightsNow())
	if err != nil {
		return contracts.PersonalInsightsOverviewDTO{}, err
	}
	aggregate, err := a.store.PersonalInsightsOverview(
		ctx,
		window.from,
		window.to,
		personalInsightsCompletionThreshold,
	)
	if err != nil {
		return contracts.PersonalInsightsOverviewDTO{}, err
	}
	var completionRate *float64
	if aggregate.StartedMovies > 0 {
		value := float64(aggregate.CompletedMovies) / float64(aggregate.StartedMovies)
		completionRate = &value
	}
	return contracts.PersonalInsightsOverviewDTO{
		Range:               window.rangeValue,
		From:                window.from,
		To:                  window.to,
		Timezone:            window.timezone,
		GeneratedAt:         window.generatedAt,
		DataSince:           copyPersonalInsightsString(window.dataSince),
		WatchedSeconds:      aggregate.WatchedSeconds,
		StartedMovies:       aggregate.StartedMovies,
		CompletedMovies:     aggregate.CompletedMovies,
		CompletionRate:      completionRate,
		CompletionThreshold: personalInsightsCompletionThreshold,
		RatedMovies:         aggregate.RatedMovies,
		AverageUserRating:   copyPersonalInsightsFloat(aggregate.AverageRating),
	}, nil
}

func (a *App) GetPersonalInsightsBreakdown(
	ctx context.Context,
	rawRange string,
	rawTimezone string,
	rawDimension string,
	limit int,
) (contracts.PersonalInsightsBreakdownDTO, error) {
	window, err := a.resolvePersonalInsightsWindow(ctx, rawRange, rawTimezone, personalInsightsNow())
	if err != nil {
		return contracts.PersonalInsightsBreakdownDTO{}, err
	}
	dimension, err := parsePersonalInsightsDimension(rawDimension)
	if err != nil {
		return contracts.PersonalInsightsBreakdownDTO{}, err
	}
	if limit == 0 {
		limit = personalInsightsDefaultLimit
	}
	if limit < 1 || limit > personalInsightsMaxLimit {
		return contracts.PersonalInsightsBreakdownDTO{}, fmt.Errorf(
			"%w: limit must be between 1 and %d",
			contracts.ErrPersonalInsightsInvalidLimit,
			personalInsightsMaxLimit,
		)
	}
	overview, err := a.store.PersonalInsightsOverview(
		ctx,
		window.from,
		window.to,
		personalInsightsCompletionThreshold,
	)
	if err != nil {
		return contracts.PersonalInsightsBreakdownDTO{}, err
	}
	rows, err := a.store.PersonalInsightsBreakdown(
		ctx,
		window.from,
		window.to,
		string(dimension),
		limit,
	)
	if err != nil {
		return contracts.PersonalInsightsBreakdownDTO{}, err
	}
	items := make([]contracts.PersonalInsightsBreakdownItemDTO, 0, len(rows))
	for _, row := range rows {
		share := 0.0
		if overview.WatchedSeconds > 0 {
			share = row.WatchedSeconds / overview.WatchedSeconds
		}
		if math.IsNaN(share) || math.IsInf(share, 0) || share < 0 {
			share = 0
		} else if share > 1 {
			share = 1
		}
		items = append(items, contracts.PersonalInsightsBreakdownItemDTO{
			Name:           row.Name,
			WatchedSeconds: row.WatchedSeconds,
			MovieCount:     row.MovieCount,
			ShareOfTotal:   share,
		})
	}
	return contracts.PersonalInsightsBreakdownDTO{
		Range:               window.rangeValue,
		Dimension:           dimension,
		From:                window.from,
		To:                  window.to,
		Timezone:            window.timezone,
		GeneratedAt:         window.generatedAt,
		DataSince:           copyPersonalInsightsString(window.dataSince),
		TotalWatchedSeconds: overview.WatchedSeconds,
		Attribution:         personalInsightsAttribution,
		Items:               items,
		Limit:               limit,
	}, nil
}

func (a *App) resolvePersonalInsightsWindow(
	ctx context.Context,
	rawRange string,
	rawTimezone string,
	now time.Time,
) (personalInsightsWindow, error) {
	rangeValue, days, err := parsePersonalInsightsRange(rawRange)
	if err != nil {
		return personalInsightsWindow{}, err
	}
	timezone := strings.TrimSpace(rawTimezone)
	if timezone == "" {
		timezone = "UTC"
	}
	if len(timezone) > 128 {
		return personalInsightsWindow{}, contracts.ErrPersonalInsightsInvalidTimezone
	}
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return personalInsightsWindow{}, fmt.Errorf(
			"%w: %q",
			contracts.ErrPersonalInsightsInvalidTimezone,
			timezone,
		)
	}
	year, month, day := now.In(location).Date()
	today := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	toDay := today.Format(time.DateOnly)
	dataSince, err := a.store.PersonalInsightsDataSince(ctx, toDay)
	if err != nil {
		return personalInsightsWindow{}, err
	}
	fromDay := toDay
	if rangeValue == contracts.PersonalInsightsRangeAll {
		if dataSince != nil {
			fromDay = *dataSince
		}
	} else {
		fromDay = today.AddDate(0, 0, -(days - 1)).Format(time.DateOnly)
	}
	return personalInsightsWindow{
		rangeValue:  rangeValue,
		from:        fromDay,
		to:          toDay,
		timezone:    timezone,
		generatedAt: now.UTC().Format(time.RFC3339Nano),
		dataSince:   copyPersonalInsightsString(dataSince),
	}, nil
}

func parsePersonalInsightsRange(raw string) (contracts.PersonalInsightsRange, int, error) {
	switch contracts.PersonalInsightsRange(strings.TrimSpace(raw)) {
	case contracts.PersonalInsightsRange30Days:
		return contracts.PersonalInsightsRange30Days, 30, nil
	case contracts.PersonalInsightsRange90Days:
		return contracts.PersonalInsightsRange90Days, 90, nil
	case contracts.PersonalInsightsRange365Days:
		return contracts.PersonalInsightsRange365Days, 365, nil
	case contracts.PersonalInsightsRangeAll:
		return contracts.PersonalInsightsRangeAll, 0, nil
	default:
		return "", 0, fmt.Errorf("%w: %q", contracts.ErrPersonalInsightsInvalidRange, raw)
	}
}

func parsePersonalInsightsDimension(raw string) (contracts.PersonalInsightsDimension, error) {
	switch contracts.PersonalInsightsDimension(strings.TrimSpace(raw)) {
	case contracts.PersonalInsightsDimensionActor:
		return contracts.PersonalInsightsDimensionActor, nil
	case contracts.PersonalInsightsDimensionStudio:
		return contracts.PersonalInsightsDimensionStudio, nil
	case contracts.PersonalInsightsDimensionTag:
		return contracts.PersonalInsightsDimensionTag, nil
	default:
		return "", fmt.Errorf("%w: %q", contracts.ErrPersonalInsightsInvalidDimension, raw)
	}
}

func copyPersonalInsightsString(value *string) *string {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
}

func copyPersonalInsightsFloat(value *float64) *float64 {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
}
