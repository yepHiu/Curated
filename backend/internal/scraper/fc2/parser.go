package fc2

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"curated-backend/internal/scraper"
	"github.com/PuerkitoBio/goquery"
)

var (
	codeInTextPattern  = regexp.MustCompile(`(?i)FC2[-_ ]*(?:PPV[-_ ]*)?([0-9]+)`)
	datePattern        = regexp.MustCompile(`\b([0-9]{4})[-/]([0-9]{1,2})[-/]([0-9]{1,2})\b`)
	isoDurationPattern = regexp.MustCompile(`^PT(?:(\d+)H)?(?:(\d+)M)?(?:(\d+)S)?$`)
	clockPattern       = regexp.MustCompile(`\b(?:(\d+):)?(\d{1,2}):(\d{2})\b`)
	actorPattern       = regexp.MustCompile(`(?m)^\s*名前[：:\s　]*([^\r\n]+)`)
)

// parseDocument 解析成功字段并校验详情身份；商品不存在页面不参与补全。
func parseDocument(doc *goquery.Document, provider, movieID, digits, homepage string) (scraper.Metadata, error) {
	pageTitle := text(doc, "title")
	if strings.Contains(pageTitle, "お探しの商品が見つかりません") || strings.Contains(pageTitle, "couldn't find any products") ||
		strings.Contains(pageTitle, "404 File Not Found") || strings.Contains(pageTitle, "お探しのページは見つかりません") {
		return scraper.Metadata{}, ErrNotFound
	}
	if err := validateDetailIdentity(doc, provider, digits, homepage); err != nil {
		return scraper.Metadata{}, err
	}
	out := scraper.Metadata{MovieID: movieID, Number: "FC2-" + digits, Provider: provider, Homepage: homepage, Series: "FC2"}
	switch provider {
	case "FC2":
		parseOfficial(doc, &out)
	case "fc2hub":
		if err := parseJavten(doc, &out, digits); err != nil {
			return scraper.Metadata{}, err
		}
	case "PPVDataBank":
		parseDatabank(doc, &out)
	case "JavDB":
		parseJavDB(doc, &out)
	}
	if out.Title == "" {
		return scraper.Metadata{}, fmt.Errorf("parser failed: missing title for FC2-%s", digits)
	}
	if out.CoverURL == "" {
		out.CoverURL = out.ThumbURL
	}
	if out.ThumbURL == "" {
		out.ThumbURL = out.CoverURL
	}
	out.Actors, out.Tags, out.PreviewImages = unique(out.Actors), unique(out.Tags), unique(out.PreviewImages)
	return out, nil
}

// javDBNumber 只读取番号栏目，避免正文或相关作品中的同号文字通过身份检查。
func javDBNumber(doc *goquery.Document) string {
	var number string
	doc.Find(".movie-panel-info .panel-block").Each(func(_ int, node *goquery.Selection) {
		// 中日文番号标签均使用其 value，忽略其它栏目的文本。
		label := strings.Trim(strings.TrimSpace(node.Find("strong").Text()), ":： ")
		if label == "番號" || label == "番号" || label == "编号" {
			number = strings.TrimSpace(node.Find(".value").Text())
		}
	})
	return number
}

// parseOfficial 同时支持 FC2 新旧页面结构与隐藏标题水印。
func parseOfficial(doc *goquery.Document, out *scraper.Metadata) {
	heading := doc.Find("div[data-section='userInfo'] h3, .items_article_headerInfo h3").First().Clone()
	heading.Find("[style]").Remove()
	out.Title = strings.TrimSpace(heading.Text())
	out.Summary = attr(doc, "meta[name='description']", "content")
	out.Studio = first(text(doc, "div[data-section='userInfo'] a[href*='/users/']"), text(doc, ".items_article_headerInfo ul li:last-child a"))
	out.ThumbURL = absoluteURL(out.Homepage, attr(doc, ".items_article_MainitemThumb img", "src"))
	out.CoverURL = out.ThumbURL
	out.PreviewImages = imageLinks(doc, out.Homepage, ".items_article_SampleImagesArea li a, .items_article_SampleImages li a")
	out.Tags = selectionTexts(doc, ".items_article_TagArea a, p.card-text a[href*='/tag/']")
	// 栏目逐项分隔，避免相邻商品 ID 文本紧贴日期末尾而导致解析失败。
	out.ReleaseDate = first(parseDate(strings.Join(selectionTexts(doc, ".items_article_Releasedate p, .items_article_softDevice p"), " ")), parseDate(text(doc, ".items_article_Releasedate")))
	out.RuntimeMinutes = durationMinutes(text(doc, ".items_article_MainitemThumb .items_article_info"))
	class := attr(doc, ".items_article_StarA span", "class")
	if match := regexp.MustCompile(`(\d+)$`).FindStringSubmatch(class); len(match) == 2 {
		out.Rating, _ = strconv.ParseFloat(match[1], 64)
	}
}

// parseJavten 组合 HTML 与 JSON-LD，并从出演说明补充演员姓名。
func parseJavten(doc *goquery.Document, out *scraper.Metadata, digits string) error {
	if number := text(doc, "h1.fc2-id"); number != "" && !numberInText(number, digits) {
		return ErrIdentity
	}
	movie := movieJSONLD(doc)
	if identifiers := stringValues(movie["identifier"]); len(identifiers) > 0 {
		matched := false
		for _, value := range identifiers {
			if Digits(value) == digits {
				matched = true
			}
		}
		if !matched {
			return ErrIdentity
		}
	}
	out.Title = first(text(doc, "h1.card-text.fc2-title"), stringValue(movie["name"]))
	out.Summary = first(descriptionText(doc), stringValue(movie["description"]))
	out.CoverURL = absoluteURL(out.Homepage, first(firstString(movie["image"]), attr(doc, "meta[property='og:image']", "content"), attr(doc, "meta[name='twitter:image']", "content")))
	out.ThumbURL = out.CoverURL
	out.ReleaseDate = parseDate(stringValue(movie["datePublished"]))
	out.RuntimeMinutes = durationMinutes(stringValue(movie["duration"]))
	out.Tags = append(selectionTexts(doc, "p.card-text a.badge"), stringValues(movie["genre"])...)
	out.Actors = actorValues(movie["actor"])
	performers := strings.SplitN(descriptionText(doc), "■出演", 2)
	if len(performers) == 2 {
		block := strings.SplitN(performers[1], "\n■", 2)[0]
		for _, match := range actorPattern.FindAllStringSubmatch(block, -1) {
			out.Actors = append(out.Actors, strings.TrimSpace(match[1]))
		}
	}
	doc.Find(".card").Each(func(_ int, node *goquery.Selection) {
		// 按卖家栏目定位，避免依赖页面列的位置。
		if strings.Contains(node.Find(".card-header").Text(), "売り手情報") {
			seller := node.Find(".col-8").First().Clone()
			seller.Find(".badge").Remove()
			out.Studio = strings.TrimSpace(seller.Text())
		}
	})
	out.Studio = first(out.Studio, namedValue(movie["director"]))
	if rating, ok := movie["aggregateRating"].(map[string]any); ok {
		out.Rating, _ = strconv.ParseFloat(fmt.Sprint(rating["ratingValue"]), 64)
	}
	out.PreviewImages = imageLinks(doc, out.Homepage, "a[data-fancybox='gallery']")
	return nil
}

// parseDatabank 读取 PPVDataBank 的番号直达页面作为官方失效后的独立来源。
func parseDatabank(doc *goquery.Document, out *scraper.Metadata) {
	out.Title = first(text(doc, ".article_title a"), attr(doc, "meta[name='title']", "content"))
	out.Summary = text(doc, ".explanation")
	out.CoverURL = absoluteURL(out.Homepage, attr(doc, ".thumb img", "src"))
	out.ThumbURL = out.CoverURL
	out.PreviewImages = imageLinks(doc, out.Homepage, ".sample_image_area a")
	doc.Find("ul.meta li").Each(func(_ int, node *goquery.Selection) {
		// 页面元信息通过标签识别，允许栏目顺序改变。
		parts := strings.SplitN(strings.TrimSpace(node.Text()), ":", 2)
		if len(parts) != 2 {
			return
		}
		value := strings.TrimSpace(parts[1])
		switch strings.TrimSpace(parts[0]) {
		case "販売者":
			out.Studio = first(strings.TrimSpace(node.Find("a").Text()), value)
		case "販売日":
			out.ReleaseDate = parseDate(value)
		case "再生時間":
			out.RuntimeMinutes = durationMinutes(value)
		}
	})
}

// parseJavDB 提取已验证番号的 JavDB 元数据，供 FC2 多源补全。
func parseJavDB(doc *goquery.Document, out *scraper.Metadata) {
	out.Title = text(doc, "h2.title strong.current-title")
	out.CoverURL = absoluteURL(out.Homepage, attr(doc, "img.video-cover", "src"))
	out.ThumbURL = out.CoverURL
	doc.Find(".movie-panel-info .panel-block").Each(func(_ int, node *goquery.Selection) {
		// 中日文栏目共同支持；身份已在详情入口检查。
		label := strings.TrimSpace(node.Find("strong").Text())
		value := strings.TrimSpace(node.Find(".value").Text())
		switch {
		case strings.Contains(label, "演員") || strings.Contains(label, "演员") || strings.Contains(label, "出演"):
			out.Actors = selectionTextsNode(node, ".value a")
		case strings.Contains(label, "日期") || strings.Contains(label, "発売"):
			out.ReleaseDate = parseDate(value)
		case strings.Contains(label, "時長") || strings.Contains(label, "时长") || strings.Contains(label, "収録"):
			out.RuntimeMinutes = durationMinutes(value)
		case strings.Contains(label, "片商") || strings.Contains(label, "メーカー"):
			out.Studio = value
		case strings.Contains(label, "類別") || strings.Contains(label, "类别") || strings.Contains(label, "ジャンル"):
			out.Tags = selectionTextsNode(node, ".value a")
		case strings.Contains(label, "評分") || strings.Contains(label, "评分"):
			if values := strings.Fields(value); len(values) > 0 {
				out.Rating, _ = strconv.ParseFloat(values[0], 64)
			}
		}
	})
	out.PreviewImages = imageLinks(doc, out.Homepage, ".preview-images a.tile-item")
}

// movieJSONLD 展开 JSON-LD 数组与 @graph，找到 Movie 记录。
func movieJSONLD(doc *goquery.Document) map[string]any {
	var movie map[string]any
	doc.Find("script[type='application/ld+json']").Each(func(_ int, node *goquery.Selection) {
		// 错误的独立脚本不影响其它脚本与 HTML 回退。
		if movie != nil {
			return
		}
		var value any
		if json.Unmarshal([]byte(node.Text()), &value) == nil {
			movie = findMovieJSON(value)
		}
	})
	return movie
}

// findMovieJSON 递归查找结构化记录，支持 @type 字符串或数组。
func findMovieJSON(value any) map[string]any {
	switch item := value.(type) {
	case []any:
		for _, entry := range item {
			if found := findMovieJSON(entry); found != nil {
				return found
			}
		}
	case map[string]any:
		for _, kind := range stringValues(item["@type"]) {
			if kind == "Movie" {
				return item
			}
		}
		return findMovieJSON(item["@graph"])
	}
	return nil
}

// descriptionText 保留简介换行，便于独立解析出演信息。
func descriptionText(doc *goquery.Document) string {
	node := doc.Find("div.col.des").First().Clone()
	node.Find("script, style, noscript").Remove()
	node.Find("br").ReplaceWithHtml("\n")
	return strings.TrimSpace(node.Text())
}

// numberInText 按数字完整比较，不接受 39776180 对 3977618 的前缀匹配。
func numberInText(value, digits string) bool {
	for _, match := range codeInTextPattern.FindAllStringSubmatch(value, -1) {
		if match[1] == digits {
			return true
		}
	}
	return false
}

// parseDate 解析常见站点日期并拒绝无效日历日期。
func parseDate(value string) string {
	match := datePattern.FindStringSubmatch(value)
	if len(match) != 4 {
		return ""
	}
	year, _ := strconv.Atoi(match[1])
	month, _ := strconv.Atoi(match[2])
	day, _ := strconv.Atoi(match[3])
	date := fmt.Sprintf("%04d-%02d-%02d", year, month, day)
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return ""
	}
	return date
}

// durationMinutes 将 ISO 时长、时分秒与分钟文字转为资料库分钟值。
func durationMinutes(value string) int {
	if match := isoDurationPattern.FindStringSubmatch(strings.TrimSpace(value)); len(match) == 4 {
		hours, _ := strconv.Atoi(match[1])
		minutes, _ := strconv.Atoi(match[2])
		seconds, _ := strconv.Atoi(match[3])
		return (hours*3600 + minutes*60 + seconds) / 60
	}
	if match := clockPattern.FindStringSubmatch(value); len(match) == 4 {
		hours, _ := strconv.Atoi(match[1])
		minutes, _ := strconv.Atoi(match[2])
		return hours*60 + minutes
	}
	match := regexp.MustCompile(`(\d+)\s*(?:分|分鐘|分钟|min)`).FindStringSubmatch(value)
	if len(match) == 2 {
		minutes, _ := strconv.Atoi(match[1])
		return minutes
	}
	return 0
}

// absoluteURL 解析相对资源地址，拒绝非 HTTP(S) 资源。
func absoluteURL(base, raw string) string {
	root, err := url.Parse(base)
	if err != nil || strings.TrimSpace(raw) == "" {
		return ""
	}
	ref, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return ""
	}
	resolved := root.ResolveReference(ref)
	if resolved.Scheme != "http" && resolved.Scheme != "https" {
		return ""
	}
	return resolved.String()
}

// attr 读取第一个节点的属性并去除首尾空白。
func attr(doc *goquery.Document, selector, attribute string) string {
	value, _ := doc.Find(selector).First().Attr(attribute)
	return strings.TrimSpace(value)
}

// text 读取第一个节点的纯文本。
func text(doc *goquery.Document, selector string) string {
	return strings.TrimSpace(doc.Find(selector).First().Text())
}

// selectionTexts 收集页面指定节点的文本值。
func selectionTexts(doc *goquery.Document, selector string) []string {
	return selectionTextsNode(doc.Selection, selector)
}

// selectionTextsNode 在栏目节点内收集文本值。
func selectionTextsNode(root *goquery.Selection, selector string) []string {
	var values []string
	root.Find(selector).Each(func(_ int, node *goquery.Selection) { // 保留节点顺序，统一在输出阶段去重。
		values = append(values, strings.TrimSpace(node.Text()))
	})
	return values
}

// imageLinks 提取图片链接并解析相对地址。
func imageLinks(doc *goquery.Document, base, selector string) []string {
	var values []string
	doc.Find(selector).Each(func(_ int, node *goquery.Selection) { // 忽略无链接的装饰节点。
		value, _ := node.Attr("href")
		if resolved := absoluteURL(base, value); resolved != "" {
			values = append(values, resolved)
		}
	})
	return values
}

// first 按优先级选取首个非空字符串。
func first(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

// unique 清理空白和重复值，保留稳定顺序。
func unique(values []string) []string {
	var out []string
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" && !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	return out
}

// stringValue 读取 JSON 字符串，避免将 null 或对象变成文本。
func stringValue(value any) string { result, _ := value.(string); return strings.TrimSpace(result) }

// stringValues 兼容 JSON 字符串与字符串数组。
func stringValues(value any) []string {
	if result := stringValue(value); result != "" {
		return []string{result}
	}
	var out []string
	if array, ok := value.([]any); ok {
		for _, item := range array {
			if result := stringValue(item); result != "" {
				out = append(out, result)
			}
		}
	}
	return out
}

// firstString 从 JSON 字符串或数组选取首项。
func firstString(value any) string {
	values := stringValues(value)
	if len(values) > 0 {
		return values[0]
	}
	return ""
}

// namedValue 兼容 JSON-LD 名称字符串与带 name 的对象。
func namedValue(value any) string {
	if object, ok := value.(map[string]any); ok {
		return stringValue(object["name"])
	}
	return stringValue(value)
}

// actorValues 兼容演员名称数组和 Person 对象数组。
func actorValues(value any) []string {
	var out []string
	if array, ok := value.([]any); ok {
		for _, actor := range array {
			if name := namedValue(actor); name != "" {
				out = append(out, name)
			}
		}
	} else if name := namedValue(value); name != "" {
		out = append(out, name)
	}
	return out
}
