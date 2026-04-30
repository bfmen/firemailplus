package handlers

import (
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

const defaultSoftDeleteCleanupRetentionDays = 30

// GetSoftDeleteStats 获取软删除统计信息
func (h *Handler) GetSoftDeleteStats(c *gin.Context) {
	stats, err := h.softDeleteService.GetSoftDeleteStats(c.Request.Context())
	if err != nil {
		h.respondWithError(c, http.StatusInternalServerError, "Failed to get soft delete stats: "+err.Error())
		return
	}

	h.respondWithSuccess(c, stats, "Soft delete statistics retrieved successfully")
}

// CleanupExpiredSoftDeletes 清理过期的软删除数据
func (h *Handler) CleanupExpiredSoftDeletes(c *gin.Context) {
	var req struct {
		RetentionDays *int `json:"retention_days"`
	}

	retentionDays := defaultSoftDeleteCleanupRetentionDays
	if c.Request != nil && c.Request.Body != nil && c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&req); err != nil && err != io.EOF {
			h.respondWithError(c, http.StatusBadRequest, "Invalid request body: "+err.Error())
			return
		}
		if req.RetentionDays != nil {
			if *req.RetentionDays < 1 {
				h.respondWithError(c, http.StatusBadRequest, "retention_days must be positive")
				return
			}
			retentionDays = *req.RetentionDays
		}
	}

	err := h.softDeleteService.CleanupExpiredSoftDeletes(c.Request.Context(), retentionDays)
	if err != nil {
		h.respondWithError(c, http.StatusInternalServerError, "Failed to cleanup expired soft deletes: "+err.Error())
		return
	}

	h.respondWithSuccess(c, nil, "Expired soft deleted records cleaned up successfully")
}

// RestoreSoftDeleted 恢复软删除的记录
func (h *Handler) RestoreSoftDeleted(c *gin.Context) {
	tableName := c.Param("table")
	idStr := c.Param("id")

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		h.respondWithError(c, http.StatusBadRequest, "Invalid ID parameter")
		return
	}

	err = h.softDeleteService.RestoreSoftDeleted(c.Request.Context(), tableName, uint(id))
	if err != nil {
		h.respondWithError(c, http.StatusInternalServerError, "Failed to restore record: "+err.Error())
		return
	}

	h.respondWithSuccess(c, nil, "Record restored successfully")
}

// PermanentlyDelete 永久删除软删除的记录
func (h *Handler) PermanentlyDelete(c *gin.Context) {
	tableName := c.Param("table")
	idStr := c.Param("id")

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		h.respondWithError(c, http.StatusBadRequest, "Invalid ID parameter")
		return
	}

	err = h.softDeleteService.PermanentlyDelete(c.Request.Context(), tableName, uint(id))
	if err != nil {
		h.respondWithError(c, http.StatusInternalServerError, "Failed to permanently delete record: "+err.Error())
		return
	}

	h.respondWithSuccess(c, nil, "Record permanently deleted successfully")
}
