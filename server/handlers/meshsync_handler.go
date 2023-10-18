package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/layer5io/meshery/server/models"
	"github.com/layer5io/meshsync/pkg/model"
	"gorm.io/gorm/clause"
)

// swagger:route GET /api/system/meshsync/resources Get GetMeshSyncResources idGetMeshSyncResources
// Handle GET request for meshsync discovered resources
//
// ```?apiVersion={apiVersion}``` If apiVersion is unspecified then all models are returned
//
// ```?search={componentname}``` If search is non empty then a greedy search is performed
//
// ```?order={field}``` orders on the passed field
//
// ```?sort={[asc/desc]}``` Default behavior is asc
//
// ```?page={page-number}``` Default page number is 1
//
// ```?pagesize={pagesize}``` Default pagesize is 25. To return all results: ```pagesize=all```
//
// ```?annotation={annotaion}``` annotation is a boolean value. If true then annotations are returned
//
// ```?lables={lables}``` lables is a boolean value. If true then lables are returned
//
// responses:
// 200: []meshsyncResourcesResponseWrapper

func (h *Handler) GetMeshSyncResources(rw http.ResponseWriter, r *http.Request, _ *models.Preference, _ *models.User, provider models.Provider) {
  rw.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(rw)

	var resources []model.KubernetesResource
	var totalCount int64
	limitstr := r.URL.Query().Get("pagesize")
	var limit int
	if limitstr != "all" {
		limit, _ = strconv.Atoi(limitstr)
		if limit <= 0 {
			limit = defaultPageSize
		}
	}

	pagestr := r.URL.Query().Get("page")
	page, _ := strconv.Atoi(pagestr)

	if page <= 0 {
		page = 1
	}

	offset := (page - 1) * limit
	order := r.URL.Query().Get("order")
	sort := r.URL.Query().Get("sort")
	search := r.URL.Query().Get("search")
	apiVersion := r.URL.Query().Get("apiVersion")
	isAnnotaion, _ := strconv.ParseBool(r.URL.Query().Get("annotation"))
	isLables, _ := strconv.ParseBool(r.URL.Query().Get("lables"))

	result := provider.GetGenericPersister().Model(&model.KubernetesResource{}).
	Preload("KubernetesResourceMeta").
	Preload("Spec").
	Preload("Status")

	if apiVersion != "" {
		result = result.Where(&model.KubernetesResource{APIVersion: apiVersion})
	}

	if isLables {
	 result = result.Preload("KubernetesResourceMeta.Labels")
	}
	if isAnnotaion {
		result = result.Preload("KubernetesResourceMeta.Annotations")
	}
		
	if search != "" {
		result = result.Where(&model.KubernetesResourceObjectMeta{Name: `%`+search+`%`})
	}

	result.Count(&totalCount)

	if limit != 0 {
		result = result.Limit(limit)
	}

	if offset != 0 {
		result = result.Offset(offset)
	}

	if order != "" {
		if sort == "desc" {
			result = result.Order(clause.OrderByColumn{Column: clause.Column{Name: order}, Desc: true})
		} else {
			result = result.Order(order)
		}
	}

	err := result.Find(&resources).Error
	if err != nil {
		h.log.Error(ErrFetchMeshSyncResources(err))
		http.Error(rw, ErrFetchMeshSyncResources(err).Error(), http.StatusInternalServerError)
		return
	}

	var pgSize int
	if limitstr == "all" {
		pgSize = len(resources)
	} else {
		pgSize = limit
	}

	response := &models.MeshSyncResourcesAPIResponse{
		Page:     page,
		PageSize: pgSize,
		TotalCount: totalCount,
		Resources:   resources,
	}

	if err := enc.Encode(response); err != nil {
	 h.log.Error(ErrFetchMeshSyncResources(err))
		http.Error(rw, ErrFetchMeshSyncResources(err).Error(), http.StatusInternalServerError)
	}
}