package types

import (
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RoleSystemAdmin         Role = "SYSTEM_ADMIN"
	RoleSchoolDirector      Role = "SCHOOL_DIRECTOR"
	RoleFinanceOfficer      Role = "FINANCE_OFFICER"
	RoleHeadTeacher         Role = "HEAD_TEACHER"
	RoleDeputyHeadTeacher   Role = "DEPUTY_HEAD_TEACHER"
	RoleAcademicCoordinator Role = "ACADEMIC_COORDINATOR"
	RoleClassTeacher        Role = "CLASS_TEACHER"
	RoleSubjectTeacher      Role = "SUBJECT_TEACHER"
	RoleLibrarian           Role = "LIBRARIAN"
	RoleTransportCoordinator Role = "TRANSPORT_COORDINATOR"
	RoleAdmissionsOfficer   Role = "ADMISSIONS_OFFICER"
	RoleReceptionist        Role = "RECEPTIONIST"
	RoleParent              Role = "PARENT"
	RoleStudent             Role = "STUDENT"
)

type PortalType string

const (
	PortalAdmin   PortalType = "admin"
	PortalFinance PortalType = "finance"
	PortalTeacher PortalType = "teacher"
	PortalParent  PortalType = "parent"
	PortalStudent PortalType = "student"
)

var AllRoles = []Role{
	RoleSystemAdmin, RoleSchoolDirector, RoleFinanceOfficer, RoleHeadTeacher,
	RoleDeputyHeadTeacher, RoleAcademicCoordinator, RoleClassTeacher, RoleSubjectTeacher,
	RoleLibrarian, RoleTransportCoordinator, RoleAdmissionsOfficer, RoleReceptionist,
	RoleParent, RoleStudent,
}

var NovaRoles = []Role{
	RoleClassTeacher, RoleSubjectTeacher, RoleHeadTeacher, RoleDeputyHeadTeacher,
	RoleAcademicCoordinator, RoleStudent, RoleSystemAdmin, RoleParent,
}

var AdminRoles = []Role{
	RoleSystemAdmin, RoleSchoolDirector, RoleHeadTeacher, RoleDeputyHeadTeacher,
	RoleAcademicCoordinator, RoleAdmissionsOfficer, RoleReceptionist,
}

var TeacherRoles = []Role{
	RoleClassTeacher, RoleSubjectTeacher, RoleHeadTeacher, RoleDeputyHeadTeacher,
	RoleAcademicCoordinator,
}

var FinanceRoles = []Role{
	RoleFinanceOfficer, RoleSystemAdmin, RoleSchoolDirector,
}

func RoleToPortal(r Role) PortalType {
	isAdmin := false
	for _, ar := range AdminRoles {
		if ar == r {
			isAdmin = true
			break
		}
	}
	if isAdmin {
		return PortalAdmin
	}
	for _, fr := range FinanceRoles {
		if fr == r {
			return PortalFinance
		}
	}
	for _, tr := range TeacherRoles {
		if tr == r {
			return PortalTeacher
		}
	}
	switch r {
	case RoleParent:
		return PortalParent
	case RoleStudent:
		return PortalStudent
	}
	return PortalAdmin
}

func IsValidRole(r Role) bool {
	for _, v := range AllRoles {
		if v == r {
			return true
		}
	}
	return false
}

type User struct {
	ID           uuid.UUID  `json:"id"`
	Email        string     `json:"email"`
	FirstName    string     `json:"first_name"`
	LastName     string     `json:"last_name"`
	Role         Role       `json:"role"`
	Portal       PortalType `json:"portal"`
	IsActive     bool       `json:"is_active"`
	PhoneNumber  string     `json:"phone_number,omitempty"`
	PasswordHash string     `json:"-"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type AuthClaims struct {
	UserID uuid.UUID  `json:"user_id"`
	Role   Role       `json:"role"`
	Portal PortalType `json:"portal"`
	Scope  string     `json:"scope"`
	Exp    int64      `json:"exp"`
	Iat    int64      `json:"iat"`
	Jti    string     `json:"jti"`
}

type PaginationParams struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

func (p PaginationParams) Offset() int {
	page := p.Page
	if page < 1 {
		page = 1
	}
	size := p.PageSize
	if size < 1 {
		size = 25
	}
	return (page - 1) * size
}

func (p PaginationParams) Limit() int {
	size := p.PageSize
	if size < 1 {
		size = 25
	}
	if size > 200 {
		size = 200
	}
	return size
}

type PaginatedResponse[T any] struct {
	Data       []T   `json:"data"`
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalCount int64 `json:"total_count"`
	TotalPages int   `json:"total_pages"`
}

func NewPaginatedResponse[T any](data []T, page, pageSize int, total int64) PaginatedResponse[T] {
	tp := PaginatedResponse[T]{
		Data:       data,
		Page:       page,
		PageSize:   pageSize,
		TotalCount: total,
	}
	if pageSize > 0 {
		tp.TotalPages = int((total + int64(pageSize) - 1) / int64(pageSize))
	}
	return tp
}
