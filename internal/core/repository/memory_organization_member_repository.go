package repository

import (
	"fmt"
	"sync"
	"tracking/internal/core/model"
)

type inMemoryOrganizationMemberRepository struct {
	members map[string]*model.OrganizationMember
	mutex   sync.RWMutex
}

func NewInMemoryOrganizationMemberRepository() OrganizationMemberRepository {
	return &inMemoryOrganizationMemberRepository{
		members: make(map[string]*model.OrganizationMember),
	}
}

func (r *inMemoryOrganizationMemberRepository) Create(member *model.OrganizationMember) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	key := fmt.Sprintf("%s:%s", member.UserID, member.OrganizationID)
	if _, exists := r.members[key]; exists {
		return fmt.Errorf("member already exists")
	}

	r.members[key] = member
	return nil
}

func (r *inMemoryOrganizationMemberRepository) Update(member *model.OrganizationMember) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	key := fmt.Sprintf("%s:%s", member.UserID, member.OrganizationID)
	if _, exists := r.members[key]; !exists {
		return fmt.Errorf("member not found")
	}

	r.members[key] = member
	return nil
}

func (r *inMemoryOrganizationMemberRepository) Delete(userID, orgID string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	key := fmt.Sprintf("%s:%s", userID, orgID)
	if _, exists := r.members[key]; !exists {
		return fmt.Errorf("member not found")
	}

	delete(r.members, key)
	return nil
}

func (r *inMemoryOrganizationMemberRepository) FindByUserAndOrg(userID, orgID string) (*model.OrganizationMember, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	key := fmt.Sprintf("%s:%s", userID, orgID)
	if member, exists := r.members[key]; exists {
		return member, nil
	}
	return nil, nil
}

func (r *inMemoryOrganizationMemberRepository) FindByUser(userID string) ([]*model.OrganizationMember, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var result []*model.OrganizationMember
	for _, member := range r.members {
		if member.UserID == userID {
			result = append(result, member)
		}
	}
	return result, nil
}