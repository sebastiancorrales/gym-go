package handlers

import (
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/internal/infrastructure/http/dto"
	"github.com/sebastiancorrales/gym-go/internal/usecases"
	"net/http"

	"github.com/gin-gonic/gin"
)

// MemberHandler maneja las peticiones HTTP relacionadas con miembros
// Sigue el principio de Responsabilidad Única (SRP)
type MemberHandler struct {
	memberUseCase *usecases.MemberUseCase
}

// NewMemberHandler crea una nueva instancia de MemberHandler
func NewMemberHandler(memberUseCase *usecases.MemberUseCase) *MemberHandler {
	return &MemberHandler{
		memberUseCase: memberUseCase,
	}
}

// CreateMember maneja la creación de un nuevo miembro
func (h *MemberHandler) CreateMember(c *gin.Context) {
	var req dto.CreateMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Bad Request",
			Message: "Solicitud inválida",
			Details: map[string]string{"detail": err.Error()},
		})
		return
	}

	member := &entities.Member{
		FirstName:   req.FirstName,
		LastName:    req.LastName,
		Email:       req.Email,
		Phone:       req.Phone,
		DateOfBirth: req.DateOfBirth,
	}

	if err := h.memberUseCase.CreateMember(c.Request.Context(), member); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Internal Server Error",
			Message: "Error al crear miembro",
			Details: map[string]string{"detail": err.Error()},
		})
		return
	}

	response := mapMemberToResponse(member)
	c.JSON(http.StatusCreated, response)
}

// GetMember obtiene un miembro por su ID
func (h *MemberHandler) GetMember(c *gin.Context) {
	id := c.Param("id")

	member, err := h.memberUseCase.GetMemberByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Error:   "Not Found",
			Message: "Miembro no encontrado",
			Details: map[string]string{"detail": err.Error()},
		})
		return
	}

	response := mapMemberToResponse(member)
	c.JSON(http.StatusOK, response)
}

// UpdateMember actualiza la información de un miembro
func (h *MemberHandler) UpdateMember(c *gin.Context) {
	id := c.Param("id")

	var req dto.UpdateMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Bad Request",
			Message: "Solicitud inválida",
			Details: map[string]string{"detail": err.Error()},
		})
		return
	}

	// Obtener miembro existente
	member, err := h.memberUseCase.GetMemberByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Error:   "Not Found",
			Message: "Miembro no encontrado",
			Details: map[string]string{"detail": err.Error()},
		})
		return
	}

	// Actualizar campos
	if req.FirstName != "" {
		member.FirstName = req.FirstName
	}
	if req.LastName != "" {
		member.LastName = req.LastName
	}
	if req.Phone != "" {
		member.Phone = req.Phone
	}

	if err := h.memberUseCase.UpdateMember(c.Request.Context(), member); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Internal Server Error",
			Message: "Error al actualizar miembro",
			Details: map[string]string{"detail": err.Error()},
		})
		return
	}

	response := mapMemberToResponse(member)
	c.JSON(http.StatusOK, response)
}

// AssignMembership asigna una membresía a un miembro
func (h *MemberHandler) AssignMembership(c *gin.Context) {
	memberID := c.Param("id")

	var req dto.AssignMembershipRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Bad Request",
			Message: "Solicitud inválida",
			Details: map[string]string{"detail": err.Error()},
		})
		return
	}

	if err := h.memberUseCase.AssignMembership(c.Request.Context(), memberID, req.MembershipID); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Internal Server Error",
			Message: "Error al asignar membresía",
			Details: map[string]string{"detail": err.Error()},
		})
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "Membresía asignada exitosamente",
	})
}

// SuspendMember suspende un miembro
func (h *MemberHandler) SuspendMember(c *gin.Context) {
	id := c.Param("id")

	if err := h.memberUseCase.SuspendMember(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Internal Server Error",
			Message: "Error al suspender miembro",
			Details: map[string]string{"detail": err.Error()},
		})
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "Miembro suspendido exitosamente",
	})
}

// ActivateMember activa un miembro
func (h *MemberHandler) ActivateMember(c *gin.Context) {
	id := c.Param("id")

	if err := h.memberUseCase.ActivateMember(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Internal Server Error",
			Message: "Error al activar miembro",
			Details: map[string]string{"detail": err.Error()},
		})
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Success: true,
		Message: "Miembro activado exitosamente",
	})
}

// mapMemberToResponse convierte una entidad Member a MemberResponse
func mapMemberToResponse(member *entities.Member) *dto.MemberResponse {
	return &dto.MemberResponse{
		ID:           member.ID,
		FirstName:    member.FirstName,
		LastName:     member.LastName,
		Email:        member.Email,
		Phone:        member.Phone,
		DateOfBirth:  member.DateOfBirth,
		JoinDate:     member.JoinDate,
		Status:       string(member.Status),
		MembershipID: member.MembershipID,
		CreatedAt:    member.CreatedAt,
		UpdatedAt:    member.UpdatedAt,
	}
}



