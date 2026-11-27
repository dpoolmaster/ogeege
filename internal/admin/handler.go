package admin

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"oge-ege/internal/auth"
	"oge-ege/internal/content"
	"oge-ege/internal/shared/httpserver"
	"oge-ege/internal/shared/storage"
)

// Handler обслуживает маршруты /api/admin.
type Handler struct {
	content ContentAdmin
	service *Service
	store   storage.Store
	logger  *slog.Logger
}

// NewHandler создаёт административный HTTP-обработчик.
func NewHandler(contentAdmin ContentAdmin, service *Service, store storage.Store, logger *slog.Logger) *Handler {
	return &Handler{content: contentAdmin, service: service, store: store, logger: logger}
}

// Register регистрирует маршруты управления контентом.
func (h *Handler) Register(group *gin.RouterGroup) {
	group.GET("/subjects", h.listSubjects)
	group.GET("/exams", h.listExams)
	group.POST("/exams", h.createExam)
	group.GET("/exams/:id", h.getExam)
	group.PATCH("/exams/:id", h.updateExam)
	group.POST("/exams/:id/publish", h.publishExam)
	group.DELETE("/exams/:id", h.deleteExam)

	group.GET("/exam-years", h.listExamYears)
	group.POST("/exam-years", h.createExamYear)
	group.GET("/exam-years/:id", h.getExamYear)
	group.PATCH("/exam-years/:id", h.updateExamYear)
	group.POST("/exam-years/:id/publish", h.publishExamYear)
	group.DELETE("/exam-years/:id", h.deleteExamYear)

	group.GET("/kim-numbers", h.listKimNumbers)
	group.POST("/kim-numbers", h.createKimNumber)
	group.GET("/kim-numbers/:id", h.getKimNumber)
	group.PATCH("/kim-numbers/:id", h.updateKimNumber)
	group.DELETE("/kim-numbers/:id", h.deleteKimNumber)
	group.GET("/kim-numbers/:id/topics", h.listKimTopics)
	group.PUT("/kim-numbers/:id/topics", h.setKimTopics)

	group.GET("/topics", h.listTopics)
	group.POST("/topics", h.createTopic)
	group.GET("/topics/:id", h.getTopic)
	group.PATCH("/topics/:id", h.updateTopic)
	group.DELETE("/topics/:id", h.deleteTopic)

	group.GET("/theory", h.listTheory)
	group.GET("/theory/active", h.getActiveTheory)
	group.POST("/theory", h.createTheory)
	group.PATCH("/theory/:id", h.updateTheory)
	group.POST("/theory/:id/publish", h.publishTheory)

	group.GET("/tasks", h.listTasks)
	group.POST("/tasks", h.createTask)
	group.POST("/tasks/import", h.importTasks)
	group.GET("/tasks/:id", h.getTask)
	group.PATCH("/tasks/:id", h.updateTask)
	group.DELETE("/tasks/:id", h.deleteTask)
	group.POST("/tasks/:id/publish", h.publishTask)
	group.POST("/tasks/:id/archive", h.archiveTask)
	group.GET("/tasks/:id/versions", h.listTaskVersions)
	group.GET("/tasks/:id/attachments", h.listTaskAttachments)
	group.POST("/tasks/:id/attachments", h.uploadTaskAttachment)

	group.GET("/variants", h.listVariants)
	group.POST("/variants", h.createVariant)
	group.POST("/variants/generate", h.generateVariant)
	group.GET("/variants/:id", h.getVariant)
	group.POST("/variants/:id/publish", h.publishVariant)

	group.GET("/rubrics", h.listRubrics)
	group.POST("/rubrics", h.createRubric)
	group.GET("/rubrics/:id", h.getRubric)
	group.PUT("/rubrics/:id", h.updateRubric)
	group.POST("/rubrics/:id/publish", h.publishRubric)

	group.GET("/score-scales", h.listScoreScales)
	group.PUT("/score-scales/:examYearId", h.replaceScoreScales)
}

// RegisterAdminOnly регистрирует маршруты, доступные только администратору.
func (h *Handler) RegisterAdminOnly(group *gin.RouterGroup) {
	group.GET("/users", h.listUsers)
	group.PATCH("/users/:id", h.patchUser)
	group.GET("/audit", h.listAudit)
}

func (h *Handler) listSubjects(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"items": content.Subjects()})
}

func (h *Handler) listExams(c *gin.Context) {
	filter := content.ExamFilter{Cursor: parseCursor(c), Limit: parseLimit(c), Query: c.Query("query")}
	if status := parseStatus(c.Query("status")); status != nil {
		filter.Status = status
	}

	items, err := h.content.ListExams(c.Request.Context(), filter)
	if err != nil {
		h.writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": toExamResponses(items), "next_cursor": nextCursor(len(items), filter.Limit, examIDs(items))})
}

func (h *Handler) getExam(c *gin.Context) {
	id, ok := h.pathID(c)
	if !ok {
		return
	}
	exam, err := h.content.GetExam(c.Request.Context(), id)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toExamResponse(exam))
}

func (h *Handler) createExam(c *gin.Context) {
	var req examRequest
	if !h.bind(c, &req) {
		return
	}
	actor, ok := h.actor(c)
	if !ok {
		return
	}

	exam, err := h.service.CreateExam(c.Request.Context(), actor, content.ExamInput(req))
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toExamResponse(exam))
}

func (h *Handler) updateExam(c *gin.Context) {
	id, ok := h.pathID(c)
	if !ok {
		return
	}
	var req examRequest
	if !h.bind(c, &req) {
		return
	}
	actor, ok := h.actor(c)
	if !ok {
		return
	}

	exam, err := h.service.UpdateExam(c.Request.Context(), actor, id, content.ExamInput(req))
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toExamResponse(exam))
}

func (h *Handler) publishExam(c *gin.Context) {
	id, ok := h.pathID(c)
	if !ok {
		return
	}
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	exam, err := h.service.PublishExam(c.Request.Context(), actor, id)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toExamResponse(exam))
}

func (h *Handler) deleteExam(c *gin.Context) {
	id, ok := h.pathID(c)
	if !ok {
		return
	}
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	if err := h.service.DeleteExam(c.Request.Context(), actor, id); err != nil {
		h.writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) listExamYears(c *gin.Context) {
	filter := content.ExamYearFilter{Cursor: parseCursor(c), Limit: parseLimit(c)}
	if examID, ok := optionalUUID(c.Query("exam_id")); ok {
		filter.ExamID = examID
	}
	if status := parseStatus(c.Query("status")); status != nil {
		filter.Status = status
	}

	items, err := h.content.ListExamYears(c.Request.Context(), filter)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": toExamYearResponses(items), "next_cursor": nextCursor(len(items), filter.Limit, examYearIDs(items))})
}

func (h *Handler) getExamYear(c *gin.Context) {
	id, ok := h.pathID(c)
	if !ok {
		return
	}
	year, err := h.content.GetExamYear(c.Request.Context(), id)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toExamYearResponse(year))
}

func (h *Handler) createExamYear(c *gin.Context) {
	var req examYearRequest
	if !h.bind(c, &req) {
		return
	}
	examID, ok := h.parseUUIDField(c, "exam_id", req.ExamID)
	if !ok {
		return
	}
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	year, err := h.service.CreateExamYear(c.Request.Context(), actor, content.ExamYearInput{
		ExamID: examID, Year: req.Year, DurationMinutes: req.DurationMinutes,
	})
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toExamYearResponse(year))
}

func (h *Handler) updateExamYear(c *gin.Context) {
	id, ok := h.pathID(c)
	if !ok {
		return
	}
	var req examYearRequest
	if !h.bind(c, &req) {
		return
	}
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	year, err := h.service.UpdateExamYear(c.Request.Context(), actor, id, content.ExamYearInput{
		Year: req.Year, DurationMinutes: req.DurationMinutes,
	})
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toExamYearResponse(year))
}

func (h *Handler) publishExamYear(c *gin.Context) {
	id, ok := h.pathID(c)
	if !ok {
		return
	}
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	year, err := h.service.PublishExamYear(c.Request.Context(), actor, id)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toExamYearResponse(year))
}

func (h *Handler) deleteExamYear(c *gin.Context) {
	id, ok := h.pathID(c)
	if !ok {
		return
	}
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	if err := h.service.DeleteExamYear(c.Request.Context(), actor, id); err != nil {
		h.writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) listKimNumbers(c *gin.Context) {
	filter := content.KimNumberFilter{Cursor: parseCursor(c), Limit: parseLimit(c)}
	if examYearID, ok := optionalUUID(c.Query("exam_year_id")); ok {
		filter.ExamYearID = examYearID
	}

	items, err := h.content.ListKimNumbers(c.Request.Context(), filter)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": toKimNumberResponses(items), "next_cursor": nextCursor(len(items), filter.Limit, kimNumberIDs(items))})
}

func (h *Handler) getKimNumber(c *gin.Context) {
	id, ok := h.pathID(c)
	if !ok {
		return
	}
	number, err := h.content.GetKimNumber(c.Request.Context(), id)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toKimNumberResponse(number))
}

func (h *Handler) createKimNumber(c *gin.Context) {
	var req kimNumberRequest
	if !h.bind(c, &req) {
		return
	}
	input, ok := h.kimNumberInput(c, req)
	if !ok {
		return
	}
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	number, err := h.service.CreateKimNumber(c.Request.Context(), actor, input)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toKimNumberResponse(number))
}

func (h *Handler) updateKimNumber(c *gin.Context) {
	id, ok := h.pathID(c)
	if !ok {
		return
	}
	var req kimNumberRequest
	if !h.bind(c, &req) {
		return
	}
	input, ok := h.kimNumberInput(c, req)
	if !ok {
		return
	}
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	number, err := h.service.UpdateKimNumber(c.Request.Context(), actor, id, input)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toKimNumberResponse(number))
}

func (h *Handler) deleteKimNumber(c *gin.Context) {
	id, ok := h.pathID(c)
	if !ok {
		return
	}
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	if err := h.service.DeleteKimNumber(c.Request.Context(), actor, id); err != nil {
		h.writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) listKimTopics(c *gin.Context) {
	id, ok := h.pathID(c)
	if !ok {
		return
	}
	topics, err := h.content.ListKimTopics(c.Request.Context(), id)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": toKimTopicResponses(topics)})
}

func (h *Handler) setKimTopics(c *gin.Context) {
	id, ok := h.pathID(c)
	if !ok {
		return
	}
	var req kimTopicsRequest
	if !h.bind(c, &req) {
		return
	}
	topics := make([]content.KimTopicInput, 0, len(req.Topics))
	for _, item := range req.Topics {
		topicID, err := uuid.Parse(strings.TrimSpace(item.TopicID))
		if err != nil {
			httpserver.AbortWithFields(c, http.StatusUnprocessableEntity, "validation_failed", "Проверьте заполнение полей", map[string]string{"topic_id": "некорректный идентификатор"})
			return
		}
		topics = append(topics, content.KimTopicInput{TopicID: topicID, Weight: item.Weight})
	}
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	links, err := h.service.SetKimTopics(c.Request.Context(), actor, id, topics)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": toKimTopicResponses(links)})
}

func (h *Handler) listTopics(c *gin.Context) {
	filter := content.TopicFilter{Cursor: parseCursor(c), Limit: parseLimit(c)}
	if examID, ok := optionalUUID(c.Query("exam_id")); ok {
		filter.ExamID = examID
	}
	if parentID, ok := optionalUUID(c.Query("parent_id")); ok {
		filter.ParentID = parentID
	}
	filter.Code = c.Query("code")

	items, err := h.content.ListTopics(c.Request.Context(), filter)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": toTopicResponses(items), "next_cursor": nextCursor(len(items), filter.Limit, topicIDs(items))})
}

func (h *Handler) getTopic(c *gin.Context) {
	id, ok := h.pathID(c)
	if !ok {
		return
	}
	topic, err := h.content.GetTopic(c.Request.Context(), id)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toTopicResponse(topic))
}

func (h *Handler) createTopic(c *gin.Context) {
	var req topicRequest
	if !h.bind(c, &req) {
		return
	}
	input, ok := h.topicInput(c, req)
	if !ok {
		return
	}
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	topic, err := h.service.CreateTopic(c.Request.Context(), actor, input)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toTopicResponse(topic))
}

func (h *Handler) updateTopic(c *gin.Context) {
	id, ok := h.pathID(c)
	if !ok {
		return
	}
	var req topicRequest
	if !h.bind(c, &req) {
		return
	}
	input, ok := h.topicInput(c, req)
	if !ok {
		return
	}
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	topic, err := h.service.UpdateTopic(c.Request.Context(), actor, id, input)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toTopicResponse(topic))
}

func (h *Handler) deleteTopic(c *gin.Context) {
	id, ok := h.pathID(c)
	if !ok {
		return
	}
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	if err := h.service.DeleteTopic(c.Request.Context(), actor, id); err != nil {
		h.writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) listTheory(c *gin.Context) {
	topicID, ok := h.parseUUIDField(c, "topic_id", c.Query("topic_id"))
	if !ok {
		return
	}
	versions, err := h.content.ListTheoryVersions(c.Request.Context(), topicID)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": toTheoryResponses(versions)})
}

func (h *Handler) getActiveTheory(c *gin.Context) {
	topicID, ok := h.parseUUIDField(c, "topic_id", c.Query("topic_id"))
	if !ok {
		return
	}
	theory, err := h.content.GetActiveTheory(c.Request.Context(), topicID)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toTheoryResponse(theory))
}

func (h *Handler) createTheory(c *gin.Context) {
	var req theoryRequest
	if !h.bind(c, &req) {
		return
	}
	topicID, ok := h.parseUUIDField(c, "topic_id", req.TopicID)
	if !ok {
		return
	}
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	theory, err := h.service.CreateTheoryVersion(c.Request.Context(), actor, content.TheoryInput{
		TopicID: topicID, ContentMD: req.ContentMD, UpdatedBy: &actor.ID,
	})
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toTheoryResponse(theory))
}

func (h *Handler) updateTheory(c *gin.Context) {
	id, ok := h.pathID(c)
	if !ok {
		return
	}
	var req theoryRequest
	if !h.bind(c, &req) {
		return
	}
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	theory, err := h.service.UpdateTheory(c.Request.Context(), actor, id, content.TheoryInput{
		ContentMD: req.ContentMD, UpdatedBy: &actor.ID,
	})
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toTheoryResponse(theory))
}

func (h *Handler) publishTheory(c *gin.Context) {
	id, ok := h.pathID(c)
	if !ok {
		return
	}
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	theory, err := h.service.PublishTheory(c.Request.Context(), actor, id)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toTheoryResponse(theory))
}

func (h *Handler) listTasks(c *gin.Context) {
	filter := content.TaskFilter{Cursor: parseCursor(c), Limit: parseLimit(c), Query: c.Query("query")}
	if examYearID, ok := optionalUUID(c.Query("exam_year_id")); ok {
		filter.ExamYearID = examYearID
	}
	if kimNumberID, ok := optionalUUID(c.Query("kim_number_id")); ok {
		filter.KimNumberID = kimNumberID
	}
	if topicID, ok := optionalUUID(c.Query("topic_id")); ok {
		filter.TopicID = topicID
	}
	if status := parseStatus(c.Query("status")); status != nil {
		filter.Status = status
	}

	items, err := h.content.ListTasks(c.Request.Context(), filter)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": toTaskResponses(items), "next_cursor": nextCursor(len(items), filter.Limit, taskIDs(items))})
}

func (h *Handler) getTask(c *gin.Context) {
	id, ok := h.pathID(c)
	if !ok {
		return
	}
	task, err := h.content.GetTask(c.Request.Context(), id)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toTaskResponse(task))
}

func (h *Handler) createTask(c *gin.Context) {
	var req taskRequest
	if !h.bind(c, &req) {
		return
	}
	input, ok := h.taskInput(c, req)
	if !ok {
		return
	}
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	input.CreatedBy = &actor.ID
	task, err := h.service.CreateTask(c.Request.Context(), actor, input)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toTaskResponse(task))
}

func (h *Handler) updateTask(c *gin.Context) {
	id, ok := h.pathID(c)
	if !ok {
		return
	}
	var req taskRequest
	if !h.bind(c, &req) {
		return
	}
	input, ok := h.taskInput(c, req)
	if !ok {
		return
	}
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	task, err := h.service.UpdateTask(c.Request.Context(), actor, id, input)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toTaskResponse(task))
}

func (h *Handler) deleteTask(c *gin.Context) {
	id, ok := h.pathID(c)
	if !ok {
		return
	}
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	if err := h.service.DeleteTask(c.Request.Context(), actor, id); err != nil {
		h.writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) publishTask(c *gin.Context) {
	id, ok := h.pathID(c)
	if !ok {
		return
	}
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	task, err := h.service.PublishTask(c.Request.Context(), actor, id)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toTaskResponse(task))
}

func (h *Handler) archiveTask(c *gin.Context) {
	id, ok := h.pathID(c)
	if !ok {
		return
	}
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	task, err := h.service.ArchiveTask(c.Request.Context(), actor, id)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toTaskResponse(task))
}

func (h *Handler) listTaskVersions(c *gin.Context) {
	id, ok := h.pathID(c)
	if !ok {
		return
	}
	versions, err := h.content.ListTaskVersions(c.Request.Context(), id)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": toTaskVersionResponses(versions)})
}

func (h *Handler) listTaskAttachments(c *gin.Context) {
	id, ok := h.pathID(c)
	if !ok {
		return
	}
	attachments, err := h.content.ListTaskAttachments(c.Request.Context(), id)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": toTaskAttachmentResponses(attachments)})
}

func (h *Handler) importTasks(c *gin.Context) {
	var req importRequest
	if !h.bind(c, &req) {
		return
	}
	items := make([]content.ImportItem, 0, len(req.Items))
	for _, item := range req.Items {
		examYearID, err := uuid.Parse(strings.TrimSpace(item.ExamYearID))
		if err != nil {
			httpserver.AbortWithFields(c, http.StatusUnprocessableEntity, "validation_failed", "Проверьте заполнение полей", map[string]string{"exam_year_id": "некорректный идентификатор"})
			return
		}
		kimNumberID, err := uuid.Parse(strings.TrimSpace(item.KimNumberID))
		if err != nil {
			httpserver.AbortWithFields(c, http.StatusUnprocessableEntity, "validation_failed", "Проверьте заполнение полей", map[string]string{"kim_number_id": "некорректный идентификатор"})
			return
		}
		var topicID *uuid.UUID
		if item.TopicID != nil && strings.TrimSpace(*item.TopicID) != "" {
			parsed, err := uuid.Parse(strings.TrimSpace(*item.TopicID))
			if err != nil {
				httpserver.AbortWithFields(c, http.StatusUnprocessableEntity, "validation_failed", "Проверьте заполнение полей", map[string]string{"topic_id": "некорректный идентификатор"})
				return
			}
			topicID = &parsed
		}
		items = append(items, content.ImportItem{
			ExamYearID: examYearID, KimNumberID: kimNumberID, TopicID: topicID,
			Difficulty: content.Difficulty(item.Difficulty), StatementMD: item.StatementMD,
			AnswerSpec: item.AnswerSpec, AnswerDisplay: item.AnswerDisplay,
			Source: item.Source, SourceRef: item.SourceRef,
		})
	}
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	for i := range items {
		items[i].CreatedBy = &actor.ID
	}
	report, err := h.service.ImportTasks(c.Request.Context(), actor, items)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toImportReportResponse(report))
}

func (h *Handler) uploadTaskAttachment(c *gin.Context) {
	taskID, ok := h.pathID(c)
	if !ok {
		return
	}
	header, err := c.FormFile("file")
	if err != nil {
		httpserver.AbortWithFields(c, http.StatusUnprocessableEntity, "validation_failed", "Файл обязателен", map[string]string{"file": "выберите файл"})
		return
	}
	file, err := header.Open()
	if err != nil {
		h.logger.Error("не удалось открыть загруженный файл", "error", err)
		httpserver.AbortWithError(c, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка")
		return
	}
	defer func() {
		_ = file.Close()
	}()

	headerBytes := make([]byte, 512)
	n, _ := file.Read(headerBytes)
	contentType, err := storage.DetectImageType(headerBytes[:n])
	if err != nil {
		httpserver.AbortWithFields(c, http.StatusUnprocessableEntity, "validation_failed", err.Error(), map[string]string{"file": err.Error()})
		return
	}
	if err := storage.ValidateImage(contentType, header.Size); err != nil {
		httpserver.AbortWithFields(c, http.StatusUnprocessableEntity, "validation_failed", err.Error(), map[string]string{"file": err.Error()})
		return
	}
	if _, err := file.Seek(0, 0); err != nil {
		h.logger.Error("не удалось вернуть файл в начало", "error", err)
		httpserver.AbortWithError(c, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка")
		return
	}

	key, err := storage.ImageKey(taskID, contentType)
	if err != nil {
		httpserver.AbortWithFields(c, http.StatusUnprocessableEntity, "validation_failed", err.Error(), map[string]string{"file": err.Error()})
		return
	}
	if err := h.store.Put(c.Request.Context(), key, contentType, file, header.Size); err != nil {
		h.logger.Error("не удалось сохранить изображение", "error", err)
		httpserver.AbortWithError(c, http.StatusBadGateway, "upstream_error", "Хранилище недоступно")
		return
	}

	actor, ok := h.actor(c)
	if !ok {
		return
	}
	attachment, err := h.service.AddTaskAttachment(c.Request.Context(), actor, taskID, content.TaskAttachmentInput{
		Kind: "image", StorageKey: key, Alt: c.PostForm("alt"),
	})
	if err != nil {
		_ = h.store.Delete(c.Request.Context(), key)
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toTaskAttachmentResponse(attachment))
}

func (h *Handler) listVariants(c *gin.Context) {
	filter := content.VariantFilter{Cursor: parseCursor(c), Limit: parseLimit(c)}
	if examYearID, ok := optionalUUID(c.Query("exam_year_id")); ok {
		filter.ExamYearID = examYearID
	}
	if status := parseStatus(c.Query("status")); status != nil {
		filter.Status = status
	}

	items, err := h.content.ListVariants(c.Request.Context(), filter)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": toVariantResponses(items), "next_cursor": nextCursor(len(items), filter.Limit, variantIDs(items))})
}

func (h *Handler) getVariant(c *gin.Context) {
	id, ok := h.pathID(c)
	if !ok {
		return
	}
	variant, err := h.content.GetVariant(c.Request.Context(), id)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toVariantResponse(variant))
}

func (h *Handler) createVariant(c *gin.Context) {
	var req variantRequest
	if !h.bind(c, &req) {
		return
	}
	input, ok := h.variantInput(c, req)
	if !ok {
		return
	}
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	variant, err := h.service.CreateVariant(c.Request.Context(), actor, input)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toVariantResponse(variant))
}

func (h *Handler) generateVariant(c *gin.Context) {
	var req generateVariantRequest
	if !h.bind(c, &req) {
		return
	}
	examYearID, ok := h.parseUUIDField(c, "exam_year_id", req.ExamYearID)
	if !ok {
		return
	}
	rules := make([]content.GenerationRule, 0, len(req.Rules))
	for _, rule := range req.Rules {
		kimNumberID, err := uuid.Parse(strings.TrimSpace(rule.KimNumberID))
		if err != nil {
			httpserver.AbortWithFields(c, http.StatusUnprocessableEntity, "validation_failed", "Проверьте заполнение полей", map[string]string{"kim_number_id": "некорректный идентификатор"})
			return
		}
		var difficulty *content.Difficulty
		if rule.Difficulty != nil {
			value := content.Difficulty(*rule.Difficulty)
			difficulty = &value
		}
		var topicID *uuid.UUID
		if rule.TopicID != nil && strings.TrimSpace(*rule.TopicID) != "" {
			parsed, err := uuid.Parse(strings.TrimSpace(*rule.TopicID))
			if err != nil {
				httpserver.AbortWithFields(c, http.StatusUnprocessableEntity, "validation_failed", "Проверьте заполнение полей", map[string]string{"topic_id": "некорректный идентификатор"})
				return
			}
			topicID = &parsed
		}
		rules = append(rules, content.GenerationRule{
			KimNumberID: kimNumberID, Count: rule.Count, Difficulty: difficulty, TopicID: topicID,
		})
	}
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	variant, err := h.service.GenerateVariant(c.Request.Context(), actor, content.GenerateVariantInput{
		ExamYearID: examYearID, Title: req.Title, Kind: content.VariantKind(req.Kind),
		DurationMinutes: req.DurationMinutes, Rules: rules,
	})
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toVariantResponse(variant))
}

func (h *Handler) publishVariant(c *gin.Context) {
	id, ok := h.pathID(c)
	if !ok {
		return
	}
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	variant, err := h.service.PublishVariant(c.Request.Context(), actor, id)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toVariantResponse(variant))
}

func (h *Handler) listRubrics(c *gin.Context) {
	filter := content.RubricFilter{Cursor: parseCursor(c), Limit: parseLimit(c)}
	if kimNumberID, ok := optionalUUID(c.Query("kim_number_id")); ok {
		filter.KimNumberID = kimNumberID
	}
	if examYearID, ok := optionalUUID(c.Query("exam_year_id")); ok {
		filter.ExamYearID = examYearID
	}
	if status := parseStatus(c.Query("status")); status != nil {
		filter.Status = status
	}

	items, err := h.content.ListRubrics(c.Request.Context(), filter)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": toRubricResponses(items), "next_cursor": nextCursor(len(items), filter.Limit, rubricIDs(items))})
}

func (h *Handler) getRubric(c *gin.Context) {
	id, ok := h.pathID(c)
	if !ok {
		return
	}
	rubric, err := h.content.GetRubric(c.Request.Context(), id)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toRubricResponse(rubric))
}

func (h *Handler) createRubric(c *gin.Context) {
	var req rubricRequest
	if !h.bind(c, &req) {
		return
	}
	kimNumberID, ok := h.parseUUIDField(c, "kim_number_id", req.KimNumberID)
	if !ok {
		return
	}
	examYearID, ok := h.parseUUIDField(c, "exam_year_id", req.ExamYearID)
	if !ok {
		return
	}
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	rubric, err := h.service.CreateRubric(c.Request.Context(), actor, content.RubricInput{
		KimNumberID: kimNumberID, ExamYearID: examYearID, Criteria: req.Criteria,
	})
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toRubricResponse(rubric))
}

func (h *Handler) updateRubric(c *gin.Context) {
	id, ok := h.pathID(c)
	if !ok {
		return
	}
	var req rubricUpdateRequest
	if !h.bind(c, &req) {
		return
	}
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	rubric, err := h.service.UpdateRubric(c.Request.Context(), actor, id, req.Criteria)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toRubricResponse(rubric))
}

func (h *Handler) publishRubric(c *gin.Context) {
	id, ok := h.pathID(c)
	if !ok {
		return
	}
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	rubric, err := h.service.PublishRubric(c.Request.Context(), actor, id)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toRubricResponse(rubric))
}

func (h *Handler) listScoreScales(c *gin.Context) {
	examYearID, ok := h.parseUUIDField(c, "exam_year_id", c.Query("exam_year_id"))
	if !ok {
		return
	}
	scales, err := h.content.ListScoreScales(c.Request.Context(), examYearID)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": toScoreScaleResponses(scales)})
}

func (h *Handler) replaceScoreScales(c *gin.Context) {
	examYearID, ok := h.pathID(c)
	if !ok {
		return
	}
	var req scoreScaleRequest
	if !h.bind(c, &req) {
		return
	}
	entries := make([]content.ScoreScaleEntry, 0, len(req.Entries))
	for _, entry := range req.Entries {
		entries = append(entries, content.ScoreScaleEntry{
			PrimaryScore: entry.PrimaryScore, TestScore: entry.TestScore,
			Grade: entry.Grade, MinThreshold: entry.MinThreshold,
		})
	}
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	scales, err := h.service.ReplaceScoreScales(c.Request.Context(), actor, content.ScoreScaleInput{
		ExamYearID: examYearID, Entries: entries,
	})
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": toScoreScaleResponses(scales)})
}

func (h *Handler) listUsers(c *gin.Context) {
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	limit := parseLimit(c)
	users, err := h.service.SearchUsers(c.Request.Context(), actor, c.Query("query"), parseCursor(c), limit)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": toUserResponses(users), "next_cursor": nextCursor(len(users), limit, userSummaryIDs(users))})
}

func (h *Handler) patchUser(c *gin.Context) {
	id, ok := h.pathID(c)
	if !ok {
		return
	}
	var req userPatchRequest
	if !h.bind(c, &req) {
		return
	}
	if req.Role == nil && req.Status == nil {
		httpserver.AbortWithFields(c, http.StatusUnprocessableEntity, "validation_failed", "Укажите роль или статус", map[string]string{"role": "изменение не задано"})
		return
	}
	actor, ok := h.actor(c)
	if !ok {
		return
	}

	if req.Role != nil {
		role := auth.Role(strings.TrimSpace(*req.Role))
		if !role.Valid() {
			httpserver.AbortWithFields(c, http.StatusUnprocessableEntity, "validation_failed", "Некорректная роль", map[string]string{"role": "неизвестная роль"})
			return
		}
		summary, err := h.service.SetUserRole(c.Request.Context(), actor, id, role)
		if err != nil {
			h.writeError(c, err)
			return
		}
		c.JSON(http.StatusOK, toUserResponse(*summary))
		return
	}

	status := auth.Status(strings.TrimSpace(*req.Status))
	if !status.Valid() {
		httpserver.AbortWithFields(c, http.StatusUnprocessableEntity, "validation_failed", "Некорректный статус", map[string]string{"status": "неизвестный статус"})
		return
	}
	summary, err := h.service.SetUserStatus(c.Request.Context(), actor, id, status)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toUserResponse(*summary))
}

func (h *Handler) listAudit(c *gin.Context) {
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	filter := AuditFilter{
		Cursor: parseCursor(c), Limit: parseLimit(c),
		Action: c.Query("action"), Entity: c.Query("entity"),
	}
	if actorID, ok := optionalUUID(c.Query("actor_id")); ok {
		filter.ActorID = actorID
	}
	if from, ok := parseTime(c.Query("from")); ok {
		filter.From = from
	}
	if to, ok := parseTime(c.Query("to")); ok {
		filter.To = to
	}

	entries, err := h.service.ListAudit(c.Request.Context(), actor, filter)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": toAuditResponses(entries), "next_cursor": nextCursor(len(entries), filter.Limit, auditIDs(entries))})
}

func (h *Handler) actor(c *gin.Context) (Actor, bool) {
	principal, ok := auth.PrincipalFrom(c)
	if !ok {
		httpserver.AbortWithError(c, http.StatusUnauthorized, "unauthorized", "Требуется вход в систему")
		return Actor{}, false
	}

	return Actor{ID: principal.ID, Role: principal.Role, IP: c.ClientIP()}, true
}

func (h *Handler) bind(c *gin.Context, target any) bool {
	if err := c.ShouldBind(target); err != nil {
		httpserver.AbortWithError(c, http.StatusUnprocessableEntity, "validation_failed", "Проверьте заполнение полей")
		return false
	}

	return true
}

func (h *Handler) pathID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(strings.TrimSpace(c.Param("id")))
	if err != nil {
		httpserver.AbortWithError(c, http.StatusNotFound, "not_found", "Объект не найден")
		return uuid.Nil, false
	}

	return id, true
}

func (h *Handler) parseUUIDField(c *gin.Context, field, value string) (uuid.UUID, bool) {
	id, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil {
		httpserver.AbortWithFields(c, http.StatusUnprocessableEntity, "validation_failed", "Проверьте заполнение полей", map[string]string{field: "некорректный идентификатор"})
		return uuid.Nil, false
	}

	return id, true
}

func (h *Handler) kimNumberInput(c *gin.Context, req kimNumberRequest) (content.KimNumberInput, bool) {
	examYearID, ok := h.parseUUIDField(c, "exam_year_id", req.ExamYearID)
	if !ok {
		return content.KimNumberInput{}, false
	}

	return content.KimNumberInput{
		ExamYearID: examYearID, Number: req.Number, Part: req.Part,
		AnswerType: content.AnswerType(req.AnswerType), MaxPrimaryScore: req.MaxPrimaryScore, Title: req.Title,
	}, true
}

func (h *Handler) topicInput(c *gin.Context, req topicRequest) (content.TopicInput, bool) {
	examID, ok := h.parseUUIDField(c, "exam_id", req.ExamID)
	if !ok {
		return content.TopicInput{}, false
	}
	var parentID *uuid.UUID
	if req.ParentID != nil && strings.TrimSpace(*req.ParentID) != "" {
		parsed, err := uuid.Parse(strings.TrimSpace(*req.ParentID))
		if err != nil {
			httpserver.AbortWithFields(c, http.StatusUnprocessableEntity, "validation_failed", "Проверьте заполнение полей", map[string]string{"parent_id": "некорректный идентификатор"})
			return content.TopicInput{}, false
		}
		parentID = &parsed
	}

	return content.TopicInput{
		ExamID: examID, ParentID: parentID, Code: req.Code, Title: req.Title, OrderIndex: req.OrderIndex,
	}, true
}

func (h *Handler) taskInput(c *gin.Context, req taskRequest) (content.TaskInput, bool) {
	examYearID, ok := h.parseUUIDField(c, "exam_year_id", req.ExamYearID)
	if !ok {
		return content.TaskInput{}, false
	}
	kimNumberID, ok := h.parseUUIDField(c, "kim_number_id", req.KimNumberID)
	if !ok {
		return content.TaskInput{}, false
	}
	var topicID *uuid.UUID
	if req.TopicID != nil && strings.TrimSpace(*req.TopicID) != "" {
		parsed, err := uuid.Parse(strings.TrimSpace(*req.TopicID))
		if err != nil {
			httpserver.AbortWithFields(c, http.StatusUnprocessableEntity, "validation_failed", "Проверьте заполнение полей", map[string]string{"topic_id": "некорректный идентификатор"})
			return content.TaskInput{}, false
		}
		topicID = &parsed
	}

	return content.TaskInput{
		ExamYearID: examYearID, KimNumberID: kimNumberID, TopicID: topicID,
		Difficulty: content.Difficulty(req.Difficulty), StatementMD: req.StatementMD,
		AnswerSpec: req.AnswerSpec, AnswerDisplay: req.AnswerDisplay,
		Source: req.Source, SourceRef: req.SourceRef,
	}, true
}

func (h *Handler) variantInput(c *gin.Context, req variantRequest) (content.VariantInput, bool) {
	examYearID, ok := h.parseUUIDField(c, "exam_year_id", req.ExamYearID)
	if !ok {
		return content.VariantInput{}, false
	}
	tasks := make([]content.VariantTaskInput, 0, len(req.Tasks))
	for _, item := range req.Tasks {
		taskID, err := uuid.Parse(strings.TrimSpace(item.TaskID))
		if err != nil {
			httpserver.AbortWithFields(c, http.StatusUnprocessableEntity, "validation_failed", "Проверьте заполнение полей", map[string]string{"task_id": "некорректный идентификатор"})
			return content.VariantInput{}, false
		}
		tasks = append(tasks, content.VariantTaskInput{TaskID: taskID, Points: item.Points})
	}

	return content.VariantInput{
		ExamYearID: examYearID, Title: req.Title, Kind: content.VariantKind(req.Kind),
		DurationMinutes: req.DurationMinutes, Tasks: tasks,
	}, true
}

func (h *Handler) writeError(c *gin.Context, err error) {
	var validationErr *content.ValidationError
	switch {
	case errors.As(err, &validationErr):
		httpserver.AbortWithFields(c, http.StatusUnprocessableEntity, "validation_failed", validationErr.Message, map[string]string{validationErr.Field: validationErr.Message})
	case errors.Is(err, content.ErrConflict):
		httpserver.AbortWithError(c, http.StatusConflict, "conflict", err.Error())
	case errors.Is(err, content.ErrNotFound):
		httpserver.AbortWithError(c, http.StatusNotFound, "not_found", "Объект не найден")
	case errors.Is(err, content.ErrInsufficient):
		httpserver.AbortWithError(c, http.StatusUnprocessableEntity, "validation_failed", err.Error())
	case errors.Is(err, ErrForbidden):
		httpserver.AbortWithError(c, http.StatusForbidden, "forbidden", "Недостаточно прав")
	default:
		h.logger.Error("ошибка административного запроса", "error", err)
		httpserver.AbortWithError(c, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка")
	}
}

func parseCursor(c *gin.Context) uuid.UUID {
	value := strings.TrimSpace(c.Query("cursor"))
	if value == "" {
		return uuid.Nil
	}
	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil
	}

	return id
}

func parseLimit(c *gin.Context) int {
	value := strings.TrimSpace(c.Query("limit"))
	if value == "" {
		return 50
	}
	limit, err := strconv.Atoi(value)
	if err != nil {
		return 50
	}

	return limit
}

func parseStatus(value string) *content.Status {
	status := content.Status(strings.TrimSpace(value))
	if !status.Valid() {
		return nil
	}

	return &status
}

func optionalUUID(value string) (*uuid.UUID, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, false
	}
	id, err := uuid.Parse(value)
	if err != nil {
		return nil, false
	}

	return &id, true
}

func parseTime(value string) (*time.Time, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, false
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, false
	}

	return &parsed, true
}

func nextCursor[T any](count, limit int, ids []T) any {
	if count < limit || count == 0 {
		return nil
	}

	return ids[count-1]
}

func examIDs(items []content.Exam) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}

	return ids
}

func examYearIDs(items []content.ExamYear) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}

	return ids
}

func kimNumberIDs(items []content.KimNumber) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}

	return ids
}

func topicIDs(items []content.Topic) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}

	return ids
}

func taskIDs(items []content.Task) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}

	return ids
}

func variantIDs(items []content.Variant) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}

	return ids
}

func userSummaryIDs(items []UserSummary) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}

	return ids
}

func auditIDs(items []AuditEntry) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}

	return ids
}

func rubricIDs(items []content.Rubric) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}

	return ids
}
