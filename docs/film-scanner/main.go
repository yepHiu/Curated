package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/metatube-community/metatube-sdk-go/engine"
	"github.com/metatube-community/metatube-sdk-go/engine/providerid"
	"github.com/metatube-community/metatube-sdk-go/imageutil"
	"github.com/metatube-community/metatube-sdk-go/model"

	"image/jpeg"
)

var (
	dir      = flag.String("dir", ".", "video directory")
	provider = flag.String("provider", "", "provider name (search all if empty)")
	exts     = flag.String("exts", "mp4,mkv,avi,wmv,flv,mov", "video extensions")
	saveNFO  = flag.Bool("nfo", true, "save NFO file")
	download = flag.Bool("download", false, "download cover image")
	preview  = flag.Bool("preview", false, "download preview images")
	output   = flag.String("output", "", "output directory for NFO and images")
	verbose  = flag.Int("v", 0, "verbose level")
)

// 解析番号的常见模式
var numberPatterns = []string{
	`(?i)([a-z]{2,})[-_]?(\d{2,6})`,    // abc-123, abc_123
	`(?i)([a-z]+)(\d{2,6})`,            // abc123
	`(?i)(\d{2,6})([a-z]{2,})`,         // 123abc
	`(?i)([A-Z]{2,})[-_]?(\d{3,6})`,    // ABC-123, ABC_123
	`(?i)(FC2[-]?\d+)`,                 // FC2
	`(?i)(heyzo[-]?\d+)`,               // heyzo
	`(?i)(tokyo[-]?hot[-]?[a-z0-9-]+)`, // tokyo-hot
	`(?i)(1pondo[-]?\d+)`,              // 1pondo
	`(?i)(caribbeancom[-]?\d+)`,        // caribbeancom
	`(?i)(sogrand?[-]?\d+)`,            // so, sog
}

func main() {
	flag.Parse()

	// 创建引擎（使用默认内存数据库）
	eng := engine.Default()

	// 获取所有可用provider
	providers := eng.GetMovieProviders()
	if len(providers) == 0 {
		log.Fatal("no providers available")
	}

	// 显示可用provider
	fmt.Println("Available providers:")
	for name := range providers {
		fmt.Printf("  - %s\n", name)
	}
	fmt.Println()

	// 查找视频文件
	files, err := findVideoFiles(*dir, strings.Split(*exts, ","))
	if err != nil {
		log.Fatalf("find videos error: %v", err)
	}

	fmt.Printf("Found %d video files\n\n", len(files))

	// 遍历处理每个文件
	successCount := 0
	failCount := 0
	for i, file := range files {
		fmt.Printf("[%d/%d] Processing: %s\n", i+1, len(files), filepath.Base(file))

		// 从文件名解析番号
		number := extractNumber(filepath.Base(file))
		if number == "" {
			fmt.Println("  ⚠ Cannot extract number from filename")
			failCount++
			continue
		}
		fmt.Printf("  Number: %s\n", number)

		// 搜索电影
		movie, pid, err := searchMovie(eng, number)
		if err != nil {
			fmt.Printf("  ✗ Error: %v\n", err)
			failCount++
			continue
		}

		if movie == nil {
			fmt.Println("  ✗ Movie not found")
			failCount++
			continue
		}

		// 显示结果
		fmt.Printf("  ✓ Found: %s\n", movie.Title)
		fmt.Printf("    Provider: %s\n", movie.Provider)
		fmt.Printf("    Cover: %s\n", movie.CoverURL)

		// 生成NFO文件
		if *saveNFO {
			outputDir := *output
			if outputDir == "" {
				outputDir = filepath.Dir(file)
			}

			if err := saveNFOFile(movie, file, outputDir); err != nil {
				fmt.Printf("  ✗ Save NFO error: %v\n", err)
			} else {
				fmt.Println("  ✓ NFO saved")
			}
		}

		// 下载封面图片
		if *download && movie.CoverURL != "" {
			outputDir := *output
			if outputDir == "" {
				outputDir = filepath.Dir(file)
			}

			if err := downloadCover(eng, pid, file, outputDir); err != nil {
				fmt.Printf("  ✗ Download cover error: %v\n", err)
			} else {
				fmt.Println("  ✓ Cover downloaded")
			}
		}

		// 下载预览图
		if *preview && len(movie.PreviewImages) > 0 {
			outputDir := *output
			if outputDir == "" {
				outputDir = filepath.Dir(file)
			}

			count, err := downloadPreviews(eng, pid, file, outputDir)
			if err != nil {
				fmt.Printf("  ✗ Download preview error: %v\n", err)
			} else if count > 0 {
				fmt.Printf("  ✓ Preview images downloaded: %d\n", count)
			}
		}

		successCount++

		// 避免请求过快
		if i < len(files)-1 {
			time.Sleep(500 * time.Millisecond)
		}
		fmt.Println()
	}

	fmt.Printf("=== Summary ===\n")
	fmt.Printf("Total: %d, Success: %d, Failed: %d\n", len(files), successCount, failCount)
}

// findVideoFiles 查找目录下的视频文件
func findVideoFiles(dir string, exts []string) ([]string, error) {
	var files []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 跳过目录
		if info.IsDir() {
			return nil
		}

		// 检查扩展名
		ext := strings.TrimPrefix(filepath.Ext(path), ".")
		for _, e := range exts {
			if strings.EqualFold(ext, e) {
				files = append(files, path)
				break
			}
		}

		return nil
	})

	return files, err
}

// extractNumber 从文件名提取番号
func extractNumber(filename string) string {
	// 移除扩展名
	name := strings.TrimSuffix(filename, filepath.Ext(filename))

	// 尝试各种模式
	for _, pattern := range numberPatterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(name)
		if len(matches) > 1 {
			// 组合匹配到的部分
			result := ""
			for _, m := range matches[1:] {
				if m != "" {
					result += m
				}
			}
			if result != "" {
				return strings.ToUpper(result)
			}
		}
	}

	// 直接返回清理后的文件名作为备选
	return strings.ToUpper(strings.ReplaceAll(name, " ", ""))
}

// searchMovie 搜索电影
func searchMovie(eng *engine.Engine, number string) (*model.MovieInfo, *providerid.ProviderID, error) {
	var results []*model.MovieSearchResult
	var err error

	// 优先使用指定provider
	if *provider != "" {
		results, err = eng.SearchMovie(number, *provider, false)
	} else {
		// 从所有provider搜索
		results, err = eng.SearchMovieAll(number, false)
	}

	if err != nil {
		return nil, nil, err
	}

	if len(results) == 0 {
		return nil, nil, nil
	}

	// 取第一个结果
	first := results[0]

	if *verbose > 0 {
		fmt.Printf("  Found %d results, using first: %s (%s)\n", len(results), first.Title, first.Provider)
	}

	// 构建ProviderID并获取详情
	pid := providerid.MustParse(first.Provider + ":" + first.ID)
	movie, err := eng.GetMovieInfoByProviderID(pid, true)
	return movie, &pid, err
}

// saveNFOFile 保存NFO文件
func saveNFOFile(movie *model.MovieInfo, videoPath, outputDir string) error {
	// 创建输出目录
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	// 生成NFO文件名（与视频文件同名）
	baseName := strings.TrimSuffix(filepath.Base(videoPath), filepath.Ext(videoPath))
	nfoPath := filepath.Join(outputDir, baseName+".nfo")

	// 生成NFO内容（Kodi/Jellyfin/Plex兼容格式）
	nfoContent := generateNFO(movie)

	// 写入文件
	return os.WriteFile(nfoPath, []byte(nfoContent), 0644)
}

// generateNFO 生成NFO XML内容
func generateNFO(movie *model.MovieInfo) string {
	var sb strings.Builder

	sb.WriteString("<?xml version=\"1.0\" encoding=\"utf-8\" standalone=\"yes\"?>\n")
	sb.WriteString("<movie>\n")

	// 标题
	sb.WriteString(fmt.Sprintf("  <title>%s</title>\n", escapeXML(movie.Title)))

	// 简介
	if movie.Summary != "" {
		sb.WriteString(fmt.Sprintf("  <plot>%s</plot>\n", escapeXML(movie.Summary)))
	}

	// 番号
	sb.WriteString(fmt.Sprintf("  <num>%s</num>\n", escapeXML(movie.Number)))

	// 发行日期 - 只在日期格式正确时添加 (格式如 "2026-02-12")
	releaseDate := fmt.Sprintf("%s", movie.ReleaseDate)
	if releaseDate != "" && !strings.Contains(releaseDate, "time.Location") && releaseDate != "0001-01-01" {
		sb.WriteString(fmt.Sprintf("  <releasedate>%s</releasedate>\n", releaseDate))
	}

	// 时长
	if movie.Runtime > 0 {
		sb.WriteString(fmt.Sprintf("  <runtime>%d</runtime>\n", movie.Runtime))
	}

	// 评分
	if movie.Score > 0 {
		sb.WriteString(fmt.Sprintf("  <rating>%.1f</rating>\n", movie.Score))
	}

	// 制作商
	if movie.Maker != "" {
		sb.WriteString(fmt.Sprintf("  <studio>%s</studio>\n", escapeXML(movie.Maker)))
	}

	// 发行商
	if movie.Label != "" {
		sb.WriteString(fmt.Sprintf("  <label>%s</label>\n", escapeXML(movie.Label)))
	}

	// 系列
	if movie.Series != "" {
		sb.WriteString(fmt.Sprintf("  <set>%s</set>\n", escapeXML(movie.Series)))
	}

	// 导演
	if movie.Director != "" {
		sb.WriteString(fmt.Sprintf("  <director>%s</director>\n", escapeXML(movie.Director)))
	}

	// 演员
	for _, actor := range movie.Actors {
		sb.WriteString(fmt.Sprintf("  <actor>\n    <name>%s</name>\n  </actor>\n", escapeXML(actor)))
	}

	// 类别/标签
	for _, genre := range movie.Genres {
		sb.WriteString(fmt.Sprintf("  <genre>%s</genre>\n", escapeXML(genre)))
	}

	// 封面URL (作为fanart URL)
	if movie.CoverURL != "" {
		sb.WriteString(fmt.Sprintf("  <thumb aspect=\"poster\">%s</thumb>\n", escapeXML(movie.CoverURL)))
	}

	sb.WriteString("</movie>\n")

	return sb.String()
}

// escapeXML 转义XML特殊字符
func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}

// downloadCover 使用SDK下载封面图片（带人脸检测和裁剪）
func downloadCover(eng *engine.Engine, pid *providerid.ProviderID, videoPath, outputDir string) error {
	if pid == nil {
		return nil
	}

	// 创建输出目录
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	// 生成图片文件名
	baseName := strings.TrimSuffix(filepath.Base(videoPath), filepath.Ext(videoPath))
	coverPath := filepath.Join(outputDir, baseName+"-poster.jpg")

	// 检查文件是否已存在
	if _, err := os.Stat(coverPath); err == nil {
		return nil // 文件已存在
	}

	// 使用SDK获取封面图片（-1 表示使用默认比例和自动人脸检测）
	img, err := eng.GetMoviePrimaryImage(*pid, -1, -1)
	if err != nil {
		return fmt.Errorf("failed to get image: %w", err)
	}

	// 创建文件并写入
	f, err := os.Create(coverPath)
	if err != nil {
		return err
	}
	defer f.Close()

	// 编码为JPEG
	if err := jpeg.Encode(f, img, &jpeg.Options{Quality: 90}); err != nil {
		return err
	}

	return nil
}

// downloadPreviews 下载预览图（不裁剪）
func downloadPreviews(eng *engine.Engine, pid *providerid.ProviderID, videoPath, outputDir string) (int, error) {
	if pid == nil {
		return 0, nil
	}

	// 创建输出目录
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return 0, err
	}

	// 获取电影信息以获取PreviewImages
	movie, err := eng.GetMovieInfoByProviderID(*pid, true)
	if err != nil {
		return 0, fmt.Errorf("failed to get movie info: %w", err)
	}

	if len(movie.PreviewImages) == 0 {
		return 0, nil
	}

	// 获取provider
	pvd := eng.MustGetMovieProviderByName(movie.Provider)

	// 生成基础文件名
	baseName := strings.TrimSuffix(filepath.Base(videoPath), filepath.Ext(videoPath))

	downloaded := 0
	for i, url := range movie.PreviewImages {
		// 生成文件名 (01, 02, 03...)
		filename := fmt.Sprintf("%s-thumb-%02d.jpg", baseName, i+1)
		previewPath := filepath.Join(outputDir, filename)

		// 检查文件是否已存在
		if _, err := os.Stat(previewPath); err == nil {
			continue // 文件已存在，跳过
		}

		// 使用SDK的Fetch获取图片（不裁剪，直接保存原始图片）
		resp, err := eng.Fetch(url, pvd)
		if err != nil {
			if *verbose > 0 {
				fmt.Printf("    Warning: failed to fetch preview %d: %v\n", i+1, err)
			}
			continue
		}
		defer resp.Body.Close()

		// 解码图片
		img, _, err := imageutil.Decode(resp.Body)
		if err != nil {
			if *verbose > 0 {
				fmt.Printf("    Warning: failed to decode preview %d: %v\n", i+1, err)
			}
			continue
		}

		// 创建文件并写入
		f, err := os.Create(previewPath)
		if err != nil {
			continue
		}

		// 编码为JPEG
		if err := jpeg.Encode(f, img, &jpeg.Options{Quality: 85}); err != nil {
			f.Close()
			continue
		}
		f.Close()
		downloaded++
	}

	return downloaded, nil
}
