package orgs

import "github.com/gin-gonic/gin"

// RegisterRoutes must be attached under a group that already carries
// RequireAuth — every handler re-derives the actor via middleware.CurrentUser
// and every mutation is additionally scope-checked in service.go.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	schools := rg.Group("/schools")
	schools.GET("", h.ListSchools)
	schools.GET("/:id", h.GetSchool)
	schools.POST("", h.CreateSchool)
	schools.PUT("/:id", h.UpdateSchool)
	schools.DELETE("/:id", h.DeleteSchool)

	departments := rg.Group("/departments")
	departments.GET("", h.ListDepartments)
	departments.GET("/:id", h.GetDepartment)
	departments.POST("", h.CreateDepartment)
	departments.PUT("/:id", h.UpdateDepartment)
	departments.DELETE("/:id", h.DeleteDepartment)

	courses := rg.Group("/courses")
	courses.GET("", h.ListCourses)
	courses.GET("/:id", h.GetCourse)
	courses.POST("", h.CreateCourse)
	courses.PUT("/:id", h.UpdateCourse)
	courses.DELETE("/:id", h.DeleteCourse)

	companies := rg.Group("/companies")
	companies.GET("", h.ListCompanies)
	companies.GET("/:id", h.GetCompany)
	companies.POST("", h.CreateCompany)
	companies.PUT("/:id", h.UpdateCompany)
	companies.DELETE("/:id", h.DeleteCompany)
}
