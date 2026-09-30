package app

import (
	"context"
	"curated-backend/internal/contracts"
	"hash/fnv"
	"sort"
	"time"
)

// HomepageTopics 先选择题材再选代表影片；仅查询已保存结果，不调用模型。
func (a *App) HomepageTopics(ctx context.Context) ([]contracts.HomepageTopicGroupDTO, error) {
	topics := []contracts.LibraryTopicDTO{}
	for offset := 0; ; offset += 100 {
		page, err := a.store.ListLibraryTopics(ctx, 100, offset)
		if err != nil {
			return nil, err
		}
		topics = append(topics, page...)
		if len(page) < 100 {
			break
		}
	}
	day := time.Now().UTC().Format("2006-01-02")
	policy, err := a.loadHomepageRecommendationFeedbackPolicy(ctx, day)
	if err != nil {
		return nil, err
	}
	// 每天稳定轮换，避免每次进入首页洗牌。
	sort.SliceStable(topics, func(i, j int) bool { return topicDailyRank(day, topics[i].ID) < topicDailyRank(day, topics[j].ID) })
	out := []contracts.HomepageTopicGroupDTO{}
	seen := map[string]bool{}
	for _, topic := range topics {
		if topic.Hidden || topic.MovieCount < 2 {
			continue
		}
		page := contracts.MoviesPageDTO{}
		for offset := 0; ; offset += 100 {
			batch, err := a.store.ListMovies(ctx, contracts.ListMoviesRequest{TopicID: topic.ID, Limit: 100, Offset: offset})
			if err != nil {
				return nil, err
			}
			page.Items = append(page.Items, batch.Items...)
			if offset+len(batch.Items) >= batch.Total || len(batch.Items) == 0 {
				break
			}
		}
		// 反馈先过滤、降权，日内 hash 保持排序稳定。
		sort.SliceStable(page.Items, func(i, j int) bool {
			fi, _ := policy.factorForMovie(page.Items[i])
			fj, _ := policy.factorForMovie(page.Items[j])
			if fi != fj {
				return fi > fj
			}
			return topicDailyRank(day, page.Items[i].ID) < topicDailyRank(day, page.Items[j].ID)
		})
		movies := []contracts.MovieListItemDTO{}
		for _, m := range page.Items {
			factor, _ := policy.factorForMovie(m)
			if seen[m.ID] || factor == 0 {
				continue
			}
			movies = append(movies, m)
			if len(movies) == 6 {
				break
			}
		}
		if len(movies) < 2 {
			continue
		}
		for _, m := range movies {
			seen[m.ID] = true
		}
		out = append(out, contracts.HomepageTopicGroupDTO{Topic: topic, Movies: movies})
		if len(out) == 3 {
			break
		}
	}
	return out, nil
}

// topicDailyRank 生成稳定排序键，不改变题材归属。
func topicDailyRank(day, id string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(day + ":" + id))
	return h.Sum64()
}
