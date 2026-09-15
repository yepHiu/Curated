package tools

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"curated-backend/internal/agent/core"
	"curated-backend/internal/contracts"
	"curated-backend/internal/storage"
)

var (
	// ErrComicLibraryDisabled is returned by App when the comic Beta is off.
	ErrComicLibraryDisabled = errors.New("comic library is disabled")
	// ErrPhotoLibraryDisabled is returned by App when the photo Beta is off.
	ErrPhotoLibraryDisabled = errors.New("photo library is disabled")
)

// BookLibraryQuery is the read surface for enabled comic and photo libraries.
type BookLibraryQuery interface {
	ComicLibraryEnabled() bool
	PhotoLibraryEnabled() bool
	ListComicBooks(ctx context.Context, req contracts.ListComicBooksRequest) (contracts.ComicBooksPageDTO, error)
	GetComicBookDetail(ctx context.Context, comicID string) (contracts.ComicBookDetailDTO, error)
	GetComicComment(ctx context.Context, comicID string) (contracts.ComicCommentDTO, error)
	ListPhotoBooks(ctx context.Context, req contracts.ListPhotoBooksRequest) (contracts.PhotoBooksPageDTO, error)
	GetPhotoBookDetail(ctx context.Context, photoID string) (contracts.PhotoBookDetailDTO, error)
	GetPhotoComment(ctx context.Context, photoID string) (contracts.PhotoCommentDTO, error)
}

// BookLibraryWrite is the personal-note and display-title write surface for comic and photo books.
type BookLibraryWrite interface {
	ComicLibraryEnabled() bool
	PhotoLibraryEnabled() bool
	GetComicBookDetail(ctx context.Context, comicID string) (contracts.ComicBookDetailDTO, error)
	GetComicComment(ctx context.Context, comicID string) (contracts.ComicCommentDTO, error)
	UpsertComicComment(ctx context.Context, comicID, body string, expected ...string) (contracts.ComicCommentDTO, error)
	PatchComicBook(ctx context.Context, comicID string, patch contracts.PatchComicBookRequest) (contracts.ComicBookDetailDTO, error)
	GetPhotoBookDetail(ctx context.Context, photoID string) (contracts.PhotoBookDetailDTO, error)
	GetPhotoComment(ctx context.Context, photoID string) (contracts.PhotoCommentDTO, error)
	UpsertPhotoComment(ctx context.Context, photoID, body string, expected ...string) (contracts.PhotoCommentDTO, error)
	PatchPhotoBook(ctx context.Context, photoID string, patch contracts.PatchPhotoBookRequest) (contracts.PhotoBookDetailDTO, error)
}

// RegisterBookQueryTools adds comic and photo search/detail tools when the App implements BookLibraryQuery.
func RegisterBookQueryTools(reg *core.Registry, query BookLibraryQuery) error {
	if query == nil {
		return nil
	}
	for _, def := range []core.ToolDefinition{
		searchComics(query),
		getComicDetail(query),
		searchPhotos(query),
		getPhotoDetail(query),
	} {
		if err := reg.Register(def); err != nil {
			return err
		}
	}
	return nil
}

// RegisterBookPresentTools adds comic and photo chat cards bound to this-turn book refs.
func RegisterBookPresentTools(reg *core.Registry, refs *core.BookRefStore) error {
	if refs == nil {
		return nil
	}
	if err := reg.Register(presentBooks(refs, "comic", core.PresentComicsName, "comicId")); err != nil {
		return err
	}
	return reg.Register(presentBooks(refs, "photo", core.PresentPhotosName, "photoId"))
}

// RegisterBookWriteTools adds comic and photo note preview/apply tools.
func RegisterBookWriteTools(reg *core.Registry, write BookLibraryWrite) error {
	if write == nil {
		return nil
	}
	for _, def := range []core.ToolDefinition{
		saveComicComment(write),
		savePhotoComment(write),
		updateComicTitle(write),
		updatePhotoTitle(write),
	} {
		if err := reg.Register(def); err != nil {
			return err
		}
	}
	return nil
}

// searchComics 在漫画 Beta 开启时按标题、文件名和标签检索本地漫画。
func searchComics(q BookLibraryQuery) core.ToolDefinition {
	schema := object(map[string]core.Schema{
		"q":          strField("Free-text query over comic title, file name, and tags"),
		"tag":        strField("Exact comic tag"),
		"favorite":   {Type: "boolean", Description: "Only favorite comics when true"},
		"readStatus": core.Schema{Type: "string", Description: "unread, reading, or read", Enum: []string{"unread", "reading", "read"}},
		"limit":      intField("Page size, max 50", 1, 50),
		"offset":     intField("Page offset", 0, 100000),
	})
	return core.ToolDefinition{
		Name: "search_comics",
		Description: "Search the local comic library when the comic Beta is enabled. " +
			"Never invent comic IDs. Results omit filesystem paths. Paginate with limit (max 50) and offset.",
		ParamsSchema: schema,
		Permission:   core.PermissionRead,
		Domain:       core.DomainQuery,
		Handler: func(ctx context.Context, call core.Call) (core.Result, error) {
			// 关闭 Beta 时拒绝检索，结果不含文件系统路径。
			if !q.ComicLibraryEnabled() {
				return disabledBookResult(true), nil
			}
			args := decodeArgs(call.Args)
			limit := clampLimit(intArg(args, "limit", 20))
			offset := intArg(args, "offset", 0)
			page, err := q.ListComicBooks(ctx, contracts.ListComicBooksRequest{
				Query:      strArg(args, "q"),
				Tag:        strArg(args, "tag"),
				Favorite:   boolPtrArg(args, "favorite"),
				ReadStatus: strArg(args, "readStatus"),
				Limit:      limit,
				Offset:     offset,
			})
			if err != nil {
				return bookQueryError(err, true), nil
			}
			cards := make([]map[string]any, 0, len(page.Items))
			for _, item := range page.Items {
				cards = append(cards, comicCard(item))
			}
			next, truncated := core.PageCursor(offset, limit, page.Total)
			return core.Result{
				OK: true,
				Data: wrapSource(map[string]any{
					"total": page.Total, "items": cards, "limit": limit, "offset": offset,
					"query": map[string]any{"q": strArg(args, "q"), "tag": strArg(args, "tag"), "readStatus": strArg(args, "readStatus")},
				}),
				Truncated:  truncated,
				NextCursor: next,
			}, nil
		},
	}
}

// getComicDetail 读取一本漫画的资料，可选附带个人笔记。
func getComicDetail(q BookLibraryQuery) core.ToolDefinition {
	schema := object(map[string]core.Schema{
		"comicId": strField("Stable comic id from search_comics or the current page"),
		"include": {Type: "array", Description: "Optional extra sections", Items: &core.Schema{Type: "string", Enum: []string{"comment"}}},
	}, "comicId")
	return core.ToolDefinition{
		Name:         "get_comic_detail",
		Description:  "Get one comic book by id when the comic Beta is enabled. Optional include: comment. Does not return archive paths or page binaries.",
		ParamsSchema: schema,
		Permission:   core.PermissionRead,
		Domain:       core.DomainQuery,
		Handler: func(ctx context.Context, call core.Call) (core.Result, error) {
			// 只返回本轮已确认存在的漫画，不含归档路径。
			if !q.ComicLibraryEnabled() {
				return disabledBookResult(true), nil
			}
			args := decodeArgs(call.Args)
			id := strArg(args, "comicId")
			detail, err := q.GetComicBookDetail(ctx, id)
			if err != nil {
				return bookQueryError(err, true), nil
			}
			payload := comicDetailCard(detail)
			for _, inc := range includeArgs(args["include"]) {
				if inc == "comment" {
					comment, _ := q.GetComicComment(ctx, id)
					payload["comment"] = comment.Body
				}
			}
			return core.Result{OK: true, Data: wrapSource(payload)}, nil
		},
	}
}

// searchPhotos 在写真 Beta 开启时按标题、文件名和标签检索本地写真。
func searchPhotos(q BookLibraryQuery) core.ToolDefinition {
	schema := object(map[string]core.Schema{
		"q":     strField("Free-text query over photo-book title, file name, and tags"),
		"tag":   strField("Exact photo tag"),
		"limit": intField("Page size, max 50", 1, 50),
		"offset": intField("Page offset", 0, 100000),
	})
	return core.ToolDefinition{
		Name: "search_photos",
		Description: "Search the local photo-book library when the photo Beta is enabled. " +
			"Never invent photo IDs. Results omit filesystem paths. Paginate with limit (max 50) and offset.",
		ParamsSchema: schema,
		Permission:   core.PermissionRead,
		Domain:       core.DomainQuery,
		Handler: func(ctx context.Context, call core.Call) (core.Result, error) {
			// 关闭 Beta 时拒绝检索，结果不含文件系统路径。
			if !q.PhotoLibraryEnabled() {
				return disabledBookResult(false), nil
			}
			args := decodeArgs(call.Args)
			limit := clampLimit(intArg(args, "limit", 20))
			offset := intArg(args, "offset", 0)
			page, err := q.ListPhotoBooks(ctx, contracts.ListPhotoBooksRequest{
				Query:  strArg(args, "q"),
				Tag:    strArg(args, "tag"),
				Limit:  limit,
				Offset: offset,
			})
			if err != nil {
				return bookQueryError(err, false), nil
			}
			cards := make([]map[string]any, 0, len(page.Items))
			for _, item := range page.Items {
				cards = append(cards, photoCard(item))
			}
			next, truncated := core.PageCursor(offset, limit, page.Total)
			return core.Result{
				OK: true,
				Data: wrapSource(map[string]any{
					"total": page.Total, "items": cards, "limit": limit, "offset": offset,
					"query": map[string]any{"q": strArg(args, "q"), "tag": strArg(args, "tag")},
				}),
				Truncated:  truncated,
				NextCursor: next,
			}, nil
		},
	}
}

// getPhotoDetail 读取一本写真的资料，可选附带个人笔记。
func getPhotoDetail(q BookLibraryQuery) core.ToolDefinition {
	schema := object(map[string]core.Schema{
		"photoId": strField("Stable photo-book id from search_photos or the current page"),
		"include": {Type: "array", Description: "Optional extra sections", Items: &core.Schema{Type: "string", Enum: []string{"comment"}}},
	}, "photoId")
	return core.ToolDefinition{
		Name:         "get_photo_detail",
		Description:  "Get one photo book by id when the photo Beta is enabled. Optional include: comment. Does not return archive paths or page binaries.",
		ParamsSchema: schema,
		Permission:   core.PermissionRead,
		Domain:       core.DomainQuery,
		Handler: func(ctx context.Context, call core.Call) (core.Result, error) {
			// 只返回本轮已确认存在的写真，不含归档路径。
			if !q.PhotoLibraryEnabled() {
				return disabledBookResult(false), nil
			}
			args := decodeArgs(call.Args)
			id := strArg(args, "photoId")
			detail, err := q.GetPhotoBookDetail(ctx, id)
			if err != nil {
				return bookQueryError(err, false), nil
			}
			payload := photoDetailCard(detail)
			for _, inc := range includeArgs(args["include"]) {
				if inc == "comment" {
					comment, _ := q.GetPhotoComment(ctx, id)
					payload["comment"] = comment.Body
				}
			}
			return core.Result{OK: true, Data: wrapSource(payload)}, nil
		},
	}
}

// presentBooks 把本轮已读取的漫画或写真投影为聊天卡片，最多 6 张。
func presentBooks(refs *core.BookRefStore, kind, name, idField string) core.ToolDefinition {
	itemSchema := object(map[string]core.Schema{
		idField: strField("Book id from search or detail in this turn"),
	}, idField)
	schema := object(map[string]core.Schema{
		"items": {
			Type:        "array",
			Description: "Books to show the user as cards, max 6. IDs must already have been retrieved.",
			MinItems:    1,
			MaxItems:    core.PresentBooksMaxItems,
			Items:       &itemSchema,
		},
	}, "items")
	searchName := "search_comics"
	detailName := "get_comic_detail"
	if kind == "photo" {
		searchName = "search_photos"
		detailName = "get_photo_detail"
	}
	return core.ToolDefinition{
		Name: name,
		Description: fmt.Sprintf("Show up to 6 %s cards in the chat UI. %s needs a local read from %s or %s in this request; page IDs alone are insufficient. Never invent IDs.",
			kind, idField, searchName, detailName),
		ParamsSchema: schema,
		Permission:   core.PermissionRead,
		Domain:       core.DomainPresent,
		Handler: func(ctx context.Context, call core.Call) (core.Result, error) {
			// 页面 ID 单独不够；必须本轮 search/detail 读过标题或封面。
			args := decodeArgs(call.Args)
			ids := presentBookIDs(args["items"], idField)
			if len(ids) == 0 {
				return core.Result{OK: false, Error: &core.ToolError{
					Code:    "AI_TOOL_INVALID_ARGS",
					Message: idField + " is required",
				}}, nil
			}
			found, missing := refs.Lookup(call.SessionID, kind, ids)
			if len(missing) > 0 {
				return core.Result{OK: false, Error: &core.ToolError{
					Code:    "AI_TOOL_INVALID_ARGS",
					Message: fmt.Sprintf("unknown %s ids: %s; retrieve them with %s or %s first", kind, strings.Join(missing, ", "), searchName, detailName),
				}}, nil
			}
			cards := make([]map[string]any, 0, len(found))
			for _, ref := range found {
				if strings.TrimSpace(ref.Title) == "" && strings.TrimSpace(ref.CoverURL) == "" {
					return core.Result{OK: false, Error: &core.ToolError{
						Code:    "AI_TOOL_INVALID_ARGS",
						Message: fmt.Sprintf("read %s with %s or %s before presenting a card", kind, searchName, detailName),
					}}, nil
				}
				card := map[string]any{
					"kind":     ref.Kind,
					"title":    ref.Title,
					"coverUrl": ref.CoverURL,
					"tags":     ref.Tags,
				}
				if kind == "comic" {
					card["comicId"] = ref.ID
				} else {
					card["photoId"] = ref.ID
				}
				cards = append(cards, card)
			}
			return core.Result{OK: true, Data: wrapSource(map[string]any{"items": cards})}, nil
		},
	}
}

// saveComicComment 预览替换一本漫画的个人笔记，确认前不落库。
func saveComicComment(w BookLibraryWrite) core.ToolDefinition {
	schema := object(map[string]core.Schema{
		"comicId": strField("Comic id whose user comment to replace"),
		"body":    strField("Replacement comment text; empty clears the note"),
	}, "comicId", "body")
	return core.ToolDefinition{
		Name:         core.SaveComicCommentName,
		Description:  "Propose replacing the user comment/note for one comic. Does not write until the user confirms the preview.",
		ParamsSchema: schema,
		Permission:   core.PermissionWritePreview,
		Domain:       core.DomainUserWrite,
		Handler: func(ctx context.Context, call core.Call) (core.Result, error) {
			// 生成确认预览，不直接写入 comic_book_comments。
			return previewSaveBookComment(ctx, w, call, true)
		},
		Apply: func(ctx context.Context, call core.Call) (core.Result, error) {
			// 用户确认后按预览票据写入漫画笔记。
			return applySaveBookComment(ctx, w, call, true)
		},
	}
}

// savePhotoComment 预览替换一本写真的个人笔记，确认前不落库。
func savePhotoComment(w BookLibraryWrite) core.ToolDefinition {
	schema := object(map[string]core.Schema{
		"photoId": strField("Photo-book id whose user comment to replace"),
		"body":    strField("Replacement comment text; empty clears the note"),
	}, "photoId", "body")
	return core.ToolDefinition{
		Name:         core.SavePhotoCommentName,
		Description:  "Propose replacing the user comment/note for one photo book. Does not write until the user confirms the preview.",
		ParamsSchema: schema,
		Permission:   core.PermissionWritePreview,
		Domain:       core.DomainUserWrite,
		Handler: func(ctx context.Context, call core.Call) (core.Result, error) {
			// 生成确认预览，不直接写入 photo_book_comments。
			return previewSaveBookComment(ctx, w, call, false)
		},
		Apply: func(ctx context.Context, call core.Call) (core.Result, error) {
			// 用户确认后按预览票据写入写真笔记。
			return applySaveBookComment(ctx, w, call, false)
		},
	}
}

// previewSaveBookComment 比较当前笔记与建议正文，生成确认变更。
func previewSaveBookComment(ctx context.Context, w BookLibraryWrite, call core.Call, comic bool) (core.Result, error) {
	args, err := decodeWriteArgs(call.Args)
	if err != nil {
		return core.Result{Error: &core.ToolError{Code: "AI_TOOL_INVALID_ARGS", Message: err.Error()}}, nil
	}
	idField := "comicId"
	if !comic {
		idField = "photoId"
	}
	id := strArg(args, idField)
	body := strArg(args, "body")
	if id == "" {
		return core.Result{Error: &core.ToolError{Code: "AI_TOOL_INVALID_ARGS", Message: idField + " is required"}}, nil
	}
	if utf8.RuneCountInString(body) > contracts.MaxBookCommentRunes {
		return core.Result{Error: &core.ToolError{Code: "AI_TOOL_INVALID_ARGS", Message: "comment body too long"}}, nil
	}
	if comic && !w.ComicLibraryEnabled() {
		return disabledBookResult(true), nil
	}
	if !comic && !w.PhotoLibraryEnabled() {
		return disabledBookResult(false), nil
	}
	var currentBody string
	if comic {
		if _, err := w.GetComicBookDetail(ctx, id); err != nil {
			return bookQueryError(err, true), nil
		}
		current, err := w.GetComicComment(ctx, id)
		if err != nil {
			return core.Result{Error: &core.ToolError{Code: "AI_CHAT_FAILED", Message: err.Error()}}, nil
		}
		currentBody = current.Body
	} else {
		if _, err := w.GetPhotoBookDetail(ctx, id); err != nil {
			return bookQueryError(err, false), nil
		}
		current, err := w.GetPhotoComment(ctx, id)
		if err != nil {
			return core.Result{Error: &core.ToolError{Code: "AI_CHAT_FAILED", Message: err.Error()}}, nil
		}
		currentBody = current.Body
	}
	if currentBody == body {
		return core.Result{OK: true, Data: map[string]any{idField: id, "noop": true}}, nil
	}
	return core.Result{
		OK:            true,
		Preconditions: []core.Change{{Path: "comment.body", Before: currentBody}},
		Data:          map[string]any{idField: id},
		Changes: []core.Change{{
			Path:   "comment.body",
			Before: currentBody,
			After:  body,
		}},
	}, nil
}

// applySaveBookComment 在确认后写入漫画或写真笔记，冲突时返回 AI_WRITE_CONFLICT。
func applySaveBookComment(ctx context.Context, w BookLibraryWrite, call core.Call, comic bool) (core.Result, error) {
	args, err := decodeWriteArgs(call.Args)
	if err != nil {
		return core.Result{Error: &core.ToolError{Code: "AI_TOOL_INVALID_ARGS", Message: err.Error()}}, nil
	}
	idField := "comicId"
	if !comic {
		idField = "photoId"
	}
	id := strArg(args, idField)
	body := strArg(args, "body")
	before, valid := previewBefore(call, "comment.body")
	if !valid {
		return conflictResult(), nil
	}
	if comic {
		dto, err := w.UpsertComicComment(ctx, id, body, before)
		if err != nil {
			return bookWriteError(err, true), nil
		}
		return core.Result{OK: true, Data: dto}, nil
	}
	dto, err := w.UpsertPhotoComment(ctx, id, body, before)
	if err != nil {
		return bookWriteError(err, false), nil
	}
	return core.Result{OK: true, Data: dto}, nil
}

// updateComicTitle 预览替换一本漫画的展示标题，确认前不落库。
func updateComicTitle(w BookLibraryWrite) core.ToolDefinition {
	schema := object(map[string]core.Schema{
		"comicId": strField("Comic id whose display title to replace"),
		"title":   strField("Replacement display title written to user_title"),
	}, "comicId", "title")
	return core.ToolDefinition{
		Name:         core.UpdateComicTitleName,
		Description:  "Propose replacing the comic display title overlay. Does not change the source filename title until the user confirms. Empty titles are rejected.",
		ParamsSchema: schema,
		Permission:   core.PermissionWritePreview,
		Domain:       core.DomainUserWrite,
		Handler: func(ctx context.Context, call core.Call) (core.Result, error) {
			// 生成确认预览，不直接写入 comic_books.user_title。
			return previewUpdateBookTitle(ctx, w, call, true)
		},
		Apply: func(ctx context.Context, call core.Call) (core.Result, error) {
			// 用户确认后按预览票据写入漫画展示标题。
			return applyUpdateBookTitle(ctx, w, call, true)
		},
	}
}

// updatePhotoTitle 预览替换一本写真的展示标题，确认前不落库。
func updatePhotoTitle(w BookLibraryWrite) core.ToolDefinition {
	schema := object(map[string]core.Schema{
		"photoId": strField("Photo-book id whose display title to replace"),
		"title":   strField("Replacement display title written to user_title"),
	}, "photoId", "title")
	return core.ToolDefinition{
		Name:         core.UpdatePhotoTitleName,
		Description:  "Propose replacing the photo-book display title overlay. Does not change the source filename title until the user confirms. Empty titles are rejected.",
		ParamsSchema: schema,
		Permission:   core.PermissionWritePreview,
		Domain:       core.DomainUserWrite,
		Handler: func(ctx context.Context, call core.Call) (core.Result, error) {
			// 生成确认预览，不直接写入 photo_books.user_title。
			return previewUpdateBookTitle(ctx, w, call, false)
		},
		Apply: func(ctx context.Context, call core.Call) (core.Result, error) {
			// 用户确认后按预览票据写入写真展示标题。
			return applyUpdateBookTitle(ctx, w, call, false)
		},
	}
}

// previewUpdateBookTitle 比较当前展示标题与建议标题，生成确认变更。
func previewUpdateBookTitle(ctx context.Context, w BookLibraryWrite, call core.Call, comic bool) (core.Result, error) {
	args, err := decodeWriteArgs(call.Args)
	if err != nil {
		return core.Result{Error: &core.ToolError{Code: "AI_TOOL_INVALID_ARGS", Message: err.Error()}}, nil
	}
	idField := "comicId"
	if !comic {
		idField = "photoId"
	}
	id := strArg(args, idField)
	title := strings.TrimSpace(strArg(args, "title"))
	if id == "" {
		return core.Result{Error: &core.ToolError{Code: "AI_TOOL_INVALID_ARGS", Message: idField + " is required"}}, nil
	}
	if title == "" {
		return core.Result{Error: &core.ToolError{Code: "AI_TOOL_INVALID_ARGS", Message: "title is required"}}, nil
	}
	if utf8.RuneCountInString(title) > contracts.MaxBookTitleRunes {
		return core.Result{Error: &core.ToolError{Code: "AI_TOOL_INVALID_ARGS", Message: "title too long"}}, nil
	}
	if comic && !w.ComicLibraryEnabled() {
		return disabledBookResult(true), nil
	}
	if !comic && !w.PhotoLibraryEnabled() {
		return disabledBookResult(false), nil
	}
	var current string
	if comic {
		detail, err := w.GetComicBookDetail(ctx, id)
		if err != nil {
			return bookQueryError(err, true), nil
		}
		current = detail.Title
	} else {
		detail, err := w.GetPhotoBookDetail(ctx, id)
		if err != nil {
			return bookQueryError(err, false), nil
		}
		current = detail.Title
	}
	if current == title {
		return core.Result{OK: true, Data: map[string]any{idField: id, "noop": true}}, nil
	}
	return core.Result{
		OK:            true,
		Preconditions: []core.Change{{Path: "display.userTitle", Before: current}},
		Data:          map[string]any{idField: id},
		Changes: []core.Change{{
			Path:   "display.userTitle",
			Before: current,
			After:  title,
		}},
	}, nil
}

// applyUpdateBookTitle 在确认后写入漫画或写真展示标题，冲突时返回 AI_WRITE_CONFLICT。
func applyUpdateBookTitle(ctx context.Context, w BookLibraryWrite, call core.Call, comic bool) (core.Result, error) {
	args, err := decodeWriteArgs(call.Args)
	if err != nil {
		return core.Result{Error: &core.ToolError{Code: "AI_TOOL_INVALID_ARGS", Message: err.Error()}}, nil
	}
	idField := "comicId"
	if !comic {
		idField = "photoId"
	}
	id := strArg(args, idField)
	title := strings.TrimSpace(strArg(args, "title"))
	before, valid := previewBefore(call, "display.userTitle")
	if !valid {
		return conflictResult(), nil
	}
	if comic {
		dto, err := w.PatchComicBook(ctx, id, contracts.PatchComicBookRequest{Title: &title, ExpectedTitle: &before})
		if err != nil {
			return bookWriteError(err, true), nil
		}
		return core.Result{OK: true, Data: dto}, nil
	}
	dto, err := w.PatchPhotoBook(ctx, id, contracts.PatchPhotoBookRequest{Title: &title, ExpectedTitle: &before})
	if err != nil {
		return bookWriteError(err, false), nil
	}
	return core.Result{OK: true, Data: dto}, nil
}

// comicCard 把漫画列表项投影为不含路径的工具结果。
func comicCard(item contracts.ComicBookListItemDTO) map[string]any {
	return map[string]any{
		"kind":       "comic",
		"comicId":    item.ID,
		"title":      item.Title,
		"tags":       item.Tags,
		"rating":     item.Rating,
		"isFavorite": item.IsFavorite,
		"readStatus": item.ReadStatus,
		"pageCount":  item.PageCount,
		"coverUrl":   item.CoverURL,
		"addedAt":    item.AddedAt,
	}
}

// comicDetailCard 在列表投影上附加阅读进度和源文件名。
func comicDetailCard(detail contracts.ComicBookDetailDTO) map[string]any {
	card := comicCard(detail.ComicBookListItemDTO)
	card["currentPageIndex"] = detail.CurrentPageIndex
	card["sourceFileName"] = detail.SourceFileName
	return card
}

// photoCard 把写真列表项投影为不含路径的工具结果。
func photoCard(item contracts.PhotoBookListItemDTO) map[string]any {
	return map[string]any{
		"kind":      "photo",
		"photoId":   item.ID,
		"title":     item.Title,
		"tags":      item.Tags,
		"rating":    item.Rating,
		"pageCount": item.PageCount,
		"coverUrl":  item.CoverURL,
		"addedAt":   item.AddedAt,
	}
}

// photoDetailCard 在列表投影上附加浏览进度和源文件名。
func photoDetailCard(detail contracts.PhotoBookDetailDTO) map[string]any {
	card := photoCard(detail.PhotoBookListItemDTO)
	card["currentPageIndex"] = detail.CurrentPageIndex
	card["sourceFileName"] = detail.SourceFileName
	return card
}

// presentBookIDs 从 present 参数中提取去重后的漫画或写真 ID。
func presentBookIDs(raw any, idField string) []string {
	arr, ok := raw.([]any)
	if !ok {
		return nil
	}
	ids := make([]string, 0, len(arr))
	seen := map[string]bool{}
	for _, item := range arr {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		id := strArg(obj, idField)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	return ids
}

// disabledBookResult 在对应 Beta 关闭时返回稳定的库禁用错误。
func disabledBookResult(comic bool) core.Result {
	code := contracts.ErrorCodePhotoLibraryDisabled
	message := ErrPhotoLibraryDisabled.Error()
	if comic {
		code = contracts.ErrorCodeComicLibraryDisabled
		message = ErrComicLibraryDisabled.Error()
	}
	return core.Result{OK: false, Error: &core.ToolError{Code: code, Message: message}}
}

// bookQueryError 把书库读取失败映射为工具错误码。
func bookQueryError(err error, comic bool) core.Result {
	if errors.Is(err, ErrComicLibraryDisabled) || errors.Is(err, ErrPhotoLibraryDisabled) {
		return disabledBookResult(comic)
	}
	if errors.Is(err, storage.ErrComicBookNotFound) || errors.Is(err, storage.ErrPhotoBookNotFound) {
		label := "photo book not found"
		if comic {
			label = "comic not found"
		}
		return core.Result{OK: false, Error: &core.ToolError{Code: "COMMON_NOT_FOUND", Message: label}}
	}
	return core.Result{OK: false, Error: &core.ToolError{Code: "AI_CHAT_FAILED", Message: err.Error()}}
}

// bookWriteError 把书库笔记写入失败映射为工具错误码。
func bookWriteError(err error, comic bool) core.Result {
	if errors.Is(err, storage.ErrAIWriteConflict) {
		return conflictResult()
	}
	if errors.Is(err, storage.ErrBookCommentTooLong) {
		return core.Result{Error: &core.ToolError{Code: "AI_TOOL_INVALID_ARGS", Message: "comment body too long"}}
	}
	if errors.Is(err, ErrComicLibraryDisabled) || errors.Is(err, ErrPhotoLibraryDisabled) {
		return disabledBookResult(comic)
	}
	if errors.Is(err, storage.ErrComicBookNotFound) || errors.Is(err, storage.ErrPhotoBookNotFound) {
		return bookQueryError(err, comic)
	}
	return core.Result{Error: &core.ToolError{Code: "AI_CHAT_FAILED", Message: err.Error()}}
}
