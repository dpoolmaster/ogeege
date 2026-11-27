package publicsite

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"oge-ege/internal/content"
	"oge-ege/internal/shared/httpserver"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

type handlerFixture struct {
	router  *gin.Engine
	handler *Handler
	signer  *DemoSigner
	clock   *testClock
	cache   *Cache
}

func newFixture(repo Repository, provider ContentProvider, recipients RecipientProvider, notifier Notifier) *handlerFixture {
	gin.SetMode(gin.TestMode)
	clk := &testClock{now: time.Date(2026, time.November, 27, 12, 0, 0, 0, time.UTC)}
	service := NewService(repo, nil, testTx{}, clk, provider, recipients, notifier, testLogger())
	csrf := httpserver.NewCSRF("secret", false, nil)
	signer := NewDemoSigner("secret", DemoTTL, clk)
	cache := NewCache(time.Minute, 100)
	handler := NewHandler(service, csrf, signer, cache, testLogger(), false, "https://example.ru", "Верно")

	router := gin.New()
	router.Use(csrf.Middleware())
	handler.Register(router, httpserver.NewRateLimiter(100, time.Minute))

	return &handlerFixture{router: router, handler: handler, signer: signer, clock: clk, cache: cache}
}

func perform(router http.Handler, method, target string, form url.Values, cookies map[string]string) *httptest.ResponseRecorder {
	var body io.Reader
	if form != nil {
		body = strings.NewReader(form.Encode())
	}
	request := httptest.NewRequest(method, target, body)
	if form != nil {
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	for name, value := range cookies {
		request.AddCookie(&http.Cookie{Name: name, Value: value})
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, request)

	return rec
}

func csrfFrom(rec *httptest.ResponseRecorder) string {
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == "csrf_token" {
			return cookie.Value
		}
	}

	return ""
}

func publishedExam() content.Exam {
	return content.Exam{
		ID: uuid.Must(uuid.NewV7()), Code: "oge_math", Title: "ОГЭ математика",
		Subject: "math", Status: content.StatusPublished, UpdatedAt: time.Now(),
	}
}

func publishedTask() content.Task {
	return content.Task{
		ID: uuid.Must(uuid.NewV7()), Status: content.StatusPublished, IsPublic: true,
		StatementMD: "Вычислите 1 + 1", AnswerSpec: json.RawMessage(`{"kind":"number","accepted":["2"]}`),
		AnswerDisplay: "2",
	}
}

func TestHandlerHomeAndLanding(t *testing.T) {
	exam := publishedExam()
	year := content.ExamYear{ID: uuid.Must(uuid.NewV7()), ExamID: exam.ID, Year: 2026, Status: content.StatusPublished}
	fixture := newFixture(&fakeRepo{}, &fakeContent{exam: &exam, year: &year}, fakeRecipients{}, &fakeNotifier{})

	rec := perform(fixture.router, http.MethodGet, "/", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("главная: %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "<title>") || strings.Contains(rec.Body.String(), "<script>") {
		t.Fatalf("некорректный HTML главной: %s", rec.Body.String())
	}

	rec = perform(fixture.router, http.MethodGet, "/tracks/oge_math", nil, nil)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "ОГЭ математика") {
		t.Fatalf("лендинг: %d %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "__CSRF__") {
		t.Fatal("CSRF-плейсхолдер не подставлен")
	}

	draft := exam
	draft.Status = content.StatusDraft
	fixture = newFixture(&fakeRepo{}, &fakeContent{exam: &draft}, fakeRecipients{}, &fakeNotifier{})
	rec = perform(fixture.router, http.MethodGet, "/tracks/oge_math", nil, nil)
	if rec.Code != http.StatusNotFound || !strings.Contains(rec.Body.String(), "<html") {
		t.Fatalf("черновик: %d %s", rec.Code, rec.Body.String())
	}
}

func TestHandlerTaskAndCheck(t *testing.T) {
	task := publishedTask()
	fixture := newFixture(&fakeRepo{}, &fakeContent{task: &task}, fakeRecipients{}, &fakeNotifier{})

	rec := perform(fixture.router, http.MethodGet, "/bank/"+task.ID.String(), nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("карточка: %d", rec.Code)
	}
	body := rec.Body.String()
	if strings.Contains(body, `"accepted"`) || strings.Contains(body, "Правильный ответ") {
		t.Fatalf("до проверки раскрыт ответ: %s", body)
	}

	csrf := csrfFrom(rec)
	form := url.Values{"csrf_token": {csrf}, "answer": {"2"}}
	rec = perform(fixture.router, http.MethodPost, "/bank/"+task.ID.String()+"/check", form, map[string]string{"csrf_token": csrf})
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Верно") || !strings.Contains(rec.Body.String(), "Правильный ответ: 2") {
		t.Fatalf("проверка ответа: %d %s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("результат проверки не должен кэшироваться: %q", rec.Header().Get("Cache-Control"))
	}

	unsupported := task
	unsupported.AnswerSpec = json.RawMessage(`{"kind":"choice","accepted":["2"]}`)
	fixture = newFixture(&fakeRepo{}, &fakeContent{task: &unsupported}, fakeRecipients{}, &fakeNotifier{})
	rec = perform(fixture.router, http.MethodGet, "/bank/"+task.ID.String(), nil, nil)
	csrf = csrfFrom(rec)
	form = url.Values{"csrf_token": {csrf}, "answer": {"2"}}
	rec = perform(fixture.router, http.MethodPost, "/bank/"+task.ID.String()+"/check", form, map[string]string{"csrf_token": csrf})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("неподдерживаемый тип: %d", rec.Code)
	}
}

func TestHandlerBank(t *testing.T) {
	exam := publishedExam()
	year := content.ExamYear{ID: uuid.Must(uuid.NewV7()), ExamID: exam.ID, Year: 2026, Status: content.StatusPublished}
	task := publishedTask()
	kim := content.KimNumber{ID: uuid.Must(uuid.NewV7()), ExamYearID: year.ID, Number: 1}
	fixture := newFixture(&fakeRepo{}, &fakeContent{exam: &exam, year: &year, kim: &kim, tasks: []content.Task{task}}, fakeRecipients{}, &fakeNotifier{})

	rec := perform(fixture.router, http.MethodGet, "/bank?exam=oge_math&kim=1", nil, nil)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), task.StatementMD) {
		t.Fatalf("банк: %d %s", rec.Code, rec.Body.String())
	}
	rec = perform(fixture.router, http.MethodGet, "/hx/bank?exam=oge_math", nil, nil)
	if rec.Code != http.StatusOK || strings.Contains(rec.Body.String(), "<html") {
		t.Fatalf("HTMX-фрагмент банка: %d", rec.Code)
	}
}

func TestHandlerDemo(t *testing.T) {
	exam := publishedExam()
	year := content.ExamYear{ID: uuid.Must(uuid.NewV7()), ExamID: exam.ID, Year: 2026, Status: content.StatusPublished}
	tasks := make([]content.Task, 0, 5)
	for i := 0; i < 5; i++ {
		task := publishedTask()
		task.StatementMD = "Задание " + task.ID.String()
		tasks = append(tasks, task)
	}
	fixture := newFixture(&fakeRepo{}, &fakeContent{exam: &exam, year: &year, tasks: tasks}, fakeRecipients{}, &fakeNotifier{})

	rec := perform(fixture.router, http.MethodGet, "/diagnostics/demo?exam=oge_math", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("демо: %d", rec.Code)
	}
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("демо не должно кэшироваться: %q", rec.Header().Get("Cache-Control"))
	}
	token := ""
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == DemoCookieName {
			token = cookie.Value
		}
	}
	if token == "" {
		t.Fatal("cookie демо не установлена")
	}

	csrf := csrfFrom(rec)
	form := url.Values{"csrf_token": {csrf}, "demo_token": {token}}
	for _, task := range tasks {
		form.Set("answer_"+task.ID.String(), "2")
	}
	rec = perform(fixture.router, http.MethodPost, "/diagnostics/demo/check", form, map[string]string{"csrf_token": csrf, DemoCookieName: token})
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Результат демо-диагностики") {
		t.Fatalf("результат демо: %d %s", rec.Code, rec.Body.String())
	}
	cleared := false
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == DemoCookieName && cookie.MaxAge < 0 {
			cleared = true
		}
	}
	if !cleared {
		t.Fatal("cookie демо не очищена")
	}

	form.Set("demo_token", token+"x")
	rec = perform(fixture.router, http.MethodPost, "/diagnostics/demo/check", form, map[string]string{"csrf_token": csrf})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("изменённый токен: %d", rec.Code)
	}

	fixture.clock.now = fixture.clock.now.Add(31 * time.Minute)
	form.Set("demo_token", token)
	rec = perform(fixture.router, http.MethodPost, "/diagnostics/demo/check", form, map[string]string{"csrf_token": csrf})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("истёкший токен: %d", rec.Code)
	}
}

func TestHandlerArticles(t *testing.T) {
	now := time.Now()
	published := Article{ID: uuid.Must(uuid.NewV7()), Slug: "post", Title: "Опубликованная", ContentMD: "## Текст", Status: ArticlePublished, PublishedAt: &now}
	draft := &Article{ID: uuid.Must(uuid.NewV7()), Slug: "draft", Title: "Черновик", ContentMD: "x", Status: ArticleDraft}
	repo := &fakeRepo{
		articles:  map[string]*Article{"post": &published, "draft": draft},
		published: []Article{published},
	}
	fixture := newFixture(repo, &fakeContent{}, fakeRecipients{}, &fakeNotifier{})

	rec := perform(fixture.router, http.MethodGet, "/articles", nil, nil)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Опубликованная") || strings.Contains(rec.Body.String(), "Черновик") {
		t.Fatalf("список статей: %d %s", rec.Code, rec.Body.String())
	}
	rec = perform(fixture.router, http.MethodGet, "/articles/post", nil, nil)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "<h2") {
		t.Fatalf("страница статьи: %d %s", rec.Code, rec.Body.String())
	}
	rec = perform(fixture.router, http.MethodGet, "/articles/draft", nil, nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("черновик статьи: %d", rec.Code)
	}
}

func TestHandlerLeads(t *testing.T) {
	repo := &fakeRepo{}
	notifier := &fakeNotifier{}
	fixture := newFixture(repo, &fakeContent{}, fakeRecipients{emails: []string{"admin@example.ru"}}, notifier)

	rec := perform(fixture.router, http.MethodGet, "/", nil, nil)
	csrf := csrfFrom(rec)
	form := url.Values{"csrf_token": {csrf}, "name": {"Иван"}, "contact": {"ivan@example.ru"}, "comment": {"Нужна помощь"}, "source": {"/oge"}}
	rec = perform(fixture.router, http.MethodPost, "/leads", form, map[string]string{"csrf_token": csrf})
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Заявка отправлена") {
		t.Fatalf("лид: %d %s", rec.Code, rec.Body.String())
	}
	if len(repo.leads) != 1 || len(notifier.messages) != 1 {
		t.Fatalf("заявка или уведомление не созданы: %d %d", len(repo.leads), len(notifier.messages))
	}

	form.Set("name", "И")
	rec = perform(fixture.router, http.MethodPost, "/leads", form, map[string]string{"csrf_token": csrf})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("ошибка валидации: %d", rec.Code)
	}

	form.Set("name", "Иван")
	form.Set("website", "spam")
	rec = perform(fixture.router, http.MethodPost, "/leads", form, map[string]string{"csrf_token": csrf})
	if rec.Code != http.StatusOK || len(repo.leads) != 1 {
		t.Fatalf("honeypot: %d, заявок %d", rec.Code, len(repo.leads))
	}

	rec = perform(fixture.router, http.MethodPost, "/leads", url.Values{"name": {"Иван"}, "contact": {"ivan@example.ru"}}, nil)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("без CSRF: %d", rec.Code)
	}

	rec = perform(fixture.router, http.MethodPost, "/hx/leads", form, map[string]string{"csrf_token": csrf})
	if rec.Code != http.StatusOK || strings.Contains(rec.Body.String(), "<html") {
		t.Fatalf("HTMX-лид: %d %s", rec.Code, rec.Body.String())
	}
}

func TestHandlerLeadRateLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := newTestService(&fakeRepo{}, &fakeContent{}, fakeRecipients{}, &fakeNotifier{})
	csrf := httpserver.NewCSRF("secret", false, nil)
	handler := NewHandler(service, csrf, NewDemoSigner("secret", DemoTTL, &testClock{now: time.Now()}), NewCache(time.Minute, 10), testLogger(), false, "https://example.ru", "Верно")
	router := gin.New()
	router.Use(csrf.Middleware())
	handler.Register(router, httpserver.NewRateLimiter(1, time.Minute))

	rec := perform(router, http.MethodGet, "/", nil, nil)
	csrfValue := csrfFrom(rec)
	form := url.Values{"csrf_token": {csrfValue}, "name": {"Иван"}, "contact": {"ivan@example.ru"}}
	perform(router, http.MethodPost, "/leads", form, map[string]string{"csrf_token": csrfValue})
	rec = perform(router, http.MethodPost, "/leads", form, map[string]string{"csrf_token": csrfValue})
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("rate limit: %d", rec.Code)
	}
}

func TestHandlerSitemapRobotsAndAssets(t *testing.T) {
	exam := publishedExam()
	now := time.Now()
	repo := &fakeRepo{
		published: []Article{{ID: uuid.Must(uuid.NewV7()), Slug: "post", Status: ArticlePublished, PublishedAt: &now}},
		seo: map[string]*SEOPage{
			"/":     {ID: uuid.Must(uuid.NewV7()), Path: "/", UpdatedAt: now},
			"/bank": {ID: uuid.Must(uuid.NewV7()), Path: "/bank", UpdatedAt: now},
		},
	}
	fixture := newFixture(repo, &fakeContent{exams: []content.Exam{exam}}, fakeRecipients{}, &fakeNotifier{})

	rec := perform(fixture.router, http.MethodGet, "/sitemap.xml", nil, nil)
	if rec.Code != http.StatusOK || !strings.HasPrefix(rec.Header().Get("Content-Type"), "application/xml") {
		t.Fatalf("sitemap: %d %q", rec.Code, rec.Header().Get("Content-Type"))
	}
	if !strings.Contains(rec.Body.String(), "/tracks/oge_math") || !strings.Contains(rec.Body.String(), "/articles/post") {
		t.Fatalf("sitemap не содержит опубликованные пути: %s", rec.Body.String())
	}

	rec = perform(fixture.router, http.MethodGet, "/robots.txt", nil, nil)
	if rec.Code != http.StatusOK || !strings.HasPrefix(rec.Header().Get("Content-Type"), "text/plain") {
		t.Fatalf("robots: %d %q", rec.Code, rec.Header().Get("Content-Type"))
	}
	if !strings.Contains(rec.Body.String(), "Disallow: /admin") || !strings.Contains(rec.Body.String(), "Sitemap: https://example.ru/sitemap.xml") {
		t.Fatalf("robots неверен: %s", rec.Body.String())
	}

	rec = perform(fixture.router, http.MethodGet, "/static/public/htmx.min.js", nil, nil)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Header().Get("Content-Type"), "javascript") {
		t.Fatalf("htmx: %d %q", rec.Code, rec.Header().Get("Content-Type"))
	}
	if rec.Header().Get("Cache-Control") == "" {
		t.Fatal("нет Cache-Control для статики")
	}
}

func TestHandlerRedirectsAndErrors(t *testing.T) {
	repo := &fakeRepo{redirects: map[string]*Redirect{
		"/oge": {ID: uuid.Must(uuid.NewV7()), FromPath: "/oge", ToPath: "/tracks/oge_math", StatusCode: 301},
	}}
	exam := publishedExam()
	fixture := newFixture(repo, &fakeContent{exam: &exam}, fakeRecipients{}, &fakeNotifier{})

	rec := perform(fixture.router, http.MethodGet, "/oge", nil, nil)
	if rec.Code != http.StatusMovedPermanently || rec.Header().Get("Location") != "/tracks/oge_math" {
		t.Fatalf("редирект: %d %q", rec.Code, rec.Header().Get("Location"))
	}
	rec = perform(fixture.router, http.MethodGet, "/unknown", nil, nil)
	if rec.Code != http.StatusNotFound || !strings.Contains(rec.Body.String(), "<html") {
		t.Fatalf("неизвестный путь: %d", rec.Code)
	}

	failing := &failingProvider{fakeContent: fakeContent{exam: &exam}}
	fixture = newFixture(&fakeRepo{}, failing, fakeRecipients{}, &fakeNotifier{})
	rec = perform(fixture.router, http.MethodGet, "/tracks/oge_math", nil, nil)
	if rec.Code != http.StatusInternalServerError || !strings.Contains(rec.Body.String(), "<html") {
		t.Fatalf("500: %d", rec.Code)
	}
}

type failingProvider struct {
	fakeContent
}

func (p *failingProvider) GetExamByCode(context.Context, string) (*content.Exam, error) {
	return nil, errors.New("база недоступна")
}

func TestHandlerCacheAndCSRF(t *testing.T) {
	exam := publishedExam()
	year := content.ExamYear{ID: uuid.Must(uuid.NewV7()), ExamID: exam.ID, Year: 2026, Status: content.StatusPublished}
	provider := &countingProvider{fakeContent: fakeContent{exam: &exam, year: &year}}
	fixture := newFixture(&fakeRepo{}, provider, fakeRecipients{}, &fakeNotifier{})

	first := perform(fixture.router, http.MethodGet, "/tracks/oge_math", nil, nil)
	second := perform(fixture.router, http.MethodGet, "/tracks/oge_math", nil, nil)
	if first.Code != http.StatusOK || second.Code != http.StatusOK {
		t.Fatalf("коды: %d %d", first.Code, second.Code)
	}
	if strings.Contains(second.Body.String(), "__CSRF__") {
		t.Fatal("в закэшированной странице не подставлен CSRF")
	}
	csrf := csrfFrom(second)
	if csrf == "" || !strings.Contains(second.Body.String(), csrf) {
		t.Fatal("CSRF-токен страницы не совпадает с cookie")
	}
}

func TestRenderCachedHitAndSessionBypass(t *testing.T) {
	gin.SetMode(gin.TestMode)
	csrf := httpserver.NewCSRF("secret", false, nil)
	handler := &Handler{
		csrf:   csrf,
		cache:  NewCache(time.Minute, 10),
		logger: testLogger(),
	}
	renders := 0
	component := templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
		renders++
		_, err := io.WriteString(w, "<form><input name=\"csrf_token\" value=\"__CSRF__\"/></form>")
		return err
	})

	render := func(cookies map[string]string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(rec)
		ctx.Request = httptest.NewRequest(http.MethodGet, "/page", nil)
		for name, value := range cookies {
			ctx.Request.AddCookie(&http.Cookie{Name: name, Value: value})
		}
		handler.renderCached(ctx, http.StatusOK, component)
		return rec
	}

	first := render(nil)
	second := render(nil)
	if renders != 1 {
		t.Fatalf("кэш не использован: рендеров %d", renders)
	}
	if strings.Contains(second.Body.String(), "__CSRF__") {
		t.Fatal("CSRF-плейсхолдер не заменён")
	}
	token := csrfFrom(second)
	if token == "" || !strings.Contains(second.Body.String(), token) {
		t.Fatal("CSRF-токен не совпадает с cookie")
	}
	if first.Header().Get("Cache-Control") != "public, max-age=60" {
		t.Fatalf("неверный Cache-Control: %q", first.Header().Get("Cache-Control"))
	}

	render(map[string]string{"session": "token"})
	if renders != 2 {
		t.Fatalf("кэш не обойдён при сессии: рендеров %d", renders)
	}
}

type countingProvider struct {
	fakeContent
	calls int
}

func (p *countingProvider) GetExamByCode(ctx context.Context, code string) (*content.Exam, error) {
	p.calls++
	return p.fakeContent.GetExamByCode(ctx, code)
}

func landingFixture(t *testing.T) (*handlerFixture, content.Task) {
	t.Helper()

	exam := publishedExam()
	year := content.ExamYear{ID: uuid.Must(uuid.NewV7()), ExamID: exam.ID, Year: 2026, Status: content.StatusPublished}
	kim := content.KimNumber{ID: uuid.Must(uuid.NewV7()), ExamYearID: year.ID, Number: 1}
	task := publishedTask()
	task.KimNumberID = kim.ID
	task.StatementMD = "Вычислите 1 + 1"
	fixture := newFixture(&fakeRepo{}, &fakeContent{
		exam:    &exam,
		year:    &year,
		kimList: []content.KimNumber{kim},
		kimByID: map[uuid.UUID]content.KimNumber{kim.ID: kim},
		tasks:   []content.Task{task},
		task:    &task,
	}, fakeRecipients{}, &fakeNotifier{})

	return fixture, task
}

func TestHandlerLandingCheckBlockAndFragment(t *testing.T) {
	fixture, task := landingFixture(t)

	rec := perform(fixture.router, http.MethodGet, "/tracks/oge_math", nil, nil)
	body := rec.Body.String()
	if rec.Code != http.StatusOK || !strings.Contains(body, "Попробуйте задание") {
		t.Fatalf("блок проверки на лендинге: %d %s", rec.Code, body)
	}
	for _, part := range []string{
		`action="/bank/` + task.ID.String() + `/check"`,
		`hx-post="/hx/tasks/` + task.ID.String() + `/check"`,
		`hx-target="#landing-check-result"`,
		`name="csrf_token"`,
	} {
		if !strings.Contains(body, part) {
			t.Fatalf("форма проверки не содержит %q", part)
		}
	}
	if strings.Contains(body, "__CSRF__") {
		t.Fatal("CSRF-плейсхолдер не подставлен")
	}

	csrf := csrfFrom(rec)
	form := url.Values{"csrf_token": {csrf}, "answer": {"2"}}
	rec = perform(fixture.router, http.MethodPost, "/hx/tasks/"+task.ID.String()+"/check", form, map[string]string{"csrf_token": csrf})
	body = rec.Body.String()
	if rec.Code != http.StatusOK || strings.Contains(body, "<html") {
		t.Fatalf("фрагмент верного ответа: %d %s", rec.Code, body)
	}
	if !strings.Contains(body, `role="status"`) || !strings.Contains(body, "Верно") {
		t.Fatalf("верный ответ не объявлен: %s", body)
	}
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("фрагмент проверки закэширован: %q", rec.Header().Get("Cache-Control"))
	}

	form.Set("answer", "3")
	rec = perform(fixture.router, http.MethodPost, "/hx/tasks/"+task.ID.String()+"/check", form, map[string]string{"csrf_token": csrf})
	body = rec.Body.String()
	if rec.Code != http.StatusOK || !strings.Contains(body, `role="alert"`) || !strings.Contains(body, "Неверно") || !strings.Contains(body, "Правильный ответ: 2") {
		t.Fatalf("неверный ответ: %d %s", rec.Code, body)
	}

	form.Set("answer", " ")
	rec = perform(fixture.router, http.MethodPost, "/hx/tasks/"+task.ID.String()+"/check", form, map[string]string{"csrf_token": csrf})
	if rec.Code != http.StatusUnprocessableEntity || !strings.Contains(rec.Body.String(), "Введите ответ") {
		t.Fatalf("пустой ответ: %d %s", rec.Code, rec.Body.String())
	}

	rec = perform(fixture.router, http.MethodPost, "/hx/tasks/"+task.ID.String()+"/check", url.Values{"answer": {"2"}}, nil)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("без CSRF: %d", rec.Code)
	}

	unsupported := task
	unsupported.AnswerSpec = json.RawMessage(`{"kind":"choice","accepted":["2"]}`)
	fixture = newFixture(&fakeRepo{}, &fakeContent{task: &unsupported}, fakeRecipients{}, &fakeNotifier{})
	rec = perform(fixture.router, http.MethodGet, "/", nil, nil)
	csrf = csrfFrom(rec)
	rec = perform(fixture.router, http.MethodPost, "/hx/tasks/"+task.ID.String()+"/check", url.Values{"csrf_token": {csrf}, "answer": {"2"}}, map[string]string{"csrf_token": csrf})
	if rec.Code != http.StatusUnprocessableEntity || !strings.Contains(rec.Body.String(), "недоступна") {
		t.Fatalf("неподдерживаемый тип: %d %s", rec.Code, rec.Body.String())
	}
}

func TestHandlerLandingWithoutTask(t *testing.T) {
	exam := publishedExam()
	year := content.ExamYear{ID: uuid.Must(uuid.NewV7()), ExamID: exam.ID, Year: 2026, Status: content.StatusPublished}
	fixture := newFixture(&fakeRepo{}, &fakeContent{exam: &exam, year: &year}, fakeRecipients{}, &fakeNotifier{})

	rec := perform(fixture.router, http.MethodGet, "/tracks/oge_math", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("лендинг без задания: %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "Попробуйте задание") {
		t.Fatal("блок проверки показан без задания")
	}
}

func demoFixture(t *testing.T) (*handlerFixture, []content.Task) {
	t.Helper()

	exam := publishedExam()
	year := content.ExamYear{ID: uuid.Must(uuid.NewV7()), ExamID: exam.ID, Year: 2026, Status: content.StatusPublished}
	tasks := make([]content.Task, 0, 5)
	for i := 0; i < 5; i++ {
		task := publishedTask()
		task.StatementMD = "Задание " + task.ID.String()
		tasks = append(tasks, task)
	}
	fixture := newFixture(&fakeRepo{}, &fakeContent{exam: &exam, year: &year, tasks: tasks}, fakeRecipients{}, &fakeNotifier{})

	return fixture, tasks
}

func demoToken(rec *httptest.ResponseRecorder) string {
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == DemoCookieName {
			return cookie.Value
		}
	}

	return ""
}

func TestHandlerDemoStepFlow(t *testing.T) {
	fixture, tasks := demoFixture(t)

	rec := perform(fixture.router, http.MethodGet, "/diagnostics/demo?exam=oge_math", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("запуск демо: %d", rec.Code)
	}
	token := demoToken(rec)
	csrf := csrfFrom(rec)
	body := rec.Body.String()
	if token == "" || csrf == "" {
		t.Fatal("демо не выдало токен или CSRF")
	}
	for _, part := range []string{`id="demo-step"`, `data-autofocus`, "Шаг", "1 из 5", tasks[0].StatementMD} {
		if !strings.Contains(body, part) {
			t.Fatalf("первый шаг не содержит %q: %s", part, body)
		}
	}
	if strings.Contains(body, tasks[1].StatementMD) {
		t.Fatal("на первом шаге показано второе задание")
	}

	form := url.Values{
		"csrf_token":                     {csrf},
		"demo_token":                     {token},
		"step":                           {"2"},
		"answer_" + tasks[0].ID.String(): {"2"},
	}
	rec = perform(fixture.router, http.MethodPost, "/hx/demo/step", form, map[string]string{"csrf_token": csrf, DemoCookieName: token})
	body = rec.Body.String()
	if rec.Code != http.StatusOK || strings.Contains(body, "<html") {
		t.Fatalf("фрагмент шага: %d %s", rec.Code, body)
	}
	if !strings.Contains(body, tasks[1].StatementMD) || !strings.Contains(body, `name="answer_`+tasks[0].ID.String()+`" value="2"`) {
		t.Fatalf("второй шаг не сохранил ответ: %s", body)
	}

	form.Set("step", "1")
	form.Set("answer_"+tasks[1].ID.String(), "3")
	rec = perform(fixture.router, http.MethodPost, "/diagnostics/demo/step", form, map[string]string{"csrf_token": csrf, DemoCookieName: token})
	body = rec.Body.String()
	if rec.Code != http.StatusOK || !strings.Contains(body, tasks[0].StatementMD) || !strings.Contains(body, `value="2"`) {
		t.Fatalf("возврат назад не сохранил ответ: %d %s", rec.Code, body)
	}

	form.Set("step", "99")
	rec = perform(fixture.router, http.MethodPost, "/hx/demo/step", form, map[string]string{"csrf_token": csrf, DemoCookieName: token})
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), tasks[4].StatementMD) {
		t.Fatalf("шаг вне диапазона не ограничен: %d %s", rec.Code, rec.Body.String())
	}

	checkForm := url.Values{"csrf_token": {csrf}, "demo_token": {token}}
	for i, task := range tasks {
		checkForm.Set("answer_"+task.ID.String(), []string{"2", "3", "2", "3", "2"}[i])
	}
	rec = perform(fixture.router, http.MethodPost, "/diagnostics/demo/check", checkForm, map[string]string{"csrf_token": csrf, DemoCookieName: token})
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Результат демо-диагностики") || !strings.Contains(rec.Body.String(), "Верно 3 из 5") {
		t.Fatalf("итог демо: %d %s", rec.Code, rec.Body.String())
	}

	form.Set("demo_token", token+"x")
	rec = perform(fixture.router, http.MethodPost, "/hx/demo/step", form, map[string]string{"csrf_token": csrf})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("изменённый токен шага: %d", rec.Code)
	}
}

func TestHandlerDemoStepMissingCSRF(t *testing.T) {
	fixture, tasks := demoFixture(t)
	rec := perform(fixture.router, http.MethodGet, "/diagnostics/demo?exam=oge_math", nil, nil)
	token := demoToken(rec)

	rec = perform(fixture.router, http.MethodPost, "/hx/demo/step", url.Values{
		"demo_token":                     {token},
		"step":                           {"2"},
		"answer_" + tasks[0].ID.String(): {"2"},
	}, nil)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("без CSRF: %d", rec.Code)
	}
}

func TestHandlerFragmentsAreNotCached(t *testing.T) {
	fixture, task := landingFixture(t)
	provider := fixture.handler.service.content.(*fakeContent)

	rec := perform(fixture.router, http.MethodGet, "/tracks/oge_math", nil, nil)
	csrf := csrfFrom(rec)
	form := url.Values{"csrf_token": {csrf}, "answer": {"2"}}
	before := provider.taskCalls
	perform(fixture.router, http.MethodPost, "/hx/tasks/"+task.ID.String()+"/check", form, map[string]string{"csrf_token": csrf})
	perform(fixture.router, http.MethodPost, "/hx/tasks/"+task.ID.String()+"/check", form, map[string]string{"csrf_token": csrf})
	if provider.taskCalls-before != 2 {
		t.Fatalf("фрагмент проверки взят из кэша: вызовов %d", provider.taskCalls-before)
	}
}

func TestHandlerCheckFragmentRateLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	exam := publishedExam()
	year := content.ExamYear{ID: uuid.Must(uuid.NewV7()), ExamID: exam.ID, Year: 2026, Status: content.StatusPublished}
	kim := content.KimNumber{ID: uuid.Must(uuid.NewV7()), ExamYearID: year.ID, Number: 1}
	task := publishedTask()
	task.KimNumberID = kim.ID
	service := NewService(&fakeRepo{}, nil, testTx{}, &testClock{now: time.Now()}, &fakeContent{
		exam: &exam, year: &year, tasks: []content.Task{task}, task: &task,
		kimByID: map[uuid.UUID]content.KimNumber{kim.ID: kim},
	}, fakeRecipients{}, &fakeNotifier{}, testLogger())
	csrf := httpserver.NewCSRF("secret", false, nil)
	handler := NewHandler(service, csrf, NewDemoSigner("secret", DemoTTL, &testClock{now: time.Now()}), NewCache(time.Minute, 10), testLogger(), false, "https://example.ru", "Верно")
	router := gin.New()
	router.Use(csrf.Middleware())
	handler.Register(router, httpserver.NewRateLimiter(1, time.Minute))

	rec := perform(router, http.MethodGet, "/", nil, nil)
	csrfValue := csrfFrom(rec)
	form := url.Values{"csrf_token": {csrfValue}, "answer": {"2"}}
	perform(router, http.MethodPost, "/hx/tasks/"+task.ID.String()+"/check", form, map[string]string{"csrf_token": csrfValue})
	rec = perform(router, http.MethodPost, "/hx/tasks/"+task.ID.String()+"/check", form, map[string]string{"csrf_token": csrfValue})
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("rate limit фрагмента: %d", rec.Code)
	}
}
