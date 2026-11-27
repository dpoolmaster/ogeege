package publicsite

import (
	"bytes"
	"encoding/xml"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"oge-ege/internal/auth"
	"oge-ege/internal/shared/httpserver"
)

// Handler обслуживает публичные HTTP-маршруты.
type Handler struct {
	service *Service
	csrf    *httpserver.CSRF
	signer  *DemoSigner
	cache   *Cache
	logger  *slog.Logger
	secure  bool
	baseURL string
	brand   string
}

// NewHandler создаёт публичный HTTP-обработчик.
func NewHandler(
	service *Service,
	csrf *httpserver.CSRF,
	signer *DemoSigner,
	cache *Cache,
	logger *slog.Logger,
	secure bool,
	baseURL string,
	brand string,
) *Handler {
	return &Handler{
		service: service,
		csrf:    csrf,
		signer:  signer,
		cache:   cache,
		logger:  logger,
		secure:  secure,
		baseURL: strings.TrimRight(baseURL, "/"),
		brand:   brand,
	}
}

// Register регистрирует публичные маршруты; catch-all должен оставаться последним.
func (h *Handler) Register(router gin.IRoutes, limiter *httpserver.RateLimiter) {
	limited := limiter.Middleware(httpserver.ClientIPKey)

	router.GET("/", h.Home)
	router.GET("/tracks/:examCode", h.Landing)
	router.GET("/bank", h.Bank)
	router.GET("/hx/bank", h.BankFragment)
	router.GET("/bank/:taskId", h.Task)
	router.POST("/bank/:taskId/check", limited, h.CheckTask)
	router.GET("/diagnostics/demo", h.Demo)
	router.POST("/diagnostics/demo/check", limited, h.CheckDemo)
	router.GET("/articles", h.Articles)
	router.GET("/articles/:slug", h.Article)
	router.GET("/hx/leads", h.LeadFormFragment)
	router.POST("/leads", limited, h.SubmitLead)
	router.POST("/hx/leads", limited, h.SubmitLeadHX)
	router.GET("/sitemap.xml", h.Sitemap)
	router.GET("/robots.txt", h.Robots)
	router.GET("/static/public/*file", AssetsHandler())
	router.GET("/:alias", h.Redirect)
}

// Home обслуживает главную страницу.
func (h *Handler) Home(c *gin.Context) {
	page, err := h.service.Home(c.Request.Context())
	if err != nil {
		h.renderError(c, http.StatusInternalServerError, "Не удалось открыть главную страницу")
		return
	}
	page.Brand = h.brand

	h.renderCached(c, http.StatusOK, HomePageView(page))
}

// Landing обслуживает лендинг трека.
func (h *Handler) Landing(c *gin.Context) {
	page, err := h.service.Landing(c.Request.Context(), c.Param("examCode"))
	if err != nil {
		h.handlePageError(c, err)
		return
	}
	page.Brand = h.brand

	h.renderCached(c, http.StatusOK, LandingView(page))
}

// Bank обслуживает страницу банка заданий.
func (h *Handler) Bank(c *gin.Context) {
	page, err := h.service.Bank(c.Request.Context(), parseBankFilter(c))
	if err != nil {
		h.handlePageError(c, err)
		return
	}
	page.Brand = h.brand

	h.renderCached(c, http.StatusOK, BankView(page))
}

// BankFragment обслуживает HTMX-фрагмент банка.
func (h *Handler) BankFragment(c *gin.Context) {
	page, err := h.service.Bank(c.Request.Context(), parseBankFilter(c))
	if err != nil {
		h.handlePageError(c, err)
		return
	}
	page.Brand = h.brand

	h.renderCached(c, http.StatusOK, BankResultsView(page))
}

// Task обслуживает карточку задания.
func (h *Handler) Task(c *gin.Context) {
	taskID, err := uuid.Parse(c.Param("taskId"))
	if err != nil {
		h.renderError(c, http.StatusNotFound, "Задание не найдено")
		return
	}

	page, err := h.service.Task(c.Request.Context(), taskID)
	if err != nil {
		h.handlePageError(c, err)
		return
	}
	page.Brand = h.brand

	h.renderCached(c, http.StatusOK, TaskView(page))
}

// CheckTask проверяет ответ на задание банка.
func (h *Handler) CheckTask(c *gin.Context) {
	taskID, err := uuid.Parse(c.Param("taskId"))
	if err != nil {
		h.renderError(c, http.StatusNotFound, "Задание не найдено")
		return
	}

	result, err := h.service.CheckTask(c.Request.Context(), taskID, c.PostForm("answer"))
	if err != nil {
		if errors.Is(err, ErrUnsupported) {
			h.renderError(c, http.StatusUnprocessableEntity, "Для этого задания публичная проверка недоступна")
			return
		}
		h.handlePageError(c, err)
		return
	}

	page, err := h.service.Task(c.Request.Context(), taskID)
	if err != nil {
		h.handlePageError(c, err)
		return
	}
	page.Result = result
	page.Brand = h.brand

	h.render(c, http.StatusOK, TaskView(page))
}

// Demo обслуживает запуск демо-диагностики.
func (h *Handler) Demo(c *gin.Context) {
	examCode := strings.TrimSpace(c.Query("exam"))
	if examCode == "" {
		page, err := h.service.Home(c.Request.Context())
		if err != nil {
			h.renderError(c, http.StatusInternalServerError, "Не удалось открыть демо-диагностику")
			return
		}
		page.Brand = h.brand
		h.render(c, http.StatusOK, DemoChoiceView(page))
		return
	}

	start, err := h.service.StartDemo(c.Request.Context(), examCode)
	if err != nil {
		if errors.Is(err, ErrInsufficient) {
			page, homeErr := h.service.Home(c.Request.Context())
			if homeErr != nil {
				h.renderError(c, http.StatusInternalServerError, "Не удалось открыть демо-диагностику")
				return
			}
			h.render(c, http.StatusOK, DemoUnavailableView(page.Exams, h.brand))
			return
		}
		h.handlePageError(c, err)
		return
	}

	payload := &DemoPayload{ExamYearID: start.Year.ID, TaskIDs: make([]uuid.UUID, 0, len(start.Tasks))}
	for _, task := range start.Tasks {
		payload.TaskIDs = append(payload.TaskIDs, task.ID)
	}
	token, err := h.signer.Sign(payload)
	if err != nil {
		h.renderError(c, http.StatusInternalServerError, "Не удалось запустить демо-диагностику")
		return
	}

	start.Brand = h.brand
	h.setDemoCookie(c, token, int(DemoTTL.Seconds()))
	h.render(c, http.StatusOK, DemoView(start, token))
}

// CheckDemo проверяет ответы демо-диагностики.
func (h *Handler) CheckDemo(c *gin.Context) {
	payload, err := h.signer.Verify(c.PostForm("demo_token"))
	if err != nil {
		h.renderError(c, http.StatusUnprocessableEntity, "Данные демо-диагностики истекли или изменены. Запустите её заново")
		return
	}

	answers := make(map[uuid.UUID]string, len(payload.TaskIDs))
	for _, taskID := range payload.TaskIDs {
		answers[taskID] = c.PostForm("answer_" + taskID.String())
	}

	result, err := h.service.CheckDemo(c.Request.Context(), payload, answers)
	if err != nil {
		if errors.Is(err, ErrInvalidSession) {
			h.renderError(c, http.StatusUnprocessableEntity, "Данные демо-диагностики истекли или изменены. Запустите её заново")
			return
		}
		h.renderError(c, http.StatusInternalServerError, "Не удалось проверить демо-диагностику")
		return
	}

	h.clearDemoCookie(c)
	h.render(c, http.StatusOK, DemoResultView(result, h.brand))
}

// Articles обслуживает список статей.
func (h *Handler) Articles(c *gin.Context) {
	var cursor *time.Time
	if value := strings.TrimSpace(c.Query("cursor")); value != "" {
		parsed, err := time.Parse(time.RFC3339Nano, value)
		if err == nil {
			cursor = &parsed
		}
	}

	articles, err := h.service.Articles(c.Request.Context(), cursor, 20)
	if err != nil {
		h.renderError(c, http.StatusInternalServerError, "Не удалось открыть статьи")
		return
	}

	next := ""
	if len(articles) == 20 && articles[len(articles)-1].PublishedAt != nil {
		next = articles[len(articles)-1].PublishedAt.Format(time.RFC3339Nano)
	}

	h.renderCached(c, http.StatusOK, ArticlesView(articles, next, h.brand))
}

// Article обслуживает страницу статьи.
func (h *Handler) Article(c *gin.Context) {
	view, err := h.service.Article(c.Request.Context(), c.Param("slug"))
	if err != nil {
		h.handlePageError(c, err)
		return
	}
	view.Brand = h.brand

	h.renderCached(c, http.StatusOK, ArticlePageView(view))
}

// LeadFormFragment возвращает HTMX-фрагмент лид-формы.
func (h *Handler) LeadFormFragment(c *gin.Context) {
	h.render(c, http.StatusOK, leadForm("/hx/leads", c.Query("source"), nil, false))
}

// SubmitLead принимает заявку с полной формы.
func (h *Handler) SubmitLead(c *gin.Context) {
	h.submitLead(c, false)
}

// SubmitLeadHX принимает заявку из HTMX-формы.
func (h *Handler) SubmitLeadHX(c *gin.Context) {
	h.submitLead(c, true)
}

func (h *Handler) submitLead(c *gin.Context, fragment bool) {
	source := c.PostForm("source")
	err := h.service.SubmitLead(c.Request.Context(), LeadInput{
		Name:    c.PostForm("name"),
		Contact: c.PostForm("contact"),
		Comment: c.PostForm("comment"),
		Source:  source,
		Website: c.PostForm("website"),
	})
	if err != nil {
		fields := map[string]string{}
		var validation *ValidationError
		if errors.As(err, &validation) {
			fields[validation.Field] = validation.Message
			if fragment {
				h.render(c, http.StatusUnprocessableEntity, leadForm("/hx/leads", source, fields, false))
				return
			}
			h.render(c, http.StatusUnprocessableEntity, LeadFormView(source, fields, false, h.brand))
			return
		}

		h.renderError(c, http.StatusInternalServerError, "Не удалось отправить заявку")
		return
	}

	if fragment {
		h.render(c, http.StatusOK, leadForm("/hx/leads", source, nil, true))
		return
	}
	h.render(c, http.StatusOK, LeadFormView(source, nil, true, h.brand))
}

// Sitemap отдаёт sitemap.xml.
func (h *Handler) Sitemap(c *gin.Context) {
	entries, err := h.service.Sitemap(c.Request.Context())
	if err != nil {
		h.renderError(c, http.StatusInternalServerError, "Не удалось построить sitemap")
		return
	}

	type urlEntry struct {
		Loc     string `xml:"loc"`
		LastMod string `xml:"lastmod,omitempty"`
	}
	type urlSet struct {
		XMLName xml.Name   `xml:"urlset"`
		XMLNS   string     `xml:"xmlns,attr"`
		URLs    []urlEntry `xml:"url"`
	}

	payload := urlSet{XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9"}
	for _, entry := range entries {
		item := urlEntry{Loc: h.absoluteURL(entry.Loc)}
		if !entry.LastMod.IsZero() {
			item.LastMod = entry.LastMod.UTC().Format(time.RFC3339)
		}
		payload.URLs = append(payload.URLs, item)
	}

	body, err := xml.MarshalIndent(payload, "", "  ")
	if err != nil {
		h.renderError(c, http.StatusInternalServerError, "Не удалось построить sitemap")
		return
	}
	body = append([]byte(xml.Header), body...)
	c.Data(http.StatusOK, "application/xml; charset=utf-8", body)
}

// Robots отдаёт robots.txt.
func (h *Handler) Robots(c *gin.Context) {
	body := strings.Join([]string{
		"User-agent: *",
		"Disallow: /admin",
		"Disallow: /account",
		"Disallow: /login",
		"Disallow: /register",
		"Disallow: /password",
		"Disallow: /api/",
		"Disallow: /hx/",
		"Allow: /",
		"",
		"Sitemap: " + h.baseURL + "/sitemap.xml",
		"",
	}, "\n")
	c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(body))
}

// Redirect обслуживает SEO-алиасы.
func (h *Handler) Redirect(c *gin.Context) {
	redirect, err := h.service.ResolveRedirect(c.Request.Context(), "/"+c.Param("alias"))
	if err != nil {
		h.renderError(c, http.StatusNotFound, "Страница не найдена")
		return
	}

	c.Redirect(redirect.StatusCode, redirect.ToPath)
}

func (h *Handler) handlePageError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		h.renderError(c, http.StatusNotFound, "Страница не найдена")
	case errors.Is(err, ErrUnsupported):
		h.renderError(c, http.StatusUnprocessableEntity, "Для этого задания публичная проверка недоступна")
	default:
		h.logger.Error("ошибка публичной страницы", "path", c.Request.URL.Path, "error", err)
		h.renderError(c, http.StatusInternalServerError, "Внутренняя ошибка")
	}
}

func (h *Handler) renderCached(c *gin.Context, status int, component templ.Component) {
	if h.cache == nil || auth.SessionTokenFromRequest(c) != "" {
		h.render(c, status, component)
		return
	}

	key := cacheKey(c.Request)
	if body, ok := h.cache.Get(key); ok {
		h.writeHTML(c, status, body, true)
		return
	}

	var buf bytes.Buffer
	if err := component.Render(c.Request.Context(), &buf); err != nil {
		h.logger.Error("не удалось отрендерить публичную страницу", "path", c.Request.URL.Path, "error", err)
		c.Status(http.StatusInternalServerError)
		return
	}
	h.cache.Set(key, buf.Bytes())
	h.writeHTML(c, status, buf.Bytes(), true)
}

func (h *Handler) render(c *gin.Context, status int, component templ.Component) {
	var buf bytes.Buffer
	if err := component.Render(c.Request.Context(), &buf); err != nil {
		h.logger.Error("не удалось отрендерить публичную страницу", "path", c.Request.URL.Path, "error", err)
		c.Status(http.StatusInternalServerError)
		return
	}
	h.writeHTML(c, status, buf.Bytes(), false)
}

func (h *Handler) renderError(c *gin.Context, status int, message string) {
	h.render(c, status, ErrorView(status, message, h.brand))
}

func (h *Handler) writeHTML(c *gin.Context, status int, body []byte, cacheable bool) {
	if cacheable {
		c.Header("Cache-Control", "public, max-age=60")
	} else {
		c.Header("Cache-Control", "no-store")
	}

	token := ""
	if h.csrf != nil {
		token = h.csrf.Token(c)
	}
	body = bytes.ReplaceAll(body, []byte("__CSRF__"), []byte(token))
	c.Data(status, "text/html; charset=utf-8", body)
}

func (h *Handler) setDemoCookie(c *gin.Context, token string, maxAge int) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     DemoCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   h.secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *Handler) clearDemoCookie(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     DemoCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *Handler) absoluteURL(path string) string {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}

	return h.baseURL + path
}

func parseBankFilter(c *gin.Context) BankFilter {
	filter := BankFilter{
		ExamCode: strings.TrimSpace(c.Query("exam")),
		Limit:    parseLimit(c.Query("limit")),
	}
	if value := strings.TrimSpace(c.Query("kim")); value != "" {
		if number, err := strconv.Atoi(value); err == nil && number > 0 {
			filter.KimNumber = number
		}
	}
	if value := strings.TrimSpace(c.Query("topic")); value != "" {
		if id, err := uuid.Parse(value); err == nil {
			filter.TopicID = &id
		}
	}
	if value := strings.TrimSpace(c.Query("cursor")); value != "" {
		if id, err := uuid.Parse(value); err == nil {
			filter.Cursor = id
		}
	}

	return filter
}

func parseLimit(value string) int {
	const defaultLimit = 20
	if value == "" {
		return defaultLimit
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return defaultLimit
	}
	if parsed > 50 {
		return 50
	}

	return parsed
}

func cacheKey(request *http.Request) string {
	query := request.URL.Query()
	keys := make([]string, 0, len(query))
	for key := range query {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	values := url.Values{}
	for _, key := range keys {
		for _, value := range query[key] {
			values.Add(key, value)
		}
	}
	if encoded := values.Encode(); encoded != "" {
		return request.URL.Path + "?" + encoded
	}

	return request.URL.Path
}
