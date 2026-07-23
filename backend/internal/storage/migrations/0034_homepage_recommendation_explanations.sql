ALTER TABLE homepage_daily_recommendations
  ADD COLUMN recommendation_items_json TEXT NOT NULL DEFAULT '[]';
